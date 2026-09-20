package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	store   *Store
	servers [2]*httptest.Server
	config  *pgxpool.Config
}

func setup(t *testing.T) *fixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required for real PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + newID()
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	other, err := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{store: &Store{pool}, config: cfg}
	f.servers[0] = httptest.NewServer((&app{f.store, make(chan struct{}, 4)}).routes())
	f.servers[1] = httptest.NewServer((&app{&Store{other}, make(chan struct{}, 4)}).routes())
	t.Cleanup(func() {
		for _, s := range f.servers {
			s.Close()
		}
		pool.Close()
		other.Close()
		admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return f
}
func (f *fixture) account(t *testing.T) (string, string) {
	t.Helper()
	id, token := newID(), newID()
	_, err := f.store.pool.Exec(context.Background(), `INSERT INTO marketplace_users(id,email,password_hash) VALUES($1,$2,'test-fixture')`, id, id+"@example.test")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.store.pool.Exec(context.Background(), `INSERT INTO marketplace_sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, hashToken(token), id)
	if err != nil {
		t.Fatal(err)
	}
	return id, token
}
func request(t *testing.T, base, method, path, token, key string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Error(err)
			return 0, nil
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, base+"/api/marketplace"+path, reader)
	if err != nil {
		t.Error(err)
		return 0, nil
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
		return 0, nil
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	return resp.StatusCode, data
}
func expect(t *testing.T, got, want int, body []byte) {
	t.Helper()
	if got != want {
		t.Fatalf("status %d, expected %d: %s", got, want, body)
	}
}
func (f *fixture) listing(t *testing.T, token string) Listing {
	t.Helper()
	code, data := request(t, f.servers[0].URL, "POST", "/listings", token, "", map[string]any{"title": "Calculus textbook", "description": "Clean pages, second edition", "price_cents": 2500, "pickup": "Snell Library lobby"})
	expect(t, code, 201, data)
	var l Listing
	if err := json.Unmarshal(data, &l); err != nil {
		t.Fatal(err)
	}
	return l
}
func parseReservation(t *testing.T, b []byte) Reservation {
	t.Helper()
	var r Reservation
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestConcurrentReservationAcrossInstances(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	tokens := make([]string, 100)
	for i := range tokens {
		_, tokens[i] = f.account(t)
	}
	successes, conflicts := 0, 0
	started := time.Now()
	for round := 0; round < 10; round++ {
		l := f.listing(t, seller)
		start := make(chan struct{})
		codes := make([]int, 100)
		var wg sync.WaitGroup
		for i := range tokens {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				codes[i], _ = request(t, f.servers[i%2].URL, "POST", "/listings/"+l.ID+"/reserve", tokens[i], fmt.Sprintf("round-%d-buyer-%d", round, i), nil)
			}(i)
		}
		close(start)
		wg.Wait()
		won, lost := 0, 0
		for _, code := range codes {
			if code == 201 {
				won++
			} else if code == 409 {
				lost++
			} else {
				t.Errorf("unexpected HTTP status: %d", code)
			}
		}
		if won != 1 || lost != 99 {
			t.Fatalf("round %d: %d winners, %d conflicts", round, won, lost)
		}
		successes += won
		conflicts += lost
		var active, events int
		var state string
		f.store.pool.QueryRow(context.Background(), `SELECT count(*) FROM marketplace_reservations WHERE listing_id=$1 AND status='active'`, l.ID).Scan(&active)
		f.store.pool.QueryRow(context.Background(), `SELECT count(*) FROM marketplace_reservation_events e JOIN marketplace_reservations r ON r.id=e.reservation_id WHERE r.listing_id=$1`, l.ID).Scan(&events)
		f.store.pool.QueryRow(context.Background(), `SELECT status FROM marketplace_listings WHERE id=$1`, l.ID).Scan(&state)
		if active != 1 || events != 1 || state != "reserved" {
			t.Fatalf("database invariant broken: active=%d events=%d state=%s", active, events, state)
		}
	}
	report := map[string]any{"test": "100 distinct authenticated buyers contend for one listing, repeated for 10 distinct listings", "instances": 2, "independent_connection_pools": true, "requests": 1000, "created": successes, "expected_conflicts": conflicts, "duplicate_active_reservations": 0, "rounds": 10, "concurrency": 100, "elapsed_seconds": time.Since(started).Seconds(), "timestamp": time.Now().UTC().Format(time.RFC3339)}
	data, _ := json.MarshalIndent(report, "", "  ")
	t.Log(string(data))
	if path := os.Getenv("MARKETPLACE_TEST_REPORT"); path != "" {
		if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIdempotencyOwnershipAndHandoff(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	buyer, buyerToken := f.account(t)
	_, other := f.account(t)
	l := f.listing(t, seller)
	base := f.servers[0].URL
	path := "/listings/" + l.ID + "/reserve"
	code, b := request(t, base, "POST", path, "", "key", nil)
	expect(t, code, 401, b)
	code, b = request(t, base, "POST", path, seller, "self", nil)
	expect(t, code, 403, b)
	code, b = request(t, base, "POST", path, buyerToken, "", nil)
	expect(t, code, 400, b)
	var wg sync.WaitGroup
	ids := make([]string, 30)
	codes := make([]int, 30)
	start := make(chan struct{})
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			c, data := request(t, f.servers[i%2].URL, "POST", path, buyerToken, "same-key", nil)
			codes[i] = c
			var r Reservation
			json.Unmarshal(data, &r)
			ids[i] = r.ID
		}(i)
	}
	close(start)
	wg.Wait()
	created := 0
	for i, c := range codes {
		if c == 201 {
			created++
		} else if c != 200 {
			t.Fatalf("retry status %d", c)
		}
		if ids[i] == "" || ids[i] != ids[0] {
			t.Fatal("retry returned different reservation")
		}
	}
	if created != 1 {
		t.Fatalf("created %d reservations", created)
	}
	id := ids[0]
	code, b = request(t, base, "POST", "/reservations/"+id+"/cancel", other, "", nil)
	expect(t, code, 403, b)
	code, b = request(t, base, "POST", "/reservations/"+id+"/complete", buyerToken, "", nil)
	expect(t, code, 403, b)
	// Database-level backstop rejects active duplicates even outside the API.
	_, err := f.store.pool.Exec(context.Background(), `INSERT INTO marketplace_reservations(id,listing_id,buyer_id,idempotency_key) VALUES($1,$2,$3,'bypass')`, newID(), l.ID, buyer)
	if err == nil {
		t.Fatal("database accepted a second active reservation")
	}
	code, b = request(t, base, "POST", "/reservations/"+id+"/cancel", buyerToken, "", nil)
	expect(t, code, 200, b)
	code, b = request(t, base, "POST", path, buyerToken, "same-key", nil)
	expect(t, code, 200, b)
	if parseReservation(t, b).Status != "cancelled" {
		t.Fatal("retry resurrected a reservation")
	}
	code, b = request(t, base, "POST", path, other, "second-buyer", nil)
	expect(t, code, 201, b)
	next := parseReservation(t, b)
	// Replaying cancellation must not release the newer buyer's reservation.
	code, b = request(t, base, "POST", "/reservations/"+id+"/cancel", buyerToken, "", nil)
	expect(t, code, 200, b)
	code, b = request(t, base, "POST", path, buyerToken, "third", nil)
	expect(t, code, 409, b)
	code, b = request(t, base, "POST", "/reservations/"+next.ID+"/complete", seller, "", nil)
	expect(t, code, 200, b)
	code, b = request(t, base, "POST", "/reservations/"+next.ID+"/cancel", other, "", nil)
	expect(t, code, 409, b)
	code, b = request(t, base, "POST", path, buyerToken, "after-sold", nil)
	expect(t, code, 409, b)
	l2 := f.listing(t, seller)
	code, b = request(t, base, "POST", "/listings/"+l2.ID+"/reserve", buyerToken, "same-key", nil)
	expect(t, code, 409, b)
	// Reopen a separate service/pool: state is in PostgreSQL, not process memory.
	pool, err := pgxpool.NewWithConfig(context.Background(), f.config.Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	restarted := httptest.NewServer((&app{&Store{pool}, make(chan struct{}, 4)}).routes())
	defer restarted.Close()
	code, b = request(t, restarted.URL, "GET", "/listings", "", "", nil)
	expect(t, code, 200, b)
	var ls []Listing
	json.Unmarshal(b, &ls)
	found := false
	for _, x := range ls {
		if x.ID == l.ID {
			found = x.Status == "sold"
		}
	}
	if !found {
		t.Fatal("state not persisted")
	}
	code, b = request(t, restarted.URL, "POST", path, buyerToken, "same-key", nil)
	expect(t, code, 200, b)
	if parseReservation(t, b).ID != id {
		t.Fatal("idempotency not persisted")
	}
}

func TestTransactionRollback(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	_, buyer := f.account(t)
	l := f.listing(t, seller)
	_, err := f.store.pool.Exec(context.Background(), `CREATE FUNCTION reject_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected audit failure'; END $$; CREATE TRIGGER reject_event BEFORE INSERT ON marketplace_reservation_events FOR EACH ROW EXECUTE FUNCTION reject_event()`)
	if err != nil {
		t.Fatal(err)
	}
	code, b := request(t, f.servers[0].URL, "POST", "/listings/"+l.ID+"/reserve", buyer, "rollback", nil)
	expect(t, code, 500, b)
	var count int
	var state string
	f.store.pool.QueryRow(context.Background(), `SELECT count(*) FROM marketplace_reservations`).Scan(&count)
	f.store.pool.QueryRow(context.Background(), `SELECT status FROM marketplace_listings WHERE id=$1`, l.ID).Scan(&state)
	if count != 0 || state != "available" {
		t.Fatalf("partial write: %d %s", count, state)
	}
	if _, err = f.store.pool.Exec(context.Background(), `DROP TRIGGER reject_event ON marketplace_reservation_events`); err != nil {
		t.Fatal(err)
	}
	code, b = request(t, f.servers[0].URL, "POST", "/listings/"+l.ID+"/reserve", buyer, "rollback", nil)
	expect(t, code, 201, b)
}

func TestAuthenticationAndInputValidation(t *testing.T) {
	f := setup(t)
	base := f.servers[0].URL
	credentials := map[string]string{"email": "student@example.test", "password": "a-long-password"}
	code, b := request(t, base, "POST", "/auth/register", "", "", credentials)
	expect(t, code, 201, b)
	var auth map[string]string
	json.Unmarshal(b, &auth)
	token := auth["token"]
	code, b = request(t, base, "POST", "/auth/register", "", "", credentials)
	expect(t, code, 409, b)
	code, b = request(t, base, "POST", "/auth/login", "", "", map[string]string{"email": "student@example.test", "password": "wrong-password"})
	expect(t, code, 401, b)
	code, b = request(t, base, "POST", "/auth/login", "", "", credentials)
	expect(t, code, 200, b)
	for _, body := range []map[string]any{
		{"title": "", "description": "ok", "pickup": "library", "price_cents": 100},
		{"title": "Test", "description": "ok", "pickup": "library", "price_cents": -1},
		{"title": "Test", "description": "ok", "pickup": "library", "price_cents": 1.5},
		{"title": "Test", "description": "ok", "pickup": "library", "price_cents": 100, "seller_id": "spoofed"},
		{"title": "Test", "description": "ok", "pickup": "library"},
	} {
		code, b = request(t, base, "POST", "/listings", token, "", body)
		expect(t, code, 400, b)
	}
	code, b = request(t, base, "POST", "/logout", token, "", nil)
	expect(t, code, 200, b)
	code, b = request(t, base, "POST", "/listings", token, "", map[string]any{})
	expect(t, code, 401, b)
}

func TestConcurrentCancelAndComplete(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	_, buyer := f.account(t)
	for round := 0; round < 10; round++ {
		l := f.listing(t, seller)
		code, data := request(t, f.servers[0].URL, "POST", "/listings/"+l.ID+"/reserve", buyer, fmt.Sprintf("transition-%d", round), nil)
		expect(t, code, 201, data)
		r := parseReservation(t, data)
		var wg sync.WaitGroup
		start := make(chan struct{})
		codes := make([]int, 2)
		for i, action := range []string{"cancel", "complete"} {
			wg.Add(1)
			go func(i int, action string) {
				defer wg.Done()
				<-start
				token := buyer
				if action == "complete" {
					token = seller
				}
				codes[i], _ = request(t, f.servers[i].URL, "POST", "/reservations/"+r.ID+"/"+action, token, "", nil)
			}(i, action)
		}
		close(start)
		wg.Wait()
		if !((codes[0] == 200 && codes[1] == 409) || (codes[0] == 409 && codes[1] == 200)) {
			t.Fatalf("incompatible transitions both applied: %v", codes)
		}
		var state, reservationState string
		var events int
		err := f.store.pool.QueryRow(context.Background(), `SELECT l.status,r.status FROM marketplace_listings l JOIN marketplace_reservations r ON r.listing_id=l.id WHERE r.id=$1`, r.ID).Scan(&state, &reservationState)
		if err != nil {
			t.Fatal(err)
		}
		if (reservationState == "completed" && state != "sold") || (reservationState == "cancelled" && state != "available") {
			t.Fatalf("inconsistent states: %s %s", state, reservationState)
		}
		if err = f.store.pool.QueryRow(context.Background(), `SELECT count(*) FROM marketplace_reservation_events WHERE reservation_id=$1`, r.ID).Scan(&events); err != nil {
			t.Fatal(err)
		}
		if events != 2 {
			t.Fatalf("duplicate state event: %d", events)
		}
	}
}

func TestListingDetailsAndImageValidation(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	input := map[string]any{"title": "Photo listing", "description": "Original form fields", "price_cents": 1234, "pickup": "Snell", "metadata": map[string]string{"category": "textbooks", "campus": "Boston", "image": "/static/img/products/campusloop/used-calculus-textbook.png", "handoff": "After class", "contact": "Test seller"}}
	code, b := request(t, f.servers[0].URL, "POST", "/listings", seller, "", input)
	expect(t, code, 201, b)
	var created Listing
	if err := json.Unmarshal(b, &created); err != nil {
		t.Fatal(err)
	}
	code, b = request(t, f.servers[1].URL, "GET", "/listings/"+created.ID, "", "", nil)
	expect(t, code, 200, b)
	var fetched Listing
	json.Unmarshal(b, &fetched)
	if fetched.Metadata["category"] != "textbooks" || fetched.Metadata["handoff"] != "After class" || fetched.PriceCents != 1234 {
		t.Fatalf("details not persisted: %s", b)
	}
	input["metadata"] = map[string]string{"image": "data:image/svg+xml;base64,PHN2Zz4="}
	code, b = request(t, f.servers[0].URL, "POST", "/listings", seller, "", input)
	expect(t, code, 400, b)
	input["metadata"] = map[string]string{"image": "javascript:alert(1)"}
	code, b = request(t, f.servers[0].URL, "POST", "/listings", seller, "", input)
	expect(t, code, 400, b)
}

func TestUploadedImageRoundTrip(t *testing.T) {
	f := setup(t)
	_, seller := f.account(t)
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	input := map[string]any{"title": "Photo", "description": "Photo test", "price_cents": 100, "pickup": "Library", "metadata": map[string]string{"image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(imageData.Bytes())}}
	code, b := request(t, f.servers[0].URL, "POST", "/listings", seller, "", input)
	expect(t, code, 201, b)
	var l Listing
	if err := json.Unmarshal(b, &l); err != nil {
		t.Fatal(err)
	}
	if l.Metadata["image"] != "/api/marketplace/listings/"+l.ID+"/image" {
		t.Fatal("image bytes exposed in listing JSON")
	}
	code, b = request(t, f.servers[1].URL, "GET", "/listings/"+l.ID+"/image", "", "", nil)
	expect(t, code, 200, b)
	if !bytes.Equal(b, imageData.Bytes()) {
		t.Fatal("uploaded image did not persist")
	}
}
