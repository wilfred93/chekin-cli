---
title: Output formats
description: Print bookings as a table, JSON or JSON Lines, and which fields each format contains.
---

Choose the format with `-o`:

| `-o` | For | Commands |
| --- | --- | --- |
| `table` (default) | Reading in a terminal | `bookings list`, `bookings get` |
| `json` | Scripts and AI agents; one JSON document | `bookings list`, `bookings get` |
| `jsonl` | Streaming and line-based tools; one booking per line | `bookings list` |

Data always goes to stdout. Messages such as `No bookings found.` and errors go to stderr, so they never
mix into JSON.

## Table

A fixed set of columns, aligned with spaces. Long property and guest names are cut short with `…`, and
empty values show as `-`. The columns are described in [List bookings](/chekin-cli/bookings/#what-you-get-by-default).

The table is meant for people; its layout may change. Use `json` or `jsonl` in scripts.

## JSON

```sh
chekin bookings list --upcoming -o json
```

An indented JSON array. Each element is the booking **exactly as the API returned it**, with every field,
including ones `chekin` doesn't use:

```json
[
  {
    "id": "a1b2c3d4",
    "check_in_date": "2026-11-07T00:00:00Z",
    "check_out_date": "2026-11-14T00:00:00Z",
    "housing_id": "…",
    "housing_name": "Sea view flat",
    "housing_display_name": "Sea view flat",
    "guest_leader_name": "",
    "guests": "0/2",
    "general_status": "INCOMPLETE",
    "detailed_statuses": { "…": "…" },
    "source_name": "Airbnb",
    "external_id": "HMABC12345"
  }
]
```

No bookings gives `[]`.

`chekin bookings get ID -o json` prints a single object: the full booking from the booking page endpoint,
about 85 fields.

## JSON Lines

```sh
chekin bookings list -o jsonl --limit 50
```

One compact JSON object per line, the same objects as `json`. Nothing is printed when there are no bookings.

```sh
chekin bookings list -o jsonl | jq -r 'select(.general_status == "INCOMPLETE") | .id'
```

## Which fields come from where

The CLI uses two endpoints, and they return different fields. See [How it works](/chekin-cli/how-it-works/).

| Output | Source | Fields |
| --- | --- | --- |
| `bookings list -o json/jsonl` | `api/v4/status/reservations/` | Dashboard list fields: `id`, dates, `housing_*`, `guest_leader_name`, `guests`, `general_status`, `detailed_statuses`, `source_name`, `external_id`… |
| `bookings get -o json` | `api/v3/light/reservations/{id}/` | The full booking (about 85 fields), including `status`, `nights_of_stay`, `price`, `currency`, `booking_reference`, `default_leader_full_name` and the guest links |

Fields added by `chekin`, not by the API:

| Field | Added to |
| --- | --- |
| `dashboard_url` | `bookings get -o json`, and `bookings list --links` |
| `signup_form_link`, `payment_link`, `upselling_link` | `bookings list --links` (copied from the booking page endpoint) |

Field names and values are Chekin's and can change without notice. Check for missing fields rather than
assuming they're there.
