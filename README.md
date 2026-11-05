# chekin-cli

`chekin` reads your [Chekin](https://chekin.com) bookings from the terminal. It
lists and filters them by date, status, property or guest, fetches each
booking's guest check-in, payment and upsell links, and prints them as a table
or as JSON for scripts and AI agents.

> **Unofficial.** This project isn't made or supported by Chekin. Chekin's
> partner API needs partner access, so `chekin` uses the undocumented API
> behind the web dashboard. It may change or break without notice.

**Documentation: https://wilfred93.github.io/chekin-cli/**

## Install

```sh
curl -fsSL https://wilfred93.github.io/chekin-cli/install.sh | sh   # macOS, Linux
go install github.com/wilfred93/chekin-cli/cmd/chekin@latest       # or, with Go
```

Windows builds are on the [releases page](https://github.com/wilfred93/chekin-cli/releases).
See [Install and update](https://wilfred93.github.io/chekin-cli/install/).

## Example

```sh
chekin login --email you@example.com          # prompts for the password
chekin bookings list --upcoming --limit 3
chekin bookings get <booking-id>              # guest links for one booking
```

## Documentation

- [Log in](https://wilfred93.github.io/chekin-cli/log-in/)
- [List bookings](https://wilfred93.github.io/chekin-cli/bookings/)
- [Scripts and AI agents](https://wilfred93.github.io/chekin-cli/agents/)
- [Command reference](https://wilfred93.github.io/chekin-cli/reference/commands/)
- [FAQ](https://wilfred93.github.io/chekin-cli/faq/)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, tests, adding an endpoint and
the docs style guide.

## License

[MIT](LICENSE)
