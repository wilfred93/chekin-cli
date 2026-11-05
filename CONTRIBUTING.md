# Contributing to chekin-cli

chekin-cli talks to Chekin through the same undocumented API the web dashboard
uses. Most features start with finding the right endpoint, so most of this guide
is about doing that without leaking anyone's data.

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Setup

You need Go 1.26 or newer.

```sh
go test ./...                              # run all tests
go run ./cmd/chekin bookings list --limit 5  # run the CLI from source
```

Tests never touch the network, so you don't need a Chekin account to run them.
To try a change against your own account, log in with `go run ./cmd/chekin login`.

## Changing a command or flag

The [command reference](docs/src/content/docs/reference/commands.md) is generated
from the CLI's help text. After changing a command, flag or help text, regenerate it:

```sh
go test ./internal/cli -run TestCommandReference -update
```

`go test ./...` (and CI) fails until the reference matches. Commit the
regenerated file with your change.

## Docs

The documentation site is in [`docs/`](docs/), built with
[Astro Starlight](https://starlight.astro.build). You need Node.js.

```sh
cd docs && npm install && npm run dev   # live preview at http://localhost:4321/chekin-cli/
npm run build                           # also checks every internal link
```

- Pages are Markdown files in `docs/src/content/docs/`.
- A new page also needs an entry in the `sidebar` in `docs/astro.config.mjs`.
- Don't edit `reference/commands.md` by hand. Regenerate it as described above.

If you change how a command behaves, update the page that describes it in the
same pull request. Follow the [docs style](#docs-style) below.

## Adding an endpoint

1. **Find it.** Open [dashboard.chekin.com](https://dashboard.chekin.com), open the
   browser's DevTools, go to the Network tab, and do the thing in the dashboard
   you want the CLI to do. Note the method, path, query parameters and the shape
   of the response.
2. **Keep private data out of the repo and out of GitHub.**
   - Never commit HAR files. They contain your token and your guests' data, and
     `*.har` is in `.gitignore` for that reason.
   - Never paste tokens, guest names, emails, phone numbers, addresses or booking
     IDs into issues, pull requests, commits or test fixtures. Replace them with
     made-up values (`b1`, `Jane Doe`, `jane@example.com`).
3. **Add it to the client** in `internal/chekin/`, with the same headers the other
   requests send (`Authorization: JWT <token>`, `X-Source: DASHBOARD`).
4. **Test it against a fake server.** Use `httptest` like `fakeAPI` in
   [`internal/chekin/reservations_test.go`](internal/chekin/reservations_test.go):
   serve made-up JSON shaped like the real response, and check the request
   path, query and headers. `chekin.MakeTestToken` builds a token for tests.
   Tests and CI must never call the real API.
5. **Document it** in the endpoint table on [How it works](docs/src/content/docs/how-it-works.md).

### Output

- `-o json` and `-o jsonl` print the raw API object, with every field the API
  returned. Don't rename, drop or reshape fields there. Agents rely on it. Adding
  a field that the API doesn't return (like `dashboard_url`) is fine.
- The table (`-o table`) is for people. Add a column only if a person needs it
  to decide what to do next. Everything else is already in `-o json`.

## Commits and pull requests

Write commit messages like the existing history: one line, imperative mood,
no trailing period.

```
Add prebuilt releases, install script and chekin update
Add documentation site built with Astro Starlight
```

Keep each pull request to one change. Before opening it:

- [ ] `go test ./...` passes
- [ ] The command reference is regenerated if you changed a command or flag
- [ ] The docs describe the new behavior
- [ ] No tokens, HAR files or real guest data anywhere in the diff

## Docs style

Use this for the docs site, the README, help text and error messages.

- Say what the command does and show it. Every sentence states a fact, a step or a limit.
- Don't use: seamless(ly), robust, powerful, leverage, effortless(ly),
  comprehensive, streamline, delve/dive into, unlock, empower, "game-changer",
  "it's worth noting".
- No "Whether you're X or Y", no rhetorical questions as openers, no lists of
  three padded for rhythm, no emoji, no exclamation marks.
- Sample output must be real output from the CLI, with made-up names and IDs.

## Releasing (maintainers)

1. Make sure `master` is green in CI.
2. Tag and push:

   ```sh
   git tag v0.3.0 && git push origin v0.3.0
   ```

   The [release workflow](.github/workflows/release.yml) runs the tests, builds
   the binaries with [`scripts/build-release.sh`](scripts/build-release.sh) and
   publishes a GitHub release with generated notes.
3. Tags with a suffix, like `v0.3.0-rc.1`, become pre-releases. `chekin update`
   and the install script skip them, so use them to test a release first.

To build the release archives locally without publishing anything, run
`scripts/build-release.sh v0.0.0-test` and look in `dist/`.
