package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/wilfred93/chekin-cli/internal/chekin"
	"github.com/wilfred93/chekin-cli/internal/config"
)

func (a *App) login(ctx context.Context, args []string) error {
	fs := a.flags("login", "chekin login [--email EMAIL] [--password-stdin] | --token JWT|-")
	email := fs.String("email", os.Getenv("CHEKIN_EMAIL"), "account email (default $CHEKIN_EMAIL)")
	passwordStdin := fs.Bool("password-stdin", false, "read the password from stdin")
	token := fs.String("token", "", "save this token instead of logging in (`-` reads it from stdin)")
	if err := parse(fs, args); err != nil {
		return err
	}

	in := bufio.NewReader(a.Stdin)

	if *token != "" {
		if *passwordStdin {
			return usageErr("--token can't be combined with --password-stdin")
		}
		t := *token
		if t == "-" {
			line, err := readLine(in)
			if err != nil {
				return fmt.Errorf("reading token from stdin: %w", err)
			}
			t = line
		}
		return a.saveToken(normalizeToken(t), "")
	}

	if *email == "" {
		if !a.interactive() {
			return usageErr("--email is required (or set CHEKIN_EMAIL)")
		}
		fmt.Fprint(a.Stderr, "Email: ")
		line, err := readLine(in)
		if err != nil {
			return err
		}
		*email = line
	}

	password, err := a.password(in, *passwordStdin)
	if err != nil {
		return err
	}

	t, err := a.client("").Login(ctx, *email, password)
	if err != nil {
		return err
	}
	return a.saveToken(t, *email)
}

func (a *App) password(in *bufio.Reader, fromStdin bool) (string, error) {
	switch {
	case fromStdin:
		line, err := readLine(in)
		if err != nil {
			return "", fmt.Errorf("reading password from stdin: %w", err)
		}
		return line, nil
	case os.Getenv("CHEKIN_PASSWORD") != "":
		return os.Getenv("CHEKIN_PASSWORD"), nil
	case a.interactive():
		fmt.Fprint(a.Stderr, "Password: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(a.Stderr)
		return string(b), err
	default:
		return "", usageErr("no password: use --password-stdin or set CHEKIN_PASSWORD")
	}
}

// saveToken checks the token looks like a JWT that hasn't expired, then saves it.
func (a *App) saveToken(token, email string) error {
	if token == "" {
		return usageErr("empty token")
	}
	claims, err := chekin.ParseClaims(token)
	if err != nil {
		return err
	}
	if claims.Exp != 0 && time.Now().After(claims.Expiry()) {
		return errors.New("that token has already expired; copy a fresh one from the browser")
	}
	if email == "" {
		email = claims.Username
	}
	if err := config.Save(&config.Credentials{Token: token, Email: email, BaseURL: a.baseURL()}); err != nil {
		return err
	}
	msg := "Logged in as " + email
	if claims.Exp != 0 {
		msg += fmt.Sprintf(" (token expires %s)", claims.Expiry().Local().Format("15:04"))
	}
	fmt.Fprintln(a.Stderr, msg)
	return nil
}

// normalizeToken accepts a bare JWT, "JWT <token>", or the JSON stored in the
// dashboard's localStorage ({"token":"..."}).
func normalizeToken(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "JWT ")
	var stored struct {
		Token string `json:"token"`
	}
	if strings.HasPrefix(s, "{") && json.Unmarshal([]byte(s), &stored) == nil {
		return stored.Token
	}
	return strings.Trim(s, `"`)
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		if errors.Is(err, io.EOF) {
			return "", errors.New("no input")
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// interactive reports whether we can prompt the user.
func (a *App) interactive() bool {
	f, ok := a.Stdin.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}
