package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/jgalea/glovo-cli/internal/glovo"
)

func cmdLogin(args []string) error {
	fs, _ := newCommonFlags("login")
	var email string
	var tokenMode bool
	var accessTokenMode bool
	var customerID int64
	fs.StringVar(&email, "email", "", "log in with your Glovo email + password")
	fs.BoolVar(&tokenMode, "token", false, "paste a glovo_refresh_token from your browser")
	fs.BoolVar(&accessTokenMode, "access-token", false, "paste your current access token; the CLI derives your customer id from it")
	fs.Int64Var(&customerID, "customer-id", 0, "override the customer id derived from --access-token")
	_ = fs.Parse(args)

	cl := glovo.NewClient(stderrLogf)

	switch {
	case email != "":
		fmt.Fprint(os.Stderr, "Password: ")
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		if err := cl.LoginPassword(email, strings.TrimSpace(string(pw))); err != nil {
			return err
		}
	case accessTokenMode:
		token, err := readTokenInput("Access token: ")
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		if err := cl.LoginAccessToken(token, customerID); err != nil {
			return err
		}
	default:
		// paste-token (default and --token)
		fmt.Fprintln(os.Stderr, "Paste your glovo_refresh_token (from browser localStorage), then Enter:")
		reader := bufio.NewReader(os.Stdin)
		tok, _ := reader.ReadString('\n')
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return fmt.Errorf("no token provided")
		}
		if err := cl.LoginToken(tok); err != nil {
			return err
		}
	}
	fmt.Fprintln(os.Stderr, "Logged in.")
	return nil
}

// readTokenInput reads a token from a hidden terminal prompt, or straight from
// stdin when it is piped (e.g. `pbpaste | glovo login --access-token`).
func readTokenInput(prompt string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, prompt)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return strings.TrimSpace(string(b)), err
	}
	b, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(b), nil
}
