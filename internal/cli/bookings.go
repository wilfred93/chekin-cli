package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/wilfred93/chekin-cli/internal/chekin"
)

// paramFlag collects repeated --param key=value flags.
type paramFlag struct{ v url.Values }

func (p *paramFlag) String() string { return "" }
func (p *paramFlag) Set(s string) error {
	k, v, ok := strings.Cut(s, "=")
	if !ok || k == "" {
		return fmt.Errorf("expected key=value, got %q", s)
	}
	if p.v == nil {
		p.v = url.Values{}
	}
	p.v.Add(k, v)
	return nil
}

func statusNames() string {
	names := make([]string, 0, len(chekin.StatusFilters))
	for k := range chekin.StatusFilters {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func (a *App) bookingsList(ctx context.Context, args []string) error {
	fs := a.flags("bookings list", "chekin bookings list [flags]")
	from := fs.String("from", "", "check-in on or after `DATE` (YYYY-MM-DD or today)")
	to := fs.String("to", "", "check-in on or before `DATE`")
	checkoutFrom := fs.String("checkout-from", "", "check-out on or after `DATE`")
	checkoutTo := fs.String("checkout-to", "", "check-out on or before `DATE`")
	upcoming := fs.Bool("upcoming", false, "only bookings checking in today or later")
	past := fs.Bool("past", false, "only bookings that checked in before today")
	status := fs.String("status", "", "filter by status: "+statusNames())
	property := fs.String("property", "", "property name contains `TEXT`")
	propertyExact := fs.String("property-exact", "", "property name is exactly `NAME`")
	guest := fs.String("guest", "", "guest name contains `TEXT`")
	externalID := fs.String("external-id", "", "external booking ID contains `TEXT`")
	source := fs.String("source", "", "booking source, e.g. Airbnb, Booking.com, \"Direct booking\"")
	batch := fs.String("batch", "", "batch number")
	desc := fs.Bool("desc", false, "newest check-in first (default oldest first)")
	limit := fs.Int("limit", 0, "return at most `N` bookings (0 = all)")
	links := fs.Bool("links", false, "add guest check-in, payment and upsell links (one extra request per booking)")
	output := fs.String("o", "table", "output format: table, json or jsonl")
	var params paramFlag
	fs.Var(&params, "param", "extra raw API query parameter `key=value` (repeatable)")
	if err := parse(fs, args); err != nil {
		return err
	}

	if *output != "table" && *output != "json" && *output != "jsonl" {
		return usageErr("-o must be table, json or jsonl")
	}
	if *upcoming && *past {
		return usageErr("--upcoming and --past can't be used together")
	}
	if *status != "" {
		if _, ok := chekin.StatusFilters[*status]; !ok {
			return usageErr("unknown --status %q; valid: %s", *status, statusNames())
		}
	}
	if *limit < 0 {
		return usageErr("--limit can't be negative")
	}

	opts := chekin.ListOptions{
		Status: *status, Property: *property, PropertyExact: *propertyExact,
		Guest: *guest, ExternalID: *externalID, Source: *source, Batch: *batch,
		Descending: *desc, Limit: *limit, Extra: params.v,
	}
	today := time.Now()
	var err error
	for _, d := range []struct {
		flag, value string
		dst         *time.Time
	}{
		{"--from", *from, &opts.CheckInFrom},
		{"--to", *to, &opts.CheckInTo},
		{"--checkout-from", *checkoutFrom, &opts.CheckOutFrom},
		{"--checkout-to", *checkoutTo, &opts.CheckOutTo},
	} {
		if *d.dst, err = parseDate(d.value, today); err != nil {
			return usageErr("%s: %v", d.flag, err)
		}
	}
	if *upcoming {
		if !opts.CheckInFrom.IsZero() {
			return usageErr("--upcoming can't be combined with --from")
		}
		opts.CheckInFrom = today
	}
	if *past {
		if !opts.CheckInTo.IsZero() {
			return usageErr("--past can't be combined with --to")
		}
		opts.CheckInTo = today.AddDate(0, 0, -1)
	}

	s, err := a.newSession(ctx)
	if err != nil {
		return err
	}
	var bookings []chekin.Reservation
	err = s.call(ctx, func(c *chekin.Client) error {
		bookings, err = c.ListReservations(ctx, opts)
		return err
	})
	if err != nil {
		return err
	}

	var details []*chekin.ReservationDetail
	if *links && len(bookings) > 0 {
		ids := make([]string, len(bookings))
		for i, b := range bookings {
			ids[i] = b.ID
		}
		err = s.call(ctx, func(c *chekin.Client) error {
			details, err = c.GetReservations(ctx, ids)
			return err
		})
		if err != nil {
			return fmt.Errorf("fetching links: %w", err)
		}
		for i := range bookings {
			bookings[i].Raw, err = withLinks(bookings[i].Raw, details[i])
			if err != nil {
				return err
			}
		}
	}

	switch *output {
	case "json":
		return writeJSON(a.Stdout, bookings)
	case "jsonl":
		for _, b := range bookings {
			if _, err := fmt.Fprintf(a.Stdout, "%s\n", compact(b.Raw)); err != nil {
				return err
			}
		}
		return nil
	default:
		writeTable(a.Stdout, bookings, details)
		if len(bookings) == 0 {
			fmt.Fprintln(a.Stderr, "No bookings found.")
		}
		return nil
	}
}

func parseDate(s string, today time.Time) (time.Time, error) {
	switch strings.ToLower(s) {
	case "":
		return time.Time{}, nil
	case "today":
		return today, nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (use YYYY-MM-DD)", s)
	}
	return t, nil
}

// writeJSON prints the bookings exactly as the API returned them.
func writeJSON(w io.Writer, bookings []chekin.Reservation) error {
	raw := make([]json.RawMessage, len(bookings))
	for i, b := range bookings {
		raw[i] = b.Raw
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(raw)
}

func compact(raw json.RawMessage) []byte {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return raw
	}
	return buf.Bytes()
}

// writeTable prints bookings; details, when present, adds a check-in link column.
func writeTable(w io.Writer, bookings []chekin.Reservation, details []*chekin.ReservationDetail) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := "ID\tCHECK-IN\tCHECK-OUT\tPROPERTY\tGUEST\tGUESTS\tSTATUS\tSOURCE"
	if details != nil {
		header += "\tCHECK-IN LINK"
	}
	fmt.Fprintln(tw, header)
	for i, b := range bookings {
		property := b.HousingDisplayName
		if property == "" {
			property = b.HousingName
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			b.ID, day(b.CheckInDate), day(b.CheckOutDate), truncate(property, 32),
			orDash(truncate(b.GuestLeaderName, 24)), orDash(b.Guests), orDash(b.GeneralStatus), orDash(b.SourceName))
		if details != nil {
			fmt.Fprintf(tw, "\t%s", orDash(details[i].SignupFormLink))
		}
		fmt.Fprintln(tw)
	}
	tw.Flush()
}

// withLinks adds the guest links and dashboard URL to a list item's JSON.
func withLinks(raw json.RawMessage, d *chekin.ReservationDetail) (json.RawMessage, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}
	for k, v := range map[string]string{
		"signup_form_link": d.SignupFormLink,
		"payment_link":     d.PaymentLink,
		"upselling_link":   d.UpsellingLink,
		"dashboard_url":    chekin.DashboardURL(d.ID),
	} {
		b, _ := json.Marshal(v)
		obj[k] = b
	}
	return json.Marshal(obj)
}

func (a *App) bookingsGet(ctx context.Context, args []string) error {
	// Accept the ID before or after the flags.
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id, args = args[0], args[1:]
	}
	fs := a.flags("bookings get", "chekin bookings get ID [-o table|json]")
	output := fs.String("o", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return errUsage{err.Error()}
	}
	if id == "" && fs.NArg() > 0 {
		id = fs.Arg(0)
	} else if fs.NArg() > 0 {
		return usageErr("unexpected argument %q", fs.Arg(0))
	}
	if id == "" {
		return usageErr("usage: chekin bookings get ID")
	}
	if *output != "table" && *output != "json" {
		return usageErr("-o must be table or json")
	}

	s, err := a.newSession(ctx)
	if err != nil {
		return err
	}
	var d *chekin.ReservationDetail
	err = s.call(ctx, func(c *chekin.Client) error {
		d, err = c.GetReservation(ctx, id)
		return err
	})
	if err != nil {
		return err
	}

	if *output == "json" {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(d.Raw, &obj); err != nil {
			return err
		}
		obj["dashboard_url"], _ = json.Marshal(chekin.DashboardURL(d.ID))
		enc := json.NewEncoder(a.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	}

	price := "-"
	if d.Price != nil && fmt.Sprint(d.Price) != "" {
		price = strings.TrimSpace(fmt.Sprint(d.Price) + " " + d.Currency)
	}
	tw := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	for _, row := range [][2]string{
		{"ID", d.ID},
		{"Check-in", day(d.CheckInDate)},
		{"Check-out", day(d.CheckOutDate)},
		{"Nights", fmt.Sprint(d.NightsOfStay)},
		{"Guest", orDash(d.LeaderName)},
		{"Status", orDash(d.Status)},
		{"Source", orDash(d.SourceName)},
		{"External ID", orDash(d.ExternalID)},
		{"Price", price},
		{"Check-in link", orDash(d.SignupFormLink)},
		{"Payment link", orDash(d.PaymentLink)},
		{"Upsell link", orDash(d.UpsellingLink)},
		{"Dashboard", chekin.DashboardURL(d.ID)},
	} {
		fmt.Fprintf(tw, "%s:\t%s\n", row[0], row[1])
	}
	return tw.Flush()
}

func day(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return orDash(s)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n-1]) + "…"
}
