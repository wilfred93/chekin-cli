package chekin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestQueryDefaultsToAllDates(t *testing.T) {
	q, err := ListOptions{}.Query()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ordering":               "check_in_date,check_out_date,created_at",
		"is_housing_deactivated": "true",
		"check_in_date_from":     "2000-01-01T00:00:00",
	}
	for k, v := range want {
		if got := q.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if q.Has("check_in_date_until") {
		t.Errorf("unexpected check_in_date_until")
	}
}

func TestQueryFilters(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	q, err := ListOptions{
		CheckInFrom: d("2026-01-01"), CheckInTo: d("2026-01-31"),
		CheckOutFrom: d("2026-02-01"), CheckOutTo: d("2026-02-28"),
		Status: "keys-generated", Property: "Playa", Guest: "Doe",
		ExternalID: "HM12", Source: "Airbnb", Batch: "7", Descending: true, PageSize: 50,
		Extra: url.Values{"foo": {"bar"}},
	}.Query()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ordering":              "-check_in_date,check_out_date,created_at",
		"check_in_date_from":    "2026-01-01T00:00:00",
		"check_in_date_until":   "2026-01-31T23:59:59",
		"check_out_date_after":  "2026-02-01T00:00:00",
		"check_out_date_before": "2026-02-28T23:59:59",
		"remote_access_status":  "GENERATED,DISPLAYED",
		"name":                  "Playa",
		"guest_name":            "Doe",
		"external_id_contains":  "HM12",
		"source_name":           "Airbnb",
		"lote":                  "7",
		"page_size":             "50",
		"foo":                   "bar",
	}
	for k, v := range want {
		if got := q.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestQueryExactPropertyWins(t *testing.T) {
	q, _ := ListOptions{Property: "a", PropertyExact: "b"}.Query()
	if q.Get("exact_name") != "b" || q.Has("name") {
		t.Errorf("got %v", q)
	}
}

func TestQueryUnknownStatus(t *testing.T) {
	if _, err := (ListOptions{Status: "nope"}).Query(); err == nil {
		t.Fatal("expected error")
	}
}

// fakeAPI serves `total` bookings in pages of 2, with absolute http:// next
// links like the real API.
func fakeAPI(t *testing.T, total int, token string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/status/reservations/" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "JWT "+token || r.Header.Get("X-Source") != "DASHBOARD" {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"code":4002,"message":"Invalid token.","status_code":401}`)
			return
		}
		pageNum := 1
		fmt.Sscan(r.URL.Query().Get("page"), &pageNum)
		var results []map[string]any
		for i := (pageNum - 1) * 2; i < total && i < pageNum*2; i++ {
			results = append(results, map[string]any{"id": fmt.Sprintf("b%d", i), "check_in_date": "2026-01-01T00:00:00Z", "extra_field": i})
		}
		var next any
		if pageNum*2 < total {
			q := r.URL.Query()
			q.Set("page", fmt.Sprint(pageNum+1))
			next = srv.URL + r.URL.Path + "?" + q.Encode()
		}
		json.NewEncoder(w).Encode(map[string]any{"count": 999999999, "next": next, "previous": nil, "results": results})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListFollowsPagination(t *testing.T) {
	srv := fakeAPI(t, 5, "tok")
	got, err := New(srv.URL, "tok").ListReservations(context.Background(), ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 || got[4].ID != "b4" {
		t.Fatalf("got %d bookings", len(got))
	}
	if !strings.Contains(string(got[0].Raw), "extra_field") {
		t.Errorf("raw JSON lost fields: %s", got[0].Raw)
	}
}

func TestListLimit(t *testing.T) {
	srv := fakeAPI(t, 5, "tok")
	got, err := New(srv.URL, "tok").ListReservations(context.Background(), ListOptions{Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d bookings, want 3", len(got))
	}
}

func TestListUnauthorized(t *testing.T) {
	srv := fakeAPI(t, 1, "tok")
	_, err := New(srv.URL, "wrong").ListReservations(context.Background(), ListOptions{})
	if !IsUnauthorized(err) {
		t.Fatalf("want unauthorized, got %v", err)
	}
	if !strings.Contains(err.Error(), "Invalid token.") {
		t.Errorf("message not surfaced: %v", err)
	}
}

func TestRefusesNextLinkToOtherHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"next":"https://evil.example.com/steal","results":[{"id":"a"}]}`)
	}))
	defer srv.Close()
	_, err := New(srv.URL, "tok").ListReservations(context.Background(), ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "unexpected host") {
		t.Fatalf("want host error, got %v", err)
	}
}

func TestLoginErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprint(w, `{"non_field_errors":["Unable to log in with provided credentials."]}`)
	}))
	defer srv.Close()
	_, err := New(srv.URL, "").Login(context.Background(), "a@b.c", "x")
	if err == nil || !strings.Contains(err.Error(), "Unable to log in") {
		t.Fatalf("got %v", err)
	}
}

func TestParseClaims(t *testing.T) {
	tok := MakeTestToken("me@example.com", time.Unix(1893456000, 0))
	c, err := ParseClaims(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Username != "me@example.com" || c.Exp != 1893456000 {
		t.Errorf("got %+v", c)
	}
	if _, err := ParseClaims("nope"); err == nil {
		t.Error("expected error for non-JWT")
	}
}
