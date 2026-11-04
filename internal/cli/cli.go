// Package cli implements the chekin command line.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wilfred93/chekin-cli/internal/chekin"
	"github.com/wilfred93/chekin-cli/internal/config"
)

const usage = `chekin: unofficial command line tool for Chekin (chekin.com)

Usage:
  chekin login [--email EMAIL] [--password-stdin]   log in with email and password
  chekin login --token JWT|-                        save a token copied from the browser
  chekin logout                                     forget the saved login
  chekin whoami                                     show the logged-in account
  chekin bookings list [flags]                      list bookings (alias: reservations)
  chekin bookings get ID [-o json]                  show one booking, with its guest links
  chekin update [--check] [--yes]                   update to the latest release
  chekin version

Run "chekin <command> --help" for a command's flags.

Environment:
  CHEKIN_TOKEN                     token to use instead of the saved login
  CHEKIN_EMAIL, CHEKIN_PASSWORD    log in automatically when there is no valid token
  CHEKIN_API_URL                   API base URL (default https://a.chekin.io)
  CHEKIN_CONFIG_DIR                where the login is saved (default ~/.config/chekin)
`

// App holds the process's I/O so commands can be tested.
type App struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	releasesURL string // GitHub releases page; tests point it at a fake server
	exePath     string // binary that `chekin update` replaces; tests use a temp file
}

// errUsage marks errors caused by bad arguments (exit code 2).
type errUsage struct{ msg string }

func (e errUsage) Error() string { return e.msg }

func usageErr(format string, args ...any) error { return errUsage{fmt.Sprintf(format, args...)} }

// Run executes the command and returns the process exit code.
func (a *App) Run(ctx context.Context, args []string) int {
	err := a.run(ctx, args)
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return 0
	}
	fmt.Fprintln(a.Stderr, "error:", err)
	var u errUsage
	if errors.As(err, &u) {
		return 2
	}
	return 1
}

func (a *App) run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		fmt.Fprint(a.Stderr, usage)
		return usageErr("missing command")
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(a.Stdout, usage)
		return nil
	case "version", "--version":
		fmt.Fprintln(a.Stdout, "chekin", Version)
		return nil
	case "update":
		return a.update(ctx, args[1:])
	case "login":
		return a.login(ctx, args[1:])
	case "logout":
		return a.logout(args[1:])
	case "whoami":
		return a.whoami(args[1:])
	case "bookings", "reservations":
		sub := ""
		if len(args) > 1 {
			sub = args[1]
		}
		switch sub {
		case "list":
			return a.bookingsList(ctx, args[2:])
		case "get":
			return a.bookingsGet(ctx, args[2:])
		default:
			return usageErr("usage: chekin %s list [flags] | get ID [flags]", args[0])
		}
	default:
		return usageErr("unknown command %q (run `chekin help`)", args[0])
	}
}

func (a *App) flags(name, synopsis string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(a.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(a.Stderr, "Usage: %s\n\nFlags:\n", synopsis)
		fs.PrintDefaults()
	}
	return fs
}

// parse wraps flag parsing so bad flags exit with code 2.
func parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return errUsage{err.Error()}
	}
	if fs.NArg() > 0 {
		return usageErr("unexpected argument %q", fs.Arg(0))
	}
	return nil
}

func (a *App) baseURL() string {
	if u := os.Getenv("CHEKIN_API_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return chekin.DefaultBaseURL
}

func (a *App) client(token string) *chekin.Client {
	c := chekin.New(a.baseURL(), token)
	c.UserAgent = "chekin-cli/" + Version
	return c
}

func (a *App) logout(args []string) error {
	if err := parse(a.flags("logout", "chekin logout"), args); err != nil {
		return err
	}
	if err := config.Remove(); err != nil {
		return err
	}
	fmt.Fprintln(a.Stderr, "Logged out.")
	return nil
}

func (a *App) whoami(args []string) error {
	if err := parse(a.flags("whoami", "chekin whoami"), args); err != nil {
		return err
	}
	token := os.Getenv("CHEKIN_TOKEN")
	source := "CHEKIN_TOKEN"
	if token == "" {
		creds, err := config.Load()
		if err != nil {
			return err
		}
		if creds == nil || creds.Token == "" {
			return errNotLoggedIn
		}
		token, source = creds.Token, "saved login"
	}
	claims, err := chekin.ParseClaims(token)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "%s (from %s)\n", claims.Username, source)
	if claims.Exp != 0 {
		fmt.Fprintf(a.Stdout, "token expires %s\n", claims.Expiry().Local().Format("2006-01-02 15:04 MST"))
	}
	return nil
}
