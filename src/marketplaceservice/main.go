package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var files embed.FS

type app struct {
	store     *Store
	authSlots chan struct{}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func send(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
func fail(w http.ResponseWriter, err error) {
	var e *apiError
	if errors.As(err, &e) {
		send(w, e.Code, map[string]string{"error": e.Message})
		return
	}
	log.Printf("request failed: %v", err)
	send(w, 500, map[string]string{"error": "Internal server error"})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 3*1024*1024)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return problem(400, "Invalid JSON request")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return problem(400, "Request must contain one JSON object")
	}
	return nil
}
func (a *app) user(r *http.Request) (string, error) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if len(token) != 48 {
		return "", problem(401, "Sign in to continue")
	}
	var user string
	err := a.store.pool.QueryRow(r.Context(), `SELECT user_id FROM marketplace_sessions WHERE token_hash=$1 AND expires_at>now()`, hashToken(token)).Scan(&user)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", problem(401, "Session expired or invalid")
	}
	return user, err
}
func (a *app) auth(w http.ResponseWriter, r *http.Request) {
	select {
	case a.authSlots <- struct{}{}:
		defer func() { <-a.authSlots }()
	default:
		fail(w, problem(429, "Please try again shortly"))
		return
	}
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := decode(w, r, &input); err != nil {
		fail(w, err)
		return
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	parsed, err := mail.ParseAddress(input.Email)
	if err != nil || parsed.Address != input.Email || len(input.Email) > 254 || len(input.Password) < 10 || len(input.Password) > 72 {
		fail(w, problem(400, "Use a valid email and a password of 10–72 bytes"))
		return
	}
	var user string
	displayName := strings.TrimSpace(input.Name)
	if !validText(displayName, 0, 80) {
		fail(w, problem(400, "Display name must be at most 80 characters"))
		return
	}
	if displayName == "" {
		displayName = strings.Split(input.Email, "@")[0]
	}
	code := 200
	if r.PathValue("action") == "register" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(w, err)
			return
		}
		user = newID()
		_, err = a.store.pool.Exec(r.Context(), `INSERT INTO marketplace_users(id,email,password_hash,display_name) VALUES($1,$2,$3,$4)`, user, input.Email, string(hash), displayName)
		if err != nil {
			fail(w, databaseError(err))
			return
		}
		code = 201
	} else if r.PathValue("action") == "login" {
		var hash string
		err = a.store.pool.QueryRow(r.Context(), `SELECT id,password_hash,display_name FROM marketplace_users WHERE email=$1`, input.Email).Scan(&user, &hash, &displayName)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, problem(401, "Invalid email or password"))
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
			fail(w, problem(401, "Invalid email or password"))
			return
		}
	} else {
		fail(w, problem(404, "Not found"))
		return
	}
	token := newID()
	_, err = a.store.pool.Exec(r.Context(), `INSERT INTO marketplace_sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '24 hours')`, hashToken(token), user)
	if err != nil {
		fail(w, err)
		return
	}
	send(w, code, map[string]string{"token": token, "user_id": user, "email": input.Email, "name": displayName})
}
func validText(s string, min, max int) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) >= min && utf8.RuneCountInString(s) <= max && !strings.ContainsRune(s, 0)
}
func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /marketplace", func(w http.ResponseWriter, r *http.Request) {
		target := os.Getenv("FRONTEND_URL")
		if target == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := a.store.pool.Ping(r.Context()); err != nil {
			send(w, 503, map[string]string{"status": "unavailable"})
			return
		}
		send(w, 200, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/marketplace/auth/{action}", a.auth)
	mux.HandleFunc("POST /api/marketplace/logout", func(w http.ResponseWriter, r *http.Request) {
		if _, err := a.user(r); err != nil {
			fail(w, err)
			return
		}
		_, err := a.store.pool.Exec(r.Context(), `DELETE FROM marketplace_sessions WHERE token_hash=$1`, hashToken(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")))
		if err != nil {
			fail(w, err)
			return
		}
		send(w, 200, map[string]bool{"ok": true})
	})
	mux.HandleFunc("GET /api/marketplace/me", func(w http.ResponseWriter, r *http.Request) {
		u, err := a.user(r)
		if err != nil {
			fail(w, err)
			return
		}
		var email, name string
		if err := a.store.pool.QueryRow(r.Context(), `SELECT email,display_name FROM marketplace_users WHERE id=$1`, u).Scan(&email, &name); err != nil {
			fail(w, err)
			return
		}
		send(w, 200, map[string]string{"user_id": u, "email": email, "name": name})
	})
	mux.HandleFunc("GET /api/marketplace/listings", func(w http.ResponseWriter, r *http.Request) {
		ls, err := a.store.listings(r.Context())
		if err != nil {
			fail(w, err)
			return
		}
		for i := range ls {
			ls[i] = publicListing(ls[i])
		}
		send(w, 200, ls)
	})
	mux.HandleFunc("GET /api/marketplace/listings/{id}/image", func(w http.ResponseWriter, r *http.Request) {
		l, err := a.store.listing(r.Context(), r.PathValue("id"))
		if err != nil {
			fail(w, err)
			return
		}
		raw := l.Metadata["image"]
		parts := strings.SplitN(raw, ",", 2)
		if len(parts) != 2 || !strings.HasPrefix(raw, "data:image/") {
			http.NotFound(w, r)
			return
		}
		data, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", http.DetectContentType(data))
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(data)
	})
	mux.HandleFunc("GET /api/marketplace/listings/{id}", func(w http.ResponseWriter, r *http.Request) {
		l, err := a.store.listing(r.Context(), r.PathValue("id"))
		if err != nil {
			fail(w, err)
			return
		}
		send(w, 200, publicListing(l))
	})
	mux.HandleFunc("POST /api/marketplace/listings", func(w http.ResponseWriter, r *http.Request) {
		u, err := a.user(r)
		if err != nil {
			fail(w, err)
			return
		}
		var input struct {
			Title       string            `json:"title"`
			Description string            `json:"description"`
			PriceCents  *int              `json:"price_cents"`
			Pickup      string            `json:"pickup"`
			Metadata    map[string]string `json:"metadata"`
		}
		if err = decode(w, r, &input); err != nil {
			fail(w, err)
			return
		}
		input.Title = strings.TrimSpace(input.Title)
		input.Description = strings.TrimSpace(input.Description)
		input.Pickup = strings.TrimSpace(input.Pickup)
		if !validText(input.Title, 1, 120) || !validText(input.Description, 1, 2000) || !validText(input.Pickup, 1, 200) || input.PriceCents == nil || *input.PriceCents < 0 || *input.PriceCents > 100000000 {
			fail(w, problem(400, "Check title, description, pickup location and price"))
			return
		}
		if err := validateMetadata(input.Metadata); err != nil {
			fail(w, err)
			return
		}
		l, err := a.store.createListing(r.Context(), u, Listing{Title: input.Title, Description: input.Description, PriceCents: *input.PriceCents, Pickup: input.Pickup, Metadata: input.Metadata})
		if err != nil {
			fail(w, err)
			return
		}
		send(w, 201, publicListing(l))
	})
	mux.HandleFunc("POST /api/marketplace/listings/{id}/reserve", func(w http.ResponseWriter, r *http.Request) {
		u, err := a.user(r)
		if err != nil {
			fail(w, err)
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if !validText(key, 1, 128) {
			fail(w, problem(400, "An Idempotency-Key of 1–128 characters is required"))
			return
		}
		res, replay, err := a.store.reserve(r.Context(), u, r.PathValue("id"), key)
		if err != nil {
			fail(w, err)
			return
		}
		code := 201
		if replay {
			code = 200
		}
		send(w, code, res)
	})
	mux.HandleFunc("GET /api/marketplace/reservations", func(w http.ResponseWriter, r *http.Request) {
		u, err := a.user(r)
		if err != nil {
			fail(w, err)
			return
		}
		out, err := a.store.reservations(r.Context(), u)
		if err != nil {
			fail(w, err)
			return
		}
		send(w, 200, out)
	})
	mux.HandleFunc("POST /api/marketplace/reservations/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		u, err := a.user(r)
		if err != nil {
			fail(w, err)
			return
		}
		action := r.PathValue("action")
		if action != "cancel" && action != "complete" {
			fail(w, problem(404, "Not found"))
			return
		}
		res, err := a.store.transition(r.Context(), u, r.PathValue("id"), action)
		if err != nil {
			fail(w, err)
			return
		}
		send(w, 200, res)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}
func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(748201905)"); err != nil {
		return err
	}
	schema, _ := files.ReadFile("schema.sql")
	if _, err = tx.Exec(ctx, string(schema)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	startup, cancel := context.WithTimeout(ctx, 15*time.Second)
	err = migrate(startup, pool)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8090"
	}
	a := &app{&Store{pool}, make(chan struct{}, 4)}
	srv := &http.Server{Addr: addr, Handler: a.routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	log.Printf("CampusLoop marketplace listening on %s", addr)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func validateMetadata(metadata map[string]string) error {
	allowed := map[string]int{"category": 40, "city": 100, "campus": 150, "dorm": 150, "address": 300, "contact": 200, "handoff": 200, "condition": 50}
	for key, value := range metadata {
		if key == "image" {
			if value == "" {
				continue
			}
			if strings.HasPrefix(value, "/static/") && path.Clean(value) == value && !strings.ContainsAny(value, "?#\\") {
				continue
			}
			parts := strings.SplitN(value, ",", 2)
			if len(parts) != 2 {
				return problem(400, "Choose a PNG, JPEG, GIF or WebP image")
			}
			allowedTypes := map[string]string{"data:image/png;base64": "image/png", "data:image/jpeg;base64": "image/jpeg", "data:image/gif;base64": "image/gif", "data:image/webp;base64": "image/webp"}
			mime, ok := allowedTypes[parts[0]]
			if !ok {
				return problem(400, "Unsupported image type")
			}
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil || len(data) > 2*1024*1024 || http.DetectContentType(data) != mime {
				return problem(400, "Use a valid image no larger than 2 MB")
			}
			continue
		}
		max, ok := allowed[key]
		if !ok || !validText(value, 0, max) {
			return problem(400, "Invalid listing details")
		}
	}
	return nil
}

// Keep uploaded image bytes out of list/detail JSON and serve them on demand.
func publicListing(l Listing) Listing {
	metadata := make(map[string]string, len(l.Metadata))
	for key, value := range l.Metadata {
		metadata[key] = value
	}
	if strings.HasPrefix(metadata["image"], "data:") {
		metadata["image"] = "/api/marketplace/listings/" + l.ID + "/image"
	}
	l.Metadata = metadata
	return l
}
