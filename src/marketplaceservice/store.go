package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type apiError struct {
	Code    int
	Message string
}

func (e *apiError) Error() string        { return e.Message }
func problem(code int, msg string) error { return &apiError{code, msg} }
func newID() string {
	var b [24]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func databaseError(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) && pe.Code == "23505" {
		return problem(409, "Conflicting or duplicate request")
	}
	return err
}

type Listing struct {
	Metadata    map[string]string `json:"metadata"`
	ID          string            `json:"id"`
	SellerID    string            `json:"seller_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	PriceCents  int               `json:"price_cents"`
	Pickup      string            `json:"pickup"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
}
type Reservation struct {
	ID        string `json:"id"`
	ListingID string `json:"listing_id"`
	BuyerID   string `json:"buyer_id"`
	Status    string `json:"status"`
}
type Store struct{ pool *pgxpool.Pool }

func (s *Store) createListing(ctx context.Context, user string, l Listing) (Listing, error) {
	if l.Metadata == nil {
		l.Metadata = map[string]string{}
	}
	l.ID = newID()
	l.SellerID = user
	l.Status = "available"
	err := s.pool.QueryRow(ctx, `INSERT INTO marketplace_listings(id,seller_id,title,description,price_cents,pickup,metadata) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING created_at`, l.ID, user, l.Title, l.Description, l.PriceCents, l.Pickup, l.Metadata).Scan(&l.CreatedAt)
	return l, err
}
func (s *Store) listings(ctx context.Context) ([]Listing, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,seller_id,title,description,price_cents,pickup,status,created_at,metadata FROM marketplace_listings ORDER BY created_at DESC,id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Listing{}
	for rows.Next() {
		var l Listing
		if err := rows.Scan(&l.ID, &l.SellerID, &l.Title, &l.Description, &l.PriceCents, &l.Pickup, &l.Status, &l.CreatedAt, &l.Metadata); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

// Lock order is listing then reservation for every state transition. The partial
// unique index is a second line of defense, including writes outside this service.
func (s *Store) reserve(ctx context.Context, user, listing, key string) (Reservation, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, false, err
	}
	defer tx.Rollback(ctx)
	var seller, state string
	err = tx.QueryRow(ctx, `SELECT seller_id,status FROM marketplace_listings WHERE id=$1 FOR UPDATE`, listing).Scan(&seller, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, false, problem(404, "Listing not found")
	}
	if err != nil {
		return Reservation{}, false, err
	}
	var r Reservation
	err = tx.QueryRow(ctx, `SELECT id,listing_id,buyer_id,status FROM marketplace_reservations WHERE buyer_id=$1 AND idempotency_key=$2`, user, key).Scan(&r.ID, &r.ListingID, &r.BuyerID, &r.Status)
	if err == nil {
		if r.ListingID != listing {
			return r, false, problem(409, "Idempotency key belongs to another listing")
		}
		return r, true, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return r, false, err
	}
	if seller == user {
		return r, false, problem(403, "You cannot reserve your own listing")
	}
	if state != "available" {
		return r, false, problem(409, "Listing is no longer available")
	}
	r = Reservation{newID(), listing, user, "active"}
	_, err = tx.Exec(ctx, `INSERT INTO marketplace_reservations(id,listing_id,buyer_id,idempotency_key) VALUES($1,$2,$3,$4)`, r.ID, listing, user, key)
	if err != nil {
		return r, false, databaseError(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE marketplace_listings SET status='reserved' WHERE id=$1`, listing); err != nil {
		return r, false, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO marketplace_reservation_events(reservation_id,actor_id,action) VALUES($1,$2,'reserved')`, r.ID, user); err != nil {
		return r, false, err
	}
	return r, false, tx.Commit(ctx)
}
func (s *Store) transition(ctx context.Context, user, id, action string) (Reservation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Reservation{}, err
	}
	defer tx.Rollback(ctx)
	var listing string
	err = tx.QueryRow(ctx, `SELECT listing_id FROM marketplace_reservations WHERE id=$1`, id).Scan(&listing)
	if errors.Is(err, pgx.ErrNoRows) {
		return Reservation{}, problem(404, "Reservation not found")
	}
	if err != nil {
		return Reservation{}, err
	}
	var seller string
	if err = tx.QueryRow(ctx, `SELECT seller_id FROM marketplace_listings WHERE id=$1 FOR UPDATE`, listing).Scan(&seller); err != nil {
		return Reservation{}, err
	}
	var r Reservation
	if err = tx.QueryRow(ctx, `SELECT id,listing_id,buyer_id,status FROM marketplace_reservations WHERE id=$1 FOR UPDATE`, id).Scan(&r.ID, &r.ListingID, &r.BuyerID, &r.Status); err != nil {
		return r, err
	}
	next, listingState := "cancelled", "available"
	if action == "complete" {
		next, listingState = "completed", "sold"
		if seller != user {
			return r, problem(403, "Only the seller can complete a handoff")
		}
	} else if r.BuyerID != user {
		return r, problem(403, "Only the buyer can cancel this reservation")
	}
	if r.Status == next {
		return r, tx.Commit(ctx)
	}
	if r.Status != "active" {
		return r, problem(409, "Reservation has already ended")
	}
	if _, err = tx.Exec(ctx, `UPDATE marketplace_reservations SET status=$2,updated_at=now() WHERE id=$1`, id, next); err != nil {
		return r, err
	}
	if _, err = tx.Exec(ctx, `UPDATE marketplace_listings SET status=$2 WHERE id=$1`, listing, listingState); err != nil {
		return r, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO marketplace_reservation_events(reservation_id,actor_id,action) VALUES($1,$2,$3)`, id, user, next); err != nil {
		return r, err
	}
	r.Status = next
	return r, tx.Commit(ctx)
}
func (s *Store) reservations(ctx context.Context, user string) ([]Reservation, error) {
	rows, err := s.pool.Query(ctx, `SELECT r.id,r.listing_id,r.buyer_id,r.status FROM marketplace_reservations r JOIN marketplace_listings l ON l.id=r.listing_id WHERE r.buyer_id=$1 OR l.seller_id=$1 ORDER BY r.created_at DESC LIMIT 100`, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Reservation{}
	for rows.Next() {
		var r Reservation
		if err = rows.Scan(&r.ID, &r.ListingID, &r.BuyerID, &r.Status); err != nil {
			return nil, fmt.Errorf("reservation: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) listing(ctx context.Context, id string) (Listing, error) {
	var l Listing
	err := s.pool.QueryRow(ctx, `SELECT id,seller_id,title,description,price_cents,pickup,status,created_at,metadata FROM marketplace_listings WHERE id=$1`, id).Scan(&l.ID, &l.SellerID, &l.Title, &l.Description, &l.PriceCents, &l.Pickup, &l.Status, &l.CreatedAt, &l.Metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, problem(404, "Listing not found")
	}
	return l, err
}
