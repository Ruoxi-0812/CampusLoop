// Seed the existing CampusLoop catalog into the local persistent marketplace.
// Existing accounts/listings are never overwritten.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	file := flag.String("catalog", "../productcatalogservice/products.json", "original catalog JSON")
	flag.Parse()
	password := os.Getenv("SEED_PASSWORD")
	if len(password) < 10 || len(password) > 72 {
		log.Fatal("SEED_PASSWORD (10–72 bytes) is required for the local demo seller")
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatal(err)
	}
	var catalog struct {
		Products []struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Picture     string   `json:"picture"`
			Categories  []string `json:"categories"`
			Price       struct {
				Units int `json:"units"`
				Nanos int `json:"nanos"`
			} `json:"priceUsd"`
		} `json:"products"`
	}
	if err = json.Unmarshal(data, &catalog); err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(ctx)
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO marketplace_users(id,email,password_hash,display_name) VALUES('campusloop-demo-seller','catalog-demo@example.test',$1,'CampusLoop Demo Seller') ON CONFLICT(email) DO NOTHING`, string(hash))
	if err != nil {
		log.Fatal(err)
	}
	var seller string
	if err = tx.QueryRow(ctx, `SELECT id FROM marketplace_users WHERE email='catalog-demo@example.test'`).Scan(&seller); err != nil {
		log.Fatal(err)
	}
	created := int64(0)
	for _, p := range catalog.Products {
		category := "other"
		if len(p.Categories) > 0 {
			category = p.Categories[0]
		}
		metadata := map[string]string{"category": category, "image": p.Picture, "campus": "Northeastern University - Boston", "city": "Boston", "handoff": "Weekdays after 5pm", "contact": "Demo listing — no real seller contact", "address": "Meet at the public lobby"}
		tag, err := tx.Exec(ctx, `INSERT INTO marketplace_listings(id,seller_id,title,description,price_cents,pickup,metadata) VALUES($1,$2,$3,$4,$5,'Snell Library lobby',$6) ON CONFLICT(id) DO NOTHING`, p.ID, seller, p.Name, p.Description, p.Price.Units*100+p.Price.Nanos/10000000, metadata)
		if err != nil {
			log.Fatal(err)
		}
		created += tag.RowsAffected()
	}
	if err = tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}
	log.Printf("Imported %d original catalog listings; existing rows preserved", created)
}
