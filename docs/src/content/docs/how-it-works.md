---
title: How it works
description: The Chekin dashboard API endpoints chekin uses, the headers it sends, pagination and known API quirks.
---

`chekin` talks to the API behind the Chekin web dashboard, at `https://a.chekin.io`. These endpoints were
found by watching the dashboard's requests. Chekin doesn't document them and may change them at any time.

## Endpoints

| What | Request | Used by |
| --- | --- | --- |
| Log in | `POST api/v3/token/` with `{"email","password"}` → `{"token"}` | `chekin login`, `CHEKIN_EMAIL`/`CHEKIN_PASSWORD` |
| Refresh | `POST api/v3/api-token-refresh/` with `{"token"}` → `{"token"}` | Any command, when the token has under 5 minutes left |
| List bookings | `GET api/v4/status/reservations/?page=N&…` | `chekin bookings list` |
| One booking | `GET api/v3/light/reservations/{id}/` | `chekin bookings get`, `bookings list --links` |

The dashboard's current login is `api/v5/token/`, but it requires a Cloudflare Turnstile captcha. The older
`api/v3/token/` doesn't.

## Headers

Every request sends:

```http
Authorization: JWT <token>
X-Source: DASHBOARD
Accept: application/json
Content-Type: application/json
User-Agent: chekin-cli/<version>
```

`Authorization: JWT` (not `Bearer`) and `X-Source: DASHBOARD` are what the dashboard sends.

## Tokens

Tokens are JWTs. `chekin` reads the payload, without verifying the signature, to show the account
(`username`) and to know when it expires (`exp`). They last about an hour. See
[Log in](/chekin-cli/log-in/#how-long-it-lasts) for the refresh rules.

## Listing bookings

`chekin bookings list` builds the same query the dashboard does:

| Parameter | Value |
| --- | --- |
| `ordering` | `check_in_date,check_out_date,created_at`, with `-` in front for `--desc` |
| `is_housing_deactivated` | `true` (the dashboard always sends it; includes deactivated properties) |
| `check_in_date_from` | `--from`, or `2000-01-01T00:00:00` when not given |
| `check_in_date_until` | `--to`, as `…T23:59:59` |
| `check_out_date_after`, `check_out_date_before` | `--checkout-from`, `--checkout-to` |
| `name`, `exact_name` | `--property`, `--property-exact` |
| `guest_name`, `source_name`, `external_id_contains`, `lote` | `--guest`, `--source`, `--external-id`, `--batch` |
| `complete`, `booking_paid`, `police_check_in_status`, `police_check_out_status`, `stats`, `identity_verification`, `remote_access_status` | `--status` (one parameter, depending on the status) |
| `page` | Starts at `1` |
| anything | `--param key=value` |

### Pagination

Responses have `results` and a `next` URL. `chekin` follows `next` until it is `null` or a page comes back
empty, or until `--limit` is reached. It gives up after 1000 pages.

`next` links come back as `http://`; `chekin` keeps the API's `https` scheme, and refuses to follow a link
to any other host so the token is never sent elsewhere.

### `--links`

The list endpoint doesn't include guest links, so `--links` fetches `api/v3/light/reservations/{id}/` for
every booking, five requests at a time, and merges the links into the results.

## Known API quirks

- **Upcoming only by default.** With no date filter, the list endpoint returns only upcoming bookings. With
  only an end date, it goes back about a year. That's why `chekin` always sends `check_in_date_from`.
- **`count` is meaningless.** The list response's `count` is always `999999999`, so `chekin` ignores it and
  follows `next` instead.
- **Two booking shapes.** The list and the single-booking endpoints return different fields, and different
  status fields (`general_status` vs `status`). See [Output formats](/chekin-cli/output/#which-fields-come-from-where).
- **Errors come in several formats:** `{"message"}`, `{"error"}`, `{"detail"}` and `{"non_field_errors": [...]}`.
  `chekin` shows whichever is present.

## Source

The client is in [`internal/chekin`](https://github.com/wilfred93/chekin-cli/tree/master/internal/chekin).
