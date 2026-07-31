package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Ensures the public 2-segment GET patterns that previously panicked at
// ServeMux registration (early-slot-offers/{token} vs {slug}/catalog|slots)
// are expressed as a single overlapping-safe pattern.
func TestPublicTwoSegmentGETRoutesRegisterWithoutPanic(t *testing.T) {
	mux := http.NewServeMux()
	var hit string
	mux.HandleFunc("GET /api/v1/public/{seg1}/{seg2}", func(w http.ResponseWriter, r *http.Request) {
		seg1 := r.PathValue("seg1")
		seg2 := r.PathValue("seg2")
		switch {
		case seg1 == "early-slot-offers":
			hit = "offer:" + seg2
		case seg2 == "catalog":
			hit = "catalog:" + seg1
		case seg2 == "slots":
			hit = "slots:" + seg1
		default:
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, hit)
	})
	mux.HandleFunc("POST /api/v1/public/early-slot-offers/{token}/accept", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("POST /api/v1/public/early-slot-offers/{token}/decline", func(http.ResponseWriter, *http.Request) {})
	mux.Handle("POST /api/v1/public/{slug}/appointments", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	mux.HandleFunc("GET /api/v1/public/appointments/manage/{token}", func(http.ResponseWriter, *http.Request) {})

	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/public/early-slot-offers/tok-1", "offer:tok-1"},
		{"/api/v1/public/meu-salao/catalog", "catalog:meu-salao"},
		{"/api/v1/public/meu-salao/slots", "slots:meu-salao"},
		// Former conflict path: must resolve as early-slot offer, not catalog.
		{"/api/v1/public/early-slot-offers/catalog", "offer:catalog"},
	}
	for _, tc := range cases {
		hit = ""
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%q", tc.path, rec.Code, rec.Body.String())
		}
		if hit != tc.want {
			t.Fatalf("%s: hit=%q want=%q", tc.path, hit, tc.want)
		}
	}
}
