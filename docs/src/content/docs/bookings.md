---
title: List bookings
description: List your Chekin bookings and filter them by date, status, property, guest, source, external ID or batch.
---

```sh
chekin bookings list
```

```text
ID        CHECK-IN    CHECK-OUT   PROPERTY       GUEST         GUESTS  STATUS      SOURCE
f3a4b5c6  2025-06-01  2025-06-08  Sea view flat  Anna Müller   2/2     COMPLETE    Airbnb
a1b2c3d4  2026-11-07  2026-11-14  Sea view flat  -             0/2     INCOMPLETE  Airbnb
e5f6a7b8  2026-11-09  2026-11-12  Old town loft  Maria Garcia  2/2     COMPLETE    Booking.com
```

`reservations` is an alias: `chekin reservations list` does the same.

## What you get by default

With no flags, `chekin bookings list` returns **every booking**, past and future, oldest check-in first.
It follows the API's pages until there are no more.

:::note
This is different from the API's own default, which returns only upcoming bookings. To get everything,
`chekin` asks for check-ins from 1 January 2000 onwards unless you pass `--from`.
:::

The table columns are:

| Column | What it shows |
| --- | --- |
| `ID` | Booking ID, for [`chekin bookings get`](/chekin-cli/booking-details/) |
| `CHECK-IN`, `CHECK-OUT` | Dates, `YYYY-MM-DD` |
| `PROPERTY` | Property display name (cut to 32 characters) |
| `GUEST` | Lead guest, or `-` if nobody has checked in yet |
| `GUESTS` | Guests registered / expected |
| `STATUS` | Overall check-in status, e.g. `COMPLETE` or `INCOMPLETE` |
| `SOURCE` | Where the booking came from, e.g. `Airbnb` |

For every field the API returns, use [`-o json`](/chekin-cli/output/).

## Upcoming and past bookings

```sh
chekin bookings list --upcoming        # checking in today or later
chekin bookings list --past --desc     # checked in before today, newest first
```

`--upcoming` and `--past` can't be used together. `--upcoming` can't be combined with `--from`, and
`--past` can't be combined with `--to`.

## Date ranges

```sh
chekin bookings list --from 2026-01-01 --to 2026-03-31        # check-in in Q1
chekin bookings list --checkout-from today --checkout-to tomorrow
```

| Flag | Meaning |
| --- | --- |
| `--from DATE` | Check-in on or after `DATE` |
| `--to DATE` | Check-in on or before `DATE` |
| `--checkout-from DATE` | Check-out on or after `DATE` |
| `--checkout-to DATE` | Check-out on or before `DATE` |

`DATE` is `YYYY-MM-DD`, `today`, `yesterday` or `tomorrow`. Ranges include both ends: `--to 2026-03-31`
includes bookings checking in on 31 March.

## Filter by status

```sh
chekin bookings list --upcoming --status incomplete     # guests still to check in
chekin bookings list --status police-check-in-error
```

These match the status dropdown in the dashboard:

| `--status` | Bookings where |
| --- | --- |
| `complete` | Guest registration is complete |
| `incomplete` | Guest registration is incomplete |
| `unpaid` | Payment is pending |
| `police-check-in-error` | Sending the guests to the police failed |
| `police-check-out-error` | Sending the check-out to the police failed |
| `stats-error` | Sending statistics failed |
| `id-verification-failed` | Identity verification failed |
| `keys-generated` | Remote access keys were generated |
| `keys-pending` | Remote access keys haven't been generated |
| `keys-error` | Generating remote access keys failed |

You can pass one status at a time.

## Filter by property, guest, source and more

```sh
chekin bookings list --property "sea view"
chekin bookings list --property-exact "Sea view flat"
chekin bookings list --guest garcia
chekin bookings list --source Airbnb
chekin bookings list --external-id HMABC
chekin bookings list --batch 12
```

| Flag | Matches |
| --- | --- |
| `--property TEXT` | Property name contains `TEXT` |
| `--property-exact NAME` | Property name is exactly `NAME` (wins over `--property`) |
| `--guest TEXT` | Guest name |
| `--source NAME` | Booking source, e.g. `Airbnb`, `Booking.com`, `Direct booking` |
| `--external-id TEXT` | The booking ID from the channel (Airbnb, Booking.com…) contains `TEXT` |
| `--batch N` | Batch number |

Filters combine: `--upcoming --source Airbnb --status incomplete` returns upcoming Airbnb bookings with
incomplete check-in.

Matching is done by the Chekin API, so how it treats case and accents is up to the API.

## Sorting and limits

```sh
chekin bookings list --desc              # newest check-in first
chekin bookings list --desc --limit 10   # the 10 most recent
```

Bookings are sorted by check-in, then check-out, then creation date. `--desc` reverses the order.
`--limit N` stops after `N` bookings and fetches no more pages than needed. `0` (the default) means no limit.

## Extra API parameters

`--param key=value` adds a raw query parameter to the API request, for filters the CLI doesn't have a flag
for yet. Repeat it to add more. A `--param` overrides a parameter the CLI sets itself.

```sh
chekin bookings list --param page_size=100
chekin bookings list --param ordering=-created_at
```

See [How it works](/chekin-cli/how-it-works/) for the parameters the CLI sends.

## Guest links for many bookings

`--links` adds each booking's guest check-in link to the table (and the check-in, payment and upsell links
plus the dashboard URL to JSON output). It costs one extra request per booking, about 20 seconds for 128
bookings, so combine it with filters:

```sh
chekin bookings list --upcoming --status incomplete --links
```

See [Booking details and guest links](/chekin-cli/booking-details/).

## No results

When nothing matches, the table has only its header and `No bookings found.` is printed to stderr. The exit
code is still `0`. JSON output is an empty list (`[]`), and JSON Lines output is empty.
