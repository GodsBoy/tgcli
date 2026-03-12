package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

// Flow handles the phone + OTP + 2FA authentication flow.
type Flow struct {
	Phone  string
	Stdin  io.Reader
	Stderr io.Writer
}

// Run executes the auth flow using the provided API client.
func (f *Flow) Run(ctx context.Context, client *auth.Client) error {
	flow := auth.NewFlow(
		&terminalAuth{
			phone:  f.Phone,
			stdin:  f.Stdin,
			stderr: f.Stderr,
		},
		auth.SendCodeOptions{},
	)
	return flow.Run(ctx, client)
}

// terminalAuth implements gotd auth.UserAuthenticator for terminal-based auth.
type terminalAuth struct {
	phone  string
	stdin  io.Reader
	stderr io.Writer
}

func (a *terminalAuth) Phone(_ context.Context) (string, error) {
	if a.phone != "" {
		return a.phone, nil
	}
	fmt.Fprint(a.stderr, "Enter phone number: ")
	return readLine(a.stdin)
}

func (a *terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Fprint(a.stderr, "Enter 2FA password: ")
	if f, ok := a.stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		pw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(a.stderr)
		return string(pw), err
	}
	return readLine(a.stdin)
}

func (a *terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Fprint(a.stderr, "Enter OTP code: ")
	return readLine(a.stdin)
}

func (a *terminalAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil // Auto-accept
}

func (a *terminalAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign up not supported; use an existing Telegram account")
}

func readLine(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("no input")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

// CheckAuthorization checks if the current session is authorized.
func CheckAuthorization(ctx context.Context, api *tg.Client) (*tg.User, error) {
	result, err := api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("not authenticated")
	}
	user, ok := result[0].(*tg.User)
	if !ok {
		return nil, fmt.Errorf("unexpected user type: %T", result[0])
	}
	return user, nil
}

// Logout invalidates the current session.
func Logout(ctx context.Context, api *tg.Client) error {
	_, err := api.AuthLogOut(ctx)
	return err
}
