---
title: Scripts and AI agents
description: Run chekin without prompts from scripts, CI and AI agents, with JSON output, exit codes and example workflows.
---

`chekin` is built to be driven by programs: it never prompts when there's no terminal, prints data to stdout
and everything else to stderr, and uses distinct exit codes for bad arguments and failures.

## Log in without prompts

Set credentials in the environment. `chekin` logs in when it needs to and saves the token, so later commands
reuse it until it expires:

```sh
export CHEKIN_EMAIL=you@example.com
export CHEKIN_PASSWORD=...          # from your secret store, not hard-coded
chekin bookings list --upcoming -o json
```

Or give it a token that is used as is and never saved:

```sh
export CHEKIN_TOKEN=eyJ...
```

A token lasts about an hour, and `CHEKIN_TOKEN` is never refreshed. For anything that runs longer, use
`CHEKIN_EMAIL` and `CHEKIN_PASSWORD`. See [Log in](/chekin-cli/log-in/#environment-variables).

To keep an agent's login apart from yours, give it its own `CHEKIN_CONFIG_DIR`.

:::caution
Give agents credentials through the environment or a secret store. Don't paste tokens or passwords into a
chat with an AI agent.
:::

## Read JSON, not tables

Use `-o json` (one array) or `-o jsonl` (one booking per line). Both contain the full API objects. The table
layout may change; the JSON is what the API returns. See [Output formats](/chekin-cli/output/).

## Exit codes and errors

| Code | Meaning |
| --- | --- |
| `0` | Success, including an empty result |
| `1` | Error: API, authentication, network |
| `2` | Bad arguments: unknown command or flag, invalid date or status |

Errors are one line on stderr, starting with `error:`. Retrying makes sense for network errors and HTTP 5xx;
it doesn't for exit code `2` or a login error. See [Troubleshooting](/chekin-cli/troubleshooting/).

## Example workflows

Upcoming bookings whose guests haven't checked in, with their check-in links:

```sh
chekin bookings list --upcoming --status incomplete --links -o jsonl \
  | jq -r '[.check_in_date[:10], .housing_name, .signup_form_link] | @tsv'
```

Bookings checking in tomorrow:

```sh
chekin bookings list --from tomorrow --to tomorrow -o json | jq length
```

Unpaid bookings, as `id` and dashboard URL:

```sh
chekin bookings list --status unpaid --links -o jsonl | jq -r '"\(.id) \(.dashboard_url)"'
```

Fail a script cleanly when not logged in:

```sh
if ! out=$(chekin bookings list --upcoming -o json 2>err.txt); then
  echo "chekin failed: $(cat err.txt)" >&2
  exit 1
fi
```

## Instructions for an agent

A starting point to give an AI agent that has shell access:

```text
Use the `chekin` CLI to read Chekin bookings. It is already logged in.
- List: chekin bookings list [--upcoming|--past] [--from D] [--to D] [--status S] [--property T] [--guest T] [--source S] [--limit N] -o json
- One booking and its guest links: chekin bookings get ID -o json
- Add guest links to a list: --links (slow: one request per booking; filter first)
- Dates: YYYY-MM-DD, today, yesterday, tomorrow
- Exit code 2 means bad arguments: fix the command. Exit code 1 means an API or login error: report it.
Never print or ask for the login token.
```

The [command reference](/chekin-cli/reference/commands/) has every flag. An agent can also read this site as
plain text at [`/llms-full.txt`](/chekin-cli/llms-full.txt).
