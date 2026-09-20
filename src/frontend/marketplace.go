package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	pb "github.com/GoogleCloudPlatform/microservices-demo/src/frontend/genproto"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Opt-in, same-origin gateway to the persistent resale workflow. The service
// verifies its own bearer sessions; browser demo identity is never trusted.
func registerMarketplaceRoutes(r *mux.Router, prefix, target string) error {
	if target == "" {
		return nil
	}
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("MARKETPLACE_SERVICE_URL must be an HTTP(S) origin")
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = 15 * time.Second
	proxy.Transport = transport
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Marketplace is temporarily unavailable", http.StatusBadGateway)
	}
	var handler http.Handler = proxy
	if prefix != "" {
		handler = http.StripPrefix(prefix, proxy)
	}
	r.HandleFunc(prefix+"/marketplace", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, prefix+"/", http.StatusSeeOther) })
	for path, name := range map[string]string{"/": "home", "/login": "signin", "/signin": "signin", "/signup": "signin", "/post-item": "post-item", "/my-listings": "my-listings", "/product/{id}": "product", "/messages": "messages"} {
		r.HandleFunc(prefix+path, marketplacePage(target, prefix, name)).Methods(http.MethodGet, http.MethodHead)
	}
	r.HandleFunc(prefix+"/cart", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, prefix+"/my-listings", http.StatusSeeOther)
	}).Methods(http.MethodGet)
	r.PathPrefix(prefix + "/api/marketplace/").Handler(handler)
	return nil
}

// Render the existing CampusLoop templates with persistent listing data.
type marketplaceListing struct {
	ID          string            `json:"id"`
	SellerID    string            `json:"seller_id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	PriceCents  int               `json:"price_cents"`
	Pickup      string            `json:"pickup"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata"`
}
type marketplaceProductView struct {
	Item   *pb.Product
	Price  *pb.Money
	Status string
	Pickup string
}

func listingView(l marketplaceListing, prefix string) marketplaceProductView {
	picture := l.Metadata["image"]
	if picture == "" {
		picture = "/static/icons/listing-no-photo.svg"
	}
	if strings.HasPrefix(picture, "data:") {
		picture = "/api/marketplace/listings/" + l.ID + "/image"
	}
	price := &pb.Money{CurrencyCode: "USD", Units: int64(l.PriceCents / 100), Nanos: int32(l.PriceCents%100) * 10000000}
	category := l.Metadata["category"]
	if category == "" {
		category = "other"
	}
	return marketplaceProductView{&pb.Product{Id: l.ID, Name: l.Title, Description: l.Description, Picture: picture, PriceUsd: price, Categories: []string{category}}, price, l.Status, l.Pickup}
}
func marketplacePage(target, prefix, name string) http.HandlerFunc {
	client := &http.Client{Timeout: 8 * time.Second}
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{"show_currency": false, "marketplaceEnabled": true, "cart_size": 0}
		fetch := func(path string, out any) bool {
			req, err := http.NewRequestWithContext(r.Context(), "GET", strings.TrimRight(target, "/")+path, nil)
			if err != nil {
				http.Error(w, "Marketplace unavailable", 502)
				return false
			}
			response, err := client.Do(req)
			if err != nil {
				http.Error(w, "Marketplace unavailable", 502)
				return false
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				code := 502
				if response.StatusCode == 404 {
					code = 404
				}
				http.Error(w, http.StatusText(code), code)
				return false
			}
			if err := json.NewDecoder(io.LimitReader(response.Body, 10*1024*1024)).Decode(out); err != nil {
				http.Error(w, "Could not load listings", 502)
				return false
			}
			return true
		}
		if name == "home" {
			var listings []marketplaceListing
			if !fetch("/api/marketplace/listings", &listings) {
				return
			}
			views := make([]marketplaceProductView, 0, len(listings))
			for _, l := range listings {
				views = append(views, listingView(l, prefix))
			}
			data["products"] = views
		}
		if name == "product" {
			var l marketplaceListing
			if !fetch("/api/marketplace/listings/"+url.PathEscape(mux.Vars(r)["id"]), &l) {
				return
			}
			data["product"] = listingView(l, prefix)
			data["listing"] = l
		}
		var output bytes.Buffer
		if err := templates.ExecuteTemplate(&output, name, injectCommonTemplateData(r, data)); err != nil {
			http.Error(w, "Could not render page", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(output.Bytes())
	}
}
