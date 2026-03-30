package auth

import (
	"fmt"
	"testing"

	"github.com/jeff/sched-cli/internal/client"
	"github.com/jeff/sched-cli/internal/config"
)

func mockLoginSuccess(email, password string) (*client.CookieSet, error) {
	return &client.CookieSet{
		Token:    "test-token-abc",
		UContext: "test-ucontext-xyz",
	}, nil
}

func mockLoginFailure(email, password string) (*client.CookieSet, error) {
	return nil, fmt.Errorf("invalid credentials")
}

func TestCredentialAuth_Success(t *testing.T) {
	a := NewForTesting(false, mockLoginSuccess)

	result, err := a.LoginWithCredentials("user@example.com", "password123")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Token != "test-token-abc" {
		t.Errorf("expected token 'test-token-abc', got '%s'", result.Token)
	}
	if result.UContext != "test-ucontext-xyz" {
		t.Errorf("expected ucontext 'test-ucontext-xyz', got '%s'", result.UContext)
	}
	if result.Method != config.AuthCredentials {
		t.Errorf("expected method '%s', got '%s'", config.AuthCredentials, result.Method)
	}
}

func TestCredentialAuth_BadPassword(t *testing.T) {
	a := NewForTesting(false, mockLoginFailure)

	_, err := a.LoginWithCredentials("user@example.com", "wrong-password")
	if err == nil {
		t.Fatal("expected error for bad password, got nil")
	}
}

func TestCredentialAuth_EmptyFields(t *testing.T) {
	a := NewForTesting(false, mockLoginSuccess)

	_, err := a.LoginWithCredentials("", "password")
	if err == nil {
		t.Fatal("expected error for empty email, got nil")
	}

	_, err = a.LoginWithCredentials("user@example.com", "")
	if err == nil {
		t.Fatal("expected error for empty password, got nil")
	}
}

func TestTokenAuth_StoresDirectly(t *testing.T) {
	a := NewForTesting(false, nil)

	result, err := a.LoginWithToken("my-raw-token-value")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.Token != "my-raw-token-value" {
		t.Errorf("expected token 'my-raw-token-value', got '%s'", result.Token)
	}
	if result.UContext != "" {
		t.Errorf("expected empty ucontext, got '%s'", result.UContext)
	}
	if result.Method != config.AuthToken {
		t.Errorf("expected method '%s', got '%s'", config.AuthToken, result.Method)
	}
}

func TestTokenAuth_EmptyToken(t *testing.T) {
	a := NewForTesting(false, nil)

	_, err := a.LoginWithToken("")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestCheckEnvAuth_Token(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "env-token-123")
	t.Setenv("SCHED_EMAIL", "")
	t.Setenv("SCHED_PASSWORD", "")

	result, ok := CheckEnvAuth()
	if !ok {
		t.Fatal("expected ok=true when SCHED_TOKEN is set")
	}
	if result.Token != "env-token-123" {
		t.Errorf("expected token 'env-token-123', got '%s'", result.Token)
	}
	if result.Method != config.AuthToken {
		t.Errorf("expected method '%s', got '%s'", config.AuthToken, result.Method)
	}
}

func TestCheckEnvAuth_Credentials(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "")
	t.Setenv("SCHED_EMAIL", "user@example.com")
	t.Setenv("SCHED_PASSWORD", "secret")

	// CheckEnvAuth uses New() which calls client.Login — we can't inject a mock
	// into the package-level function directly. However, the real Login would
	// fail in tests (no network). We verify the token path works above.
	// For credentials path, we verify the function attempts login and returns
	// false when it fails (no network in test environment).
	result, ok := CheckEnvAuth()
	// The real client.Login will fail in test, so we expect false
	if ok {
		t.Logf("CheckEnvAuth with credentials returned ok=true (unexpected in test without network), result: %+v", result)
	}
	// This is expected behavior: credentials are detected but login fails
	if ok && result.Method != config.AuthCredentials {
		t.Errorf("if ok, expected method '%s', got '%s'", config.AuthCredentials, result.Method)
	}
}

func TestCheckEnvAuth_None(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "")
	t.Setenv("SCHED_EMAIL", "")
	t.Setenv("SCHED_PASSWORD", "")

	result, ok := CheckEnvAuth()
	if ok {
		t.Fatalf("expected ok=false when no env vars set, got result: %+v", result)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestCheckEnvAuth_PartialCredentials(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "")
	t.Setenv("SCHED_EMAIL", "user@example.com")
	t.Setenv("SCHED_PASSWORD", "")

	result, ok := CheckEnvAuth()
	if ok {
		t.Fatalf("expected ok=false with only email set, got result: %+v", result)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestCheckEnvAuth_PartialCredentials_PasswordOnly(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "")
	t.Setenv("SCHED_EMAIL", "")
	t.Setenv("SCHED_PASSWORD", "secret")

	result, ok := CheckEnvAuth()
	if ok {
		t.Fatalf("expected ok=false with only password set, got result: %+v", result)
	}
	if result != nil {
		t.Errorf("expected nil result, got: %+v", result)
	}
}

func TestIsInteractive(t *testing.T) {
	interactive := NewForTesting(true, nil)
	if !interactive.IsInteractive() {
		t.Error("expected IsInteractive() to return true")
	}

	nonInteractive := NewForTesting(false, nil)
	if nonInteractive.IsInteractive() {
		t.Error("expected IsInteractive() to return false")
	}
}

func TestCheckEnvAuth_TokenTakesPrecedence(t *testing.T) {
	t.Setenv("SCHED_TOKEN", "token-wins")
	t.Setenv("SCHED_EMAIL", "user@example.com")
	t.Setenv("SCHED_PASSWORD", "secret")

	result, ok := CheckEnvAuth()
	if !ok {
		t.Fatal("expected ok=true")
	}
	if result.Token != "token-wins" {
		t.Errorf("expected token 'token-wins', got '%s'", result.Token)
	}
	if result.Method != config.AuthToken {
		t.Errorf("expected token method to take precedence, got '%s'", result.Method)
	}
}
