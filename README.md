# chekin-cli

Unofficial [Chekin](https://chekin.com) CLI tool for your agents.

**Documentation: https://wilfred93.github.io/chekin-cli/**

> **Unofficial.** Chekin's partner API (`api/v3/reservations/`) needs partner
> access, so this CLI uses the same API as the web dashboard
> (`dashboard.chekin.com` → `https://a.chekin.io`). Those endpoints are
> undocumented and may change without notice.

## Install

```sh
curl -fsSL https://wilfred93.github.io/chekin-cli/install.sh | sh   # macOS, Linux
go install github.com/wilfred93/chekin-cli/cmd/chekin@latest       # or, with Go
```

Windows builds are on the [releases page](https://github.com/wilfred93/chekin-cli/releases).
See [Install and update](https://wilfred93.github.io/chekin-cli/install/) for the details.

## Update

```sh
chekin update --check   # is there a newer release?
chekin update           # download, verify the checksum, replace the binary
```

## Log in

```sh
chekin login --email you@example.com          # prompts for the password
echo "$PASSWORD" | chekin login --email you@example.com --password-stdin
```

If password login doesn't work for you (for example, you sign in with Google
or GitHub), copy the token from a logged-in dashboard tab instead. In the
browser console on `dashboard.chekin.com`, run:

```js
copy(localStorage.getItem('user'))
```

Then paste it (either the JSON or the bare token works):

```sh
pbpaste | chekin login --token -
```

The token is saved to `~/.config/chekin/credentials.json` (mode 0600). It lasts
about an hour and is refreshed automatically while it's still valid. Once it has
expired, run `chekin login` again.

`chekin whoami` shows the account and when the token expires. `chekin logout`
deletes the saved token.

### For agents and scripts

Set these instead of running `chekin login`:

| Variable | Purpose |
| --- | --- |
| `CHEKIN_TOKEN` | Token to use (not saved) |
| `CHEKIN_EMAIL`, `CHEKIN_PASSWORD` | Log in automatically whenever there is no valid token |
| `CHEKIN_API_URL` | API base URL (default `https://a.chekin.io`) |
| `CHEKIN_CONFIG_DIR` | Where the login is saved |

Exit codes: `0` success, `1` error (API, auth, network), `2` bad arguments.
Errors go to stderr; data goes to stdout.

## List bookings

```sh
chekin bookings list                       # all bookings, oldest check-in first
chekin bookings list --upcoming            # checking in today or later
chekin bookings list --past --desc         # past bookings, newest first
chekin bookings list --from 2026-01-01 --to 2026-03-31
chekin bookings list --status incomplete --source Airbnb
chekin bookings list -o json               # full API objects, for agents
chekin bookings list -o jsonl --limit 50   # one JSON object per line
```

`reservations` is an alias for `bookings`.

| Flag | Filter |
| --- | --- |
| `--from`, `--to` | Check-in date range (`YYYY-MM-DD`, `today`, `yesterday`, `tomorrow`) |
| `--checkout-from`, `--checkout-to` | Check-out date range |
| `--upcoming` / `--past` | Check-in from today / before today |
| `--status` | `complete`, `incomplete`, `unpaid`, `police-check-in-error`, `police-check-out-error`, `stats-error`, `id-verification-failed`, `keys-generated`, `keys-pending`, `keys-error` |
| `--property` / `--property-exact` | Property name contains / equals |
| `--guest` | Guest name |
| `--external-id` | External booking ID contains |
| `--source` | Booking source, e.g. `Airbnb`, `Booking.com`, `Direct booking` |
| `--batch` | Batch number |
| `--desc` | Newest check-in first |
| `--limit N` | Stop after N bookings |
| `-o` | `table` (default), `json`, `jsonl` |
| `--param key=value` | Extra raw query parameter for the API (repeatable) |
| `--links` | Add guest links (check-in, payment, upsell) and the dashboard URL. Costs one extra request per booking (~20 s for 128) |

## Show one booking and its links

```sh
chekin bookings get <booking-id>          # summary with check-in, payment, upsell and dashboard links
chekin bookings get <booking-id> -o json  # full booking (~85 fields) plus dashboard_url
chekin bookings list --upcoming --links   # links for many bookings at once
```

Guest links look like `https://guest.chekin.com/<code>`. Each is its own code, unrelated to
the booking ID, so they have to be fetched. The dashboard page is
`https://dashboard.chekin.com/bookings/<booking-id>`.

With no date flags the CLI asks for every booking since 2000. Left to its own
defaults, the API returns only upcoming bookings.

## How it talks to Chekin

| What | Endpoint |
| --- | --- |
| Login | `POST api/v3/token/` (`{"email","password"}` → `{"token"}`). The dashboard's own login, `api/v5/token/`, requires a Cloudflare captcha. |
| Refresh | `POST api/v3/api-token-refresh/` (`{"token"}` → `{"token"}`) |
| Bookings | `GET api/v4/status/reservations/?page=N&…`, paginated with `next` links |
| One booking | `GET api/v3/light/reservations/{id}/` (has `signup_form_link`, `payment_link`, `upselling_link`) |

Requests send `Authorization: JWT <token>` and `X-Source: DASHBOARD`, like the
dashboard does.

## Develop

```sh
go test ./...
```

To release, push a version tag (`git tag v0.2.0 && git push origin v0.2.0`). The
[release workflow](.github/workflows/release.yml) builds the binaries with
[`scripts/build-release.sh`](scripts/build-release.sh) and publishes them.

The documentation site is in [`docs/`](docs/) ([Astro Starlight](https://starlight.astro.build)):

```sh
cd docs && npm install && npm run dev
```

Its [command reference](docs/src/content/docs/reference/commands.md) is generated from the CLI's help
text. After changing a command or flag, regenerate it (`go test` fails until you do):

```sh
go test ./internal/cli -run TestCommandReference -update
```
