---
title: Troubleshooting and exit codes
description: What chekin's exit codes mean, and what to do about common errors such as expired tokens, wrong credentials and API changes.
---

## Exit codes

| Code | Meaning | What to do |
| --- | --- | --- |
| `0` | Success. An empty result is still a success. | |
| `1` | Something failed: the API, the login, or the network. | Read the error message. |
| `2` | Bad arguments: unknown command or flag, invalid value, conflicting flags. | Fix the command; see the [command reference](/chekin-cli/reference/commands/). |

Errors are printed to stderr as one line starting with `error:`.

## Not logged in

```text
error: not logged in: run `chekin login`, or set CHEKIN_TOKEN, or CHEKIN_EMAIL and CHEKIN_PASSWORD
```

There is no saved token and no credentials in the environment. [Log in](/chekin-cli/log-in/), or set the
environment variables.

## Session expired

```text
error: session expired: run `chekin login` again
error: chekin API error (HTTP 401): … (session expired or invalid: run `chekin login` again)
```

Tokens last about an hour. `chekin` refreshes a token that is about to expire, but an expired one can't be
refreshed. Run `chekin login` again. To make this automatic, set `CHEKIN_EMAIL` and `CHEKIN_PASSWORD`.

If you use `CHEKIN_TOKEN`, it's never refreshed: copy a new token, or switch to email and password.

## Wrong email or password

```text
error: chekin API error (HTTP 400): Unable to log in with provided credentials.
```

Check the email and password by logging in to [dashboard.chekin.com](https://dashboard.chekin.com). If you
sign in there with Google or GitHub, your account has no password: use
[a token from the browser](/chekin-cli/log-in/#paste-a-token-from-the-browser) instead.

## The token isn't accepted

```text
error: not a JWT (expected three dot-separated parts)
error: that token has already expired; copy a fresh one from the browser
```

`chekin login --token` checks the token before saving it. Copy it again with
`copy(localStorage.getItem('user'))` on a dashboard tab where you're logged in, and make sure nothing was
cut off. The JSON, the bare token and `JWT <token>` are all accepted.

## Login asks for a captcha, or stops working

The dashboard's own login (`api/v5/token/`) requires a Cloudflare captcha, which a CLI can't solve, so
`chekin` uses the older `api/v3/token/` endpoint. If Chekin removes or blocks that endpoint, password login
will fail. [Paste a token from the browser](/chekin-cli/log-in/#paste-a-token-from-the-browser) still works.

## 403 errors and the partner API

Chekin's official partner API (for example `api/v3/reservations/`) returns 403 to accounts without partner
access. `chekin` doesn't use it: it calls the dashboard's endpoints. If you see a 403 from `chekin`, the
token belongs to an account that can't see those bookings, or the dashboard API has changed. Check that you
can see the bookings in the dashboard with the same account.

## Missing bookings

- No date flags: `chekin` returns every booking since 2000. If you only see upcoming ones, you're probably
  passing `--upcoming` or `--from`.
- Filters combine, and `--property-exact` must match the whole name. Try `--property` with part of it.
- Bookings of deactivated properties are included, as in the dashboard.

## Booking not found

```text
error: booking a1b2c3d4 not found
```

The ID doesn't exist or belongs to another account. Copy the `ID` column from `chekin bookings list`.

## Corrupt saved login

```text
error: /Users/you/.config/chekin/credentials.json is corrupt (run `chekin logout`): …
```

Run `chekin logout` to delete the file, then log in again.

## Network errors

```text
error: request to a.chekin.io failed: …
```

`chekin` couldn't reach the API: check your connection, proxy or VPN. Requests time out after 30 seconds.
To point the CLI at another host, set `CHEKIN_API_URL`.

## Other API errors

`error: chekin API error (HTTP <code>): <message>` is a response from Chekin's API. 5xx errors are usually
temporary; try again. Since the API is undocumented, other errors may mean it has changed. Please
[open an issue](https://github.com/wilfred93/chekin-cli/issues) with the command you ran and the error,
**without your token**.
