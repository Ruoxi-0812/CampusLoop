package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestMarketplaceProxyPreservesAuthAndIdempotency(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/marketplace/listings/example/reserve" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Idempotency-Key") != "retry-key" {
			t.Errorf("proxy changed request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(409)
		io.WriteString(w, `{"error":"Listing is no longer available"}`)
	}))
	defer upstream.Close()
	router := mux.NewRouter()
	if err := registerMarketplaceRoutes(router, "/campus", upstream.URL); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/campus/api/marketplace/listings/example/reserve", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Idempotency-Key", "retry-key")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != 409 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestMarketplaceProxyConfiguration(t *testing.T) {
	for _, target := range []string{"file:///tmp/test", "http://", "http://example.test/path", "http://user:pass@example.test"} {
		if err := registerMarketplaceRoutes(mux.NewRouter(), "", target); err == nil {
			t.Errorf("accepted %q", target)
		}
	}
	router := mux.NewRouter()
	if err := registerMarketplaceRoutes(router, "", ""); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/marketplace", nil))
	if response.Code != 404 {
		t.Fatalf("disabled proxy registered a route: %d", response.Code)
	}
}

func TestMarketplaceUsesExistingPages(t *testing.T) {
	t.Setenv("MARKETPLACE_SERVICE_URL", "http://example.test")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		listing := `{"id":"item-one","title":"Shared textbook","description":"A persistent listing","price_cents":1250,"pickup":"Library","status":"available","seller_id":"seller","metadata":{"category":"textbooks"}}`
		if r.URL.Path == "/api/marketplace/listings" {
			io.WriteString(w, "["+listing+"]")
		} else if r.URL.Path == "/api/marketplace/listings/item-one" {
			io.WriteString(w, listing)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	r := mux.NewRouter()
	if err := registerMarketplaceRoutes(r, "", upstream.URL); err != nil {
		t.Fatal(err)
	}
	for path, marker := range map[string]string{"/": "campusloop-listings-grid", "/login": "name=\"password\"", "/post-item": "campusloop-post-form", "/my-listings": "data-reservation-list", "/product/item-one": "data-reserve-listing=\"item-one\""} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), marker) || !strings.Contains(w.Body.String(), "CampusLoopBackendConfig") {
			t.Errorf("%s did not render original integrated page: %d %s", path, w.Code, w.Body.String())
		}
		if path == "/product/item-one" && strings.Contains(w.Body.String(), "Pay online, then coordinate pickup.") {
			t.Error("real listing retained mock checkout")
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/marketplace", nil))
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/" {
		t.Fatal("standalone page still exposed")
	}
}
