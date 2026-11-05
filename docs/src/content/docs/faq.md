---
title: FAQ
description: Whether chekin is official or allowed, what it changes in your account and on your computer, why it doesn't use the partner API, and where your token goes.
---

## Is this official?

No. `chekin` isn't made, reviewed or supported by Chekin. It uses the API behind the web dashboard, which
Chekin doesn't document and may change or block at any time. See [How it works](/chekin-cli/how-it-works/).

## Is it allowed?

Chekin hasn't approved this project. `chekin` logs in with your own account and sends the same requests
the dashboard sends when you use it in a browser. Whether that fits your agreement with Chekin is between
you and Chekin: read their terms or ask them. A login that works doesn't mean Chekin permits it, and it
doesn't mean the endpoints will stay the same.

## Does it change anything in my account?

No booking is created, changed or deleted. Every booking request is a `GET`. The only other requests are
`POST`s that ask Chekin for a token: logging in and refreshing a token.

Some commands change files on your computer:

| Command | Effect |
| --- | --- |
| `chekin login` | Saves the token to `credentials.json` (see below). |
| Any command that refreshes a token or logs in from `CHEKIN_EMAIL` and `CHEKIN_PASSWORD` | Saves the new token to the same file. |
| `chekin logout` | Deletes that file. The token itself stays valid until it expires. |
| `chekin update` | Downloads a release from GitHub and replaces the `chekin` binary. |

`chekin` sends no telemetry. Apart from the Chekin API, the only host it contacts is GitHub, and only when
you run `chekin update`.

## Why not the partner API?

Chekin's partner API (for example `api/v3/reservations/`) needs partner access, and returns 403 to regular
accounts. The dashboard API works with any account that can log in to the dashboard. If you have partner
access, the partner API is documented and supported by Chekin, and this CLI's endpoints are not.

## Where does my token go?

On disk, to `~/.config/chekin/credentials.json`, readable only by you. `CHEKIN_CONFIG_DIR` and
`XDG_CONFIG_HOME` change the directory, and a token in `CHEKIN_TOKEN` is never written to disk. Your password
is never saved. [Where the token is saved](/chekin-cli/log-in/#where-the-token-is-saved) has the full rules.

Over the network, in the `Authorization` header of every request to the API at `CHEKIN_API_URL`, which
defaults to `https://a.chekin.io`. `chekin` refuses to follow a page link to any other host. Only set
`CHEKIN_API_URL` to a host you trust.
