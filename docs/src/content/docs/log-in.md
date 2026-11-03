---
title: Log in
description: Log in with your email and password or with a token copied from the dashboard, and learn where the token is saved and how it is refreshed.
---

`chekin` needs a Chekin login token. You can get one by logging in with your email and password, or by
copying it from a dashboard tab where you're already logged in.

:::danger[Keep your token private]
The token gives full access to your Chekin account until it expires. Don't paste it into chats, AI
prompts, GitHub issues or logs. When reporting a problem, leave the token out.
:::

## Email and password

```sh
chekin login --email you@example.com
```

`chekin` asks for the password without echoing it, then saves the token:

```text
Password:
Logged in as you@example.com (token expires 15:42)
```

If you leave out `--email`, `chekin` asks for it too (or uses `CHEKIN_EMAIL` when it is set).

To pass the password without a prompt, for example from a password manager, pipe it in with
`--password-stdin`:

```sh
echo "$PASSWORD" | chekin login --email you@example.com --password-stdin
op read "op://Private/Chekin/password" | chekin login --email you@example.com --password-stdin
```

`chekin` never saves your password.

## Paste a token from the browser

Use this if password login doesn't work for you, for example because you sign in to Chekin with Google or
GitHub and have no password.

1. Open [dashboard.chekin.com](https://dashboard.chekin.com) and log in.
2. Open the browser's developer console and run:

   ```js
   copy(localStorage.getItem('user'))
   ```

   This copies the dashboard's saved login to the clipboard.
3. Pass it to `chekin login --token -`, which reads it from stdin:

   ```sh
   pbpaste | chekin login --token -          # macOS
   xclip -o -selection clipboard | chekin login --token -   # Linux
   ```

`--token` accepts the JSON from `localStorage` (`{"token":"…"}`), a bare token, or `JWT <token>`.
You can also pass the token directly (`--token eyJ…`), but then it ends up in your shell history.

`chekin` checks that the token is well formed and hasn't expired before saving it.

## Where the token is saved

The token is saved to `~/.config/chekin/credentials.json`, readable only by you (mode `0600`), together
with the email and API URL. If `XDG_CONFIG_HOME` is set, the file goes in `$XDG_CONFIG_HOME/chekin/` instead,
and `CHEKIN_CONFIG_DIR` overrides both.

## How long it lasts

Chekin tokens last about an hour. Each time you run a command, `chekin` checks the saved token:

- **More than 5 minutes left:** it is used as is.
- **Less than 5 minutes left:** `chekin` swaps it for a new one (`api/v3/api-token-refresh/`) and saves that.
- **Expired:** it can't be refreshed. `chekin` logs in again with `CHEKIN_EMAIL` and `CHEKIN_PASSWORD` if
  they are set; otherwise it stops with ``session expired: run `chekin login` again``.

So as long as you use `chekin` at least once an hour, a login lasts indefinitely. If you don't, run
`chekin login` again, or set `CHEKIN_EMAIL` and `CHEKIN_PASSWORD` so it logs in by itself.

If the API rejects a token mid-command, `chekin` logs in again from the environment variables (when set)
and retries once.

## Check who is logged in

```sh
chekin whoami
```

```text
you@example.com (from saved login)
token expires 2026-11-03 15:42 CET
```

When `CHEKIN_TOKEN` is set, `whoami` shows that token instead (`from CHEKIN_TOKEN`).

## Log out

```sh
chekin logout
```

This deletes the saved token file. It doesn't sign you out of the dashboard, and a copied token stays
valid until it expires.

## Environment variables

Use these instead of `chekin login` in scripts, CI and AI agents. See
[Scripts and AI agents](/chekin-cli/agents/).

| Variable | Purpose |
| --- | --- |
| `CHEKIN_TOKEN` | Token to use. Takes priority over the saved login and is never saved or refreshed. |
| `CHEKIN_EMAIL`, `CHEKIN_PASSWORD` | Log in automatically when there is no valid token. The new token is saved. |
| `CHEKIN_API_URL` | API base URL (default `https://a.chekin.io`). |
| `CHEKIN_CONFIG_DIR` | Directory for `credentials.json` (default `~/.config/chekin`). |

`chekin` looks for a token in this order: `CHEKIN_TOKEN`, then the saved login, then `CHEKIN_EMAIL` and
`CHEKIN_PASSWORD`.
