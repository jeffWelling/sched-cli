package auth

import (
	"fmt"
	"io"
	"os"

	"github.com/jeff/sched-cli/internal/client"
	"github.com/jeff/sched-cli/internal/config"
)

// AuthResult holds the outcome of any authentication method.
type AuthResult struct {
	Token    string
	UContext string
	Method   config.AuthMethod
}

// Authenticator handles multi-method authentication for Sched.com.
type Authenticator struct {
	isInteractive bool
	loginFn       func(email, password string) (*client.CookieSet, error)
	// output is the writer for user-facing messages. Defaults to os.Stdout.
	output io.Writer
	// readyCh is closed when the browser loopback server is ready to accept
	// connections. Nil in production; set in tests to synchronize.
	readyCh chan struct{}
}

// Output returns the writer used for user-facing messages.
func (a *Authenticator) Output() io.Writer {
	if a.output != nil {
		return a.output
	}
	return os.Stdout
}

// New creates an Authenticator that uses the real client.Login function.
func New(isInteractive bool) *Authenticator {
	return &Authenticator{
		isInteractive: isInteractive,
		loginFn:       client.Login,
	}
}

// NewForTesting creates an Authenticator with an injectable login function.
func NewForTesting(isInteractive bool, loginFn func(string, string) (*client.CookieSet, error)) *Authenticator {
	return &Authenticator{
		isInteractive: isInteractive,
		loginFn:       loginFn,
	}
}

// LoginWithCredentials authenticates using email and password via the Sched login form.
func (a *Authenticator) LoginWithCredentials(email, password string) (*AuthResult, error) {
	if email == "" || password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	cookies, err := a.loginFn(email, password)
	if err != nil {
		return nil, fmt.Errorf("credential login failed: %w", err)
	}

	return &AuthResult{
		Token:    cookies.Token,
		UContext: cookies.UContext,
		Method:   config.AuthCredentials,
	}, nil
}

// LoginWithToken wraps a raw token value into an AuthResult.
// The token is used directly without server-side validation.
func (a *Authenticator) LoginWithToken(token string) (*AuthResult, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}

	return &AuthResult{
		Token:  token,
		Method: config.AuthToken,
	}, nil
}

// IsInteractive returns whether the authenticator is running in an interactive terminal.
func (a *Authenticator) IsInteractive() bool {
	return a.isInteractive
}

// CheckEnvAuth checks environment variables for authentication.
// It checks SCHED_TOKEN first, then SCHED_EMAIL + SCHED_PASSWORD.
// Returns nil and false if no env auth is available.
func CheckEnvAuth() (*AuthResult, bool) {
	if token := config.EnvToken(); token != "" {
		return &AuthResult{
			Token:  token,
			Method: config.AuthToken,
		}, true
	}

	if email, password, ok := config.EnvCredentials(); ok {
		a := New(false)
		result, err := a.LoginWithCredentials(email, password)
		if err != nil {
			return nil, false
		}
		return result, true
	}

	return nil, false
}
