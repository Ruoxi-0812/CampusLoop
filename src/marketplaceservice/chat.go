package main

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

type conversation struct {
	ID        string `json:"id"`
	ListingID string `json:"listing_id"`
	Title     string `json:"title"`
	OtherName string `json:"other_name"`
}
type chatMessage struct {
	ID        int64     `json:"id"`
	SenderID  string    `json:"sender_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (a *app) chatRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/marketplace/listings/{id}/conversation", a.startConversation)
	mux.HandleFunc("GET /api/marketplace/conversations", a.conversations)
	mux.HandleFunc("GET /api/marketplace/conversations/{id}/messages", a.messages)
	mux.HandleFunc("POST /api/marketplace/conversations/{id}/messages", a.messages)
}
func (a *app) startConversation(w http.ResponseWriter, r *http.Request) {
	user, err := a.user(r)
	if err != nil {
		fail(w, err)
		return
	}
	var seller string
	err = a.store.pool.QueryRow(r.Context(), `SELECT seller_id FROM marketplace_listings WHERE id=$1`, r.PathValue("id")).Scan(&seller)
	if errors.Is(err, pgx.ErrNoRows) {
		fail(w, problem(404, "Item not found"))
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	if seller == user {
		fail(w, problem(400, "Open Messages to reply to buyers"))
		return
	}
	if seller == "campusloop-demo-seller" {
		fail(w, problem(400, "Sample items use demo chat"))
		return
	}
	var id string
	err = a.store.pool.QueryRow(r.Context(), `INSERT INTO marketplace_conversations(id,listing_id,buyer_id,seller_id) VALUES($1,$2,$3,$4) ON CONFLICT(listing_id,buyer_id) DO UPDATE SET listing_id=EXCLUDED.listing_id RETURNING id`, newID(), r.PathValue("id"), user, seller).Scan(&id)
	if err != nil {
		fail(w, err)
		return
	}
	send(w, 200, map[string]string{"id": id})
}
func (a *app) conversations(w http.ResponseWriter, r *http.Request) {
	user, err := a.user(r)
	if err != nil {
		fail(w, err)
		return
	}
	rows, err := a.store.pool.Query(r.Context(), `SELECT c.id,c.listing_id,l.title,COALESCE(NULLIF(u.display_name,''),'Student') FROM marketplace_conversations c JOIN marketplace_listings l ON l.id=c.listing_id JOIN marketplace_users u ON u.id=CASE WHEN c.buyer_id=$1 THEN c.seller_id ELSE c.buyer_id END WHERE c.buyer_id=$1 OR c.seller_id=$1 ORDER BY COALESCE((SELECT max(m.created_at) FROM marketplace_messages m WHERE m.conversation_id=c.id),c.created_at) DESC,c.id DESC LIMIT 100`, user)
	if err != nil {
		fail(w, err)
		return
	}
	defer rows.Close()
	result := []conversation{}
	for rows.Next() {
		var c conversation
		if err = rows.Scan(&c.ID, &c.ListingID, &c.Title, &c.OtherName); err != nil {
			fail(w, err)
			return
		}
		result = append(result, c)
	}
	if err = rows.Err(); err != nil {
		fail(w, err)
		return
	}
	send(w, 200, result)
}
func (a *app) messages(w http.ResponseWriter, r *http.Request) {
	user, err := a.user(r)
	if err != nil {
		fail(w, err)
		return
	}
	id := r.PathValue("id")
	var allowed bool
	err = a.store.pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM marketplace_conversations WHERE id=$1 AND (buyer_id=$2 OR seller_id=$2))`, id, user).Scan(&allowed)
	if err != nil {
		fail(w, err)
		return
	}
	if !allowed {
		fail(w, problem(404, "Conversation not found"))
		return
	}
	if r.Method == http.MethodPost {
		var input struct {
			Body     string `json:"body"`
			ClientID string `json:"client_id"`
		}
		if err = decode(w, r, &input); err != nil {
			fail(w, err)
			return
		}
		input.Body = strings.TrimSpace(input.Body)
		if !validText(input.Body, 1, 2000) || !validText(input.ClientID, 1, 128) {
			fail(w, problem(400, "Message must contain 1–2000 characters and a request ID"))
			return
		}
		var m chatMessage
		err = a.store.pool.QueryRow(r.Context(), `INSERT INTO marketplace_messages(conversation_id,sender_id,client_id,body) VALUES($1,$2,$3,$4) ON CONFLICT(conversation_id,sender_id,client_id) DO UPDATE SET client_id=EXCLUDED.client_id RETURNING id,sender_id,body,created_at`, id, user, input.ClientID, input.Body).Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt)
		if err != nil {
			fail(w, err)
			return
		}
		if m.Body != input.Body {
			fail(w, problem(409, "Request ID already used for another message"))
			return
		}
		send(w, 200, m)
		return
	}
	rows, err := a.store.pool.Query(r.Context(), `SELECT id,sender_id,body,created_at FROM (SELECT id,sender_id,body,created_at FROM marketplace_messages WHERE conversation_id=$1 ORDER BY id DESC LIMIT 100) latest ORDER BY id`, id)
	if err != nil {
		fail(w, err)
		return
	}
	defer rows.Close()
	result := []chatMessage{}
	for rows.Next() {
		var m chatMessage
		if err = rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt); err != nil {
			fail(w, err)
			return
		}
		result = append(result, m)
	}
	if err = rows.Err(); err != nil {
		fail(w, err)
		return
	}
	send(w, 200, result)
}
