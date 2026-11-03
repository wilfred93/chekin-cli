package chekin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Reservation is one booking as returned by the dashboard's bookings list.
// Raw keeps the full object so JSON output loses nothing.
type Reservation struct {
	ID                 string            `json:"id"`
	CheckInDate        string            `json:"check_in_date"`
	CheckOutDate       string            `json:"check_out_date"`
	HousingID          string            `json:"housing_id"`
	HousingName        string            `json:"housing_name"`
	HousingDisplayName string            `json:"housing_display_name"`
	GuestLeaderName    string            `json:"guest_leader_name"`
	Guests             string            `json:"guests"`
	GeneralStatus      string            `json:"general_status"`
	SourceName         string            `json:"source_name"`
	ExternalID         string            `json:"external_id"`
	DetailedStatuses   map[string]string `json:"detailed_statuses"`
	Raw                json.RawMessage   `json:"-"`
}

// Status filters, as offered by the dashboard's status dropdown. Each maps to
// one query parameter and value.
var StatusFilters = map[string][2]string{
	"complete":               {"complete", "COMPLETE"},
	"incomplete":             {"complete", "INCOMPLETE"},
	"unpaid":                 {"booking_paid", "PENDING"},
	"police-check-in-error":  {"police_check_in_status", "ERROR"},
	"police-check-out-error": {"police_check_out_status", "ERROR"},
	"stats-error":            {"stats", "ERROR"},
	"id-verification-failed": {"identity_verification", "ERROR"},
	"keys-generated":         {"remote_access_status", "GENERATED,DISPLAYED"},
	"keys-pending":           {"remote_access_status", "NOT_GENERATED"},
	"keys-error":             {"remote_access_status", "GENERATION_FAILED"},
}

// allTimeStart is sent as check_in_date_from when no start date is given.
// Without any date filter the API returns only upcoming bookings, and with
// only an end date it returns about one year back.
var allTimeStart = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// ListOptions are the bookings list filters. Dates are whole days; zero means
// no bound.
type ListOptions struct {
	CheckInFrom   time.Time
	CheckInTo     time.Time
	CheckOutFrom  time.Time
	CheckOutTo    time.Time
	Status        string // a key of StatusFilters
	Property      string // property name, partial match
	PropertyExact string // property name, exact match
	Guest         string // guest name
	ExternalID    string // partial match
	Source        string // e.g. "Airbnb", "Booking.com"
	Batch         string // batch number ("lote")
	Descending    bool   // newest check-in first
	PageSize      int    // 0 uses the API default (20)
	Limit         int    // stop after this many bookings; 0 means all
	Extra         url.Values
}

// Query builds the query string the dashboard sends to api/v4/status/reservations/.
func (o ListOptions) Query() (url.Values, error) {
	q := url.Values{}

	order := "check_in_date,check_out_date,created_at"
	if o.Descending {
		order = "-" + order
	}
	q.Set("ordering", order)
	// The dashboard always sends this; it includes bookings of deactivated properties.
	q.Set("is_housing_deactivated", "true")

	from := o.CheckInFrom
	if from.IsZero() {
		from = allTimeStart
	}
	q.Set("check_in_date_from", startOfDay(from))
	if !o.CheckInTo.IsZero() {
		q.Set("check_in_date_until", endOfDay(o.CheckInTo))
	}
	if !o.CheckOutFrom.IsZero() {
		q.Set("check_out_date_after", startOfDay(o.CheckOutFrom))
	}
	if !o.CheckOutTo.IsZero() {
		q.Set("check_out_date_before", endOfDay(o.CheckOutTo))
	}

	if o.Status != "" {
		f, ok := StatusFilters[o.Status]
		if !ok {
			return nil, fmt.Errorf("unknown status %q", o.Status)
		}
		q.Set(f[0], f[1])
	}
	if o.PropertyExact != "" {
		q.Set("exact_name", o.PropertyExact)
	} else if o.Property != "" {
		q.Set("name", o.Property)
	}
	setIf(q, "guest_name", o.Guest)
	setIf(q, "external_id_contains", o.ExternalID)
	setIf(q, "source_name", o.Source)
	setIf(q, "lote", o.Batch)
	if o.PageSize > 0 {
		q.Set("page_size", strconv.Itoa(o.PageSize))
	}
	for k, vs := range o.Extra {
		q[k] = vs
	}
	return q, nil
}

type page struct {
	Next    *string           `json:"next"`
	Results []json.RawMessage `json:"results"`
}

// ListReservations returns bookings matching opts, following pagination.
func (c *Client) ListReservations(ctx context.Context, opts ListOptions) ([]Reservation, error) {
	q, err := opts.Query()
	if err != nil {
		return nil, err
	}
	q.Set("page", "1")
	next := "api/v4/status/reservations/?" + q.Encode()

	var out []Reservation
	for pages := 0; next != ""; pages++ {
		if pages >= 1000 {
			return nil, fmt.Errorf("stopped after %d pages; the API kept returning more", pages)
		}
		var p page
		if err := c.do(ctx, "GET", next, nil, &p); err != nil {
			return nil, err
		}
		for _, raw := range p.Results {
			var r Reservation
			if err := json.Unmarshal(raw, &r); err != nil {
				return nil, fmt.Errorf("unexpected booking format: %w", err)
			}
			r.Raw = raw
			out = append(out, r)
			if opts.Limit > 0 && len(out) >= opts.Limit {
				return out, nil
			}
		}
		next = ""
		if p.Next != nil && len(p.Results) > 0 {
			next = *p.Next
		}
	}
	return out, nil
}

func setIf(q url.Values, key, value string) {
	if value != "" {
		q.Set(key, value)
	}
}

func startOfDay(t time.Time) string { return t.Format("2006-01-02") + "T00:00:00" }
func endOfDay(t time.Time) string   { return t.Format("2006-01-02") + "T23:59:59" }
