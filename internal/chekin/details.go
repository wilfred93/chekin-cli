package chekin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
)

// DashboardURL is where a booking opens in the Chekin web dashboard.
func DashboardURL(id string) string {
	return "https://dashboard.chekin.com/bookings/" + url.PathEscape(id)
}

// ReservationDetail is one booking from the dashboard's booking page. It has
// many more fields than the list, including the guest links. Raw keeps the
// full object.
type ReservationDetail struct {
	ID               string          `json:"id"`
	CheckInDate      string          `json:"check_in_date"`
	CheckOutDate     string          `json:"check_out_date"`
	NightsOfStay     int             `json:"nights_of_stay"`
	HousingID        string          `json:"housing_id"`
	Status           string          `json:"status"`
	AggregatedStatus string          `json:"aggregated_status"`
	SourceName       string          `json:"source_name"`
	ExternalID       string          `json:"external_id"`
	BookingReference string          `json:"booking_reference"`
	LeaderName       string          `json:"default_leader_full_name"`
	Price            any             `json:"price"`
	Currency         string          `json:"currency"`
	SignupFormLink   string          `json:"signup_form_link"` // guest online check-in
	PaymentLink      string          `json:"payment_link"`
	UpsellingLink    string          `json:"upselling_link"`
	Raw              json.RawMessage `json:"-"`
}

// GetReservation fetches one booking by ID.
func (c *Client) GetReservation(ctx context.Context, id string) (*ReservationDetail, error) {
	if id == "" || strings.ContainsAny(id, "/?#") {
		return nil, fmt.Errorf("invalid booking ID %q", id)
	}
	var raw json.RawMessage
	if err := c.do(ctx, "GET", "api/v3/light/reservations/"+url.PathEscape(id)+"/", nil, &raw); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return nil, fmt.Errorf("booking %s not found", id)
		}
		return nil, err
	}
	var d ReservationDetail
	if err := json.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("unexpected booking format: %w", err)
	}
	d.Raw = raw
	return &d, nil
}

// GetReservations fetches several bookings, a few at a time, keeping order.
// It stops at the first error.
func (c *Client) GetReservations(ctx context.Context, ids []string) ([]*ReservationDetail, error) {
	const workers = 5
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make([]*ReservationDetail, len(ids))
	jobs := make(chan int)
	var (
		wg       sync.WaitGroup
		once     sync.Once
		firstErr error
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				d, err := c.GetReservation(ctx, ids[i])
				if err != nil {
					once.Do(func() { firstErr = err; cancel() })
					continue
				}
				out[i] = d
			}
		}()
	}
	for i := range ids {
		select {
		case jobs <- i:
		case <-ctx.Done():
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, ctx.Err()
}
