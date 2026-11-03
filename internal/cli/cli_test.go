package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wilfred93/chekin-cli/internal/chekin"
	"github.com/wilfred93/chekin-cli/internal/config"
)

type fakeServer struct {
	*httptest.Server
	valid     atomic.Value // the token the server accepts
	logins    atomic.Int32
	refreshes atomic.Int32
	lastQuery atomic.Value
}

func newFake(t *testing.T) *fakeServer {
	t.Helper()
	f := &fakeServer{}
	f.valid.Store("")
	f.lastQuery.Store("")
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/token/":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["email"] != "me@example.com" || body["password"] != "secret" {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"non_field_errors":["Unable to log in with provided credentials."]}`)
				return
			}
			f.logins.Add(1)
			tok := chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))
			f.valid.Store(tok)
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
		case "/api/v3/api-token-refresh/":
			f.refreshes.Add(1)
			tok := chekin.MakeTestToken("me@example.com", time.Now().Add(2*time.Hour))
			f.valid.Store(tok)
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
		case "/api/v4/status/reservations/":
			if r.Header.Get("Authorization") != "JWT "+f.valid.Load().(string) {
				w.WriteHeader(401)
				fmt.Fprint(w, `{"message":"Invalid token.","status_code":401}`)
				return
			}
			f.lastQuery.Store(r.URL.RawQuery)
			fmt.Fprint(w, `{"count":999999999,"next":null,"results":[
				{"id":"abc123","check_in_date":"2026-11-07T00:00:00Z","check_out_date":"2026-11-14T00:00:00Z",
				 "housing_name":"Sea view flat","guest_leader_name":"","guests":"0/2","general_status":"INCOMPLETE","source_name":"Airbnb"}]}`)
		case "/api/v3/light/reservations/abc123/":
			if r.Header.Get("Authorization") != "JWT "+f.valid.Load().(string) {
				w.WriteHeader(401)
				return
			}
			fmt.Fprint(w, `{"id":"abc123","check_in_date":"2026-11-07T00:00:00Z","check_out_date":"2026-11-14T00:00:00Z",
				"nights_of_stay":7,"status":"NEW","source_name":"Airbnb","price":"420.00","currency":"EUR",
				"signup_form_link":"https://guest.chekin.com/CHECKIN","payment_link":"https://guest.chekin.com/PAY",
				"upselling_link":"https://guest.chekin.com/UPSELL","tax":null}`)
		default:
			w.WriteHeader(404)
			fmt.Fprint(w, `{"detail":"Not found."}`)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func setup(t *testing.T) *fakeServer {
	f := newFake(t)
	t.Setenv("CHEKIN_API_URL", f.URL)
	t.Setenv("CHEKIN_CONFIG_DIR", t.TempDir())
	t.Setenv("CHEKIN_TOKEN", "")
	t.Setenv("CHEKIN_EMAIL", "")
	t.Setenv("CHEKIN_PASSWORD", "")
	return f
}

func run(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	app := &App{Stdin: strings.NewReader(stdin), Stdout: &out, Stderr: &errb}
	code = app.Run(context.Background(), args)
	return code, out.String(), errb.String()
}

func TestLoginThenList(t *testing.T) {
	setup(t)
	if code, _, stderr := run(t, "secret\n", "login", "--email", "me@example.com", "--password-stdin"); code != 0 {
		t.Fatalf("login failed: %s", stderr)
	}
	creds, _ := config.Load()
	if creds == nil || creds.Email != "me@example.com" || creds.Token == "" {
		t.Fatalf("credentials not saved: %+v", creds)
	}

	code, out, stderr := run(t, "", "bookings", "list")
	if code != 0 {
		t.Fatalf("list failed: %s", stderr)
	}
	for _, want := range []string{"ID", "abc123", "2026-11-07", "Sea view flat", "INCOMPLETE", "Airbnb"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}
}

func TestWrongPassword(t *testing.T) {
	setup(t)
	code, _, stderr := run(t, "nope\n", "login", "--email", "me@example.com", "--password-stdin")
	if code != 1 || !strings.Contains(stderr, "Unable to log in") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestLoginWithPastedLocalStorageJSON(t *testing.T) {
	f := setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))
	f.valid.Store(tok)
	if code, _, stderr := run(t, `{"token":"`+tok+`"}`, "login", "--token", "-"); code != 0 {
		t.Fatalf("login --token failed: %s", stderr)
	}
	code, out, _ := run(t, "", "whoami")
	if code != 0 || !strings.Contains(out, "me@example.com") {
		t.Fatalf("whoami: %d %s", code, out)
	}
}

func TestRejectsExpiredToken(t *testing.T) {
	setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(-time.Minute))
	code, _, stderr := run(t, "", "login", "--token", tok)
	if code != 1 || !strings.Contains(stderr, "expired") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestRefreshesTokenAboutToExpire(t *testing.T) {
	f := setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(2*time.Minute))
	config.Save(&config.Credentials{Token: tok})
	f.valid.Store(tok)

	if code, _, stderr := run(t, "", "bookings", "list"); code != 0 {
		t.Fatalf("list failed: %s", stderr)
	}
	if f.refreshes.Load() != 1 {
		t.Fatalf("refreshes = %d, want 1", f.refreshes.Load())
	}
	creds, _ := config.Load()
	if creds.Token == tok {
		t.Error("refreshed token was not saved")
	}
}

func TestEnvCredentialsLogInAutomatically(t *testing.T) {
	f := setup(t)
	t.Setenv("CHEKIN_EMAIL", "me@example.com")
	t.Setenv("CHEKIN_PASSWORD", "secret")
	if code, _, stderr := run(t, "", "bookings", "list", "-o", "json"); code != 0 {
		t.Fatalf("list failed: %s", stderr)
	}
	if f.logins.Load() != 1 {
		t.Fatalf("logins = %d", f.logins.Load())
	}
}

func TestRejectedTokenRetriesWithEnvLogin(t *testing.T) {
	f := setup(t)
	config.Save(&config.Credentials{Token: chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))})
	t.Setenv("CHEKIN_EMAIL", "me@example.com")
	t.Setenv("CHEKIN_PASSWORD", "secret")
	if code, _, stderr := run(t, "", "bookings", "list"); code != 0 {
		t.Fatalf("list failed: %s", stderr)
	}
	if f.logins.Load() != 1 {
		t.Fatalf("logins = %d, want 1", f.logins.Load())
	}
}

func TestNotLoggedIn(t *testing.T) {
	setup(t)
	code, _, stderr := run(t, "", "bookings", "list")
	if code != 1 || !strings.Contains(stderr, "not logged in") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
}

func TestJSONOutputKeepsRawFields(t *testing.T) {
	f := setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))
	f.valid.Store(tok)
	t.Setenv("CHEKIN_TOKEN", tok)
	code, out, stderr := run(t, "", "reservations", "list", "-o", "json")
	if code != 0 {
		t.Fatalf("failed: %s", stderr)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got) != 1 || got[0]["guests"] != "0/2" {
		t.Fatalf("bad JSON (%v): %s", err, out)
	}
}

func TestFlagsBecomeQuery(t *testing.T) {
	f := setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))
	f.valid.Store(tok)
	t.Setenv("CHEKIN_TOKEN", tok)
	code, _, stderr := run(t, "", "bookings", "list", "--from", "2026-01-01", "--to", "2026-03-31",
		"--status", "unpaid", "--source", "Booking.com", "--guest", "Doe", "--desc", "--param", "x=1", "-o", "jsonl")
	if code != 0 {
		t.Fatalf("failed: %s", stderr)
	}
	q := f.lastQuery.Load().(string)
	for _, want := range []string{"check_in_date_from=2026-01-01T00%3A00%3A00", "check_in_date_until=2026-03-31T23%3A59%3A59",
		"booking_paid=PENDING", "source_name=Booking.com", "guest_name=Doe", "ordering=-check_in_date", "x=1"} {
		if !strings.Contains(q, want) {
			t.Errorf("query missing %q: %s", want, q)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	setup(t)
	for _, args := range [][]string{
		{},
		{"nope"},
		{"bookings"},
		{"bookings", "list", "--status", "bogus"},
		{"bookings", "list", "--from", "01/02/2026"},
		{"bookings", "list", "--upcoming", "--past"},
		{"bookings", "list", "-o", "xml"},
	} {
		if code, _, _ := run(t, "", args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

func loggedIn(t *testing.T) *fakeServer {
	f := setup(t)
	tok := chekin.MakeTestToken("me@example.com", time.Now().Add(time.Hour))
	f.valid.Store(tok)
	t.Setenv("CHEKIN_TOKEN", tok)
	return f
}

func TestGetShowsLinks(t *testing.T) {
	loggedIn(t)
	code, out, stderr := run(t, "", "bookings", "get", "abc123")
	if code != 0 {
		t.Fatalf("failed: %s", stderr)
	}
	for _, want := range []string{"Check-in link:", "https://guest.chekin.com/CHECKIN", "https://guest.chekin.com/PAY",
		"https://guest.chekin.com/UPSELL", "https://dashboard.chekin.com/bookings/abc123", "420.00 EUR", "Nights:"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
}

func TestGetJSON(t *testing.T) {
	loggedIn(t)
	code, out, stderr := run(t, "", "bookings", "get", "-o", "json", "abc123")
	if code != 0 {
		t.Fatalf("failed: %s", stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if got["signup_form_link"] != "https://guest.chekin.com/CHECKIN" || got["dashboard_url"] != "https://dashboard.chekin.com/bookings/abc123" {
		t.Errorf("got %v", got)
	}
	if _, ok := got["tax"]; !ok {
		t.Error("raw fields dropped")
	}
}

func TestGetErrors(t *testing.T) {
	loggedIn(t)
	if code, _, stderr := run(t, "", "bookings", "get", "nope"); code != 1 || !strings.Contains(stderr, "not found") {
		t.Errorf("not found: code=%d %s", code, stderr)
	}
	if code, _, _ := run(t, "", "bookings", "get"); code != 2 {
		t.Errorf("missing id: code=%d", code)
	}
	if code, _, _ := run(t, "", "bookings", "get", "a/b"); code != 1 {
		t.Errorf("bad id: code=%d", code)
	}
}

func TestListWithLinks(t *testing.T) {
	loggedIn(t)
	code, out, stderr := run(t, "", "bookings", "list", "--links")
	if code != 0 {
		t.Fatalf("failed: %s", stderr)
	}
	if !strings.Contains(out, "CHECK-IN LINK") || !strings.Contains(out, "https://guest.chekin.com/CHECKIN") {
		t.Errorf("table missing links:\n%s", out)
	}

	code, out, _ = run(t, "", "bookings", "list", "--links", "-o", "jsonl")
	var got map[string]any
	if code != 0 || json.Unmarshal([]byte(out), &got) != nil {
		t.Fatalf("bad jsonl: %s", out)
	}
	if got["payment_link"] != "https://guest.chekin.com/PAY" || got["guests"] != "0/2" {
		t.Errorf("got %v", got)
	}
}
