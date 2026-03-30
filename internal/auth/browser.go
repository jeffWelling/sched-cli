package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jeff/sched-cli/internal/config"
)

// LoginWithBrowser starts a local HTTP server and waits for the user to
// authenticate via a browser bookmarklet that POSTs document.cookie to
// the callback endpoint. The server binds to 127.0.0.1 only and uses a
// random port. A state token prevents CSRF-style attacks against the
// loopback endpoint.
func (a *Authenticator) LoginWithBrowser(timeout time.Duration) (*AuthResult, error) {
	// Bind to loopback only on a random port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start loopback listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Generate a random state token (32 bytes → 64 hex chars).
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Print instructions.
	out := a.Output()
	bookmarklet := fmt.Sprintf(
		"javascript:void(fetch('http://127.0.0.1:%d/callback?state=%s',{method:'POST',body:document.cookie}))",
		port, state,
	)
	fmt.Fprintln(out, "Open this URL in your browser: https://sched.com/login")
	fmt.Fprintln(out, "After logging in, click this bookmarklet or paste it into the address bar:")
	fmt.Fprintln(out, bookmarklet)

	// Channel to receive the result from the callback handler.
	type callbackResult struct {
		result *AuthResult
		err    error
	}
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Validate state token.
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid state token", http.StatusBadRequest)
			return
		}

		// Read the POST body (document.cookie string).
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}

		cookies := parseCookieString(string(body))
		token := cookies["token"]
		if token == "" {
			http.Error(w, "no token cookie found", http.StatusBadRequest)
			return
		}

		// Allow cross-origin requests from sched.com.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "Authentication successful! You can close this tab.")

		resultCh <- callbackResult{
			result: &AuthResult{
				Token:    token,
				UContext: cookies["ucontext"],
				Method:   config.AuthBrowser,
			},
		}
	})

	// Handle CORS preflight requests.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{Handler: mux}

	// Start serving in a goroutine.
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			resultCh <- callbackResult{err: fmt.Errorf("server error: %w", err)}
		}
	}()

	// Signal that the server is ready (used by tests to synchronize).
	if a.readyCh != nil {
		close(a.readyCh)
	}

	// Wait for the callback or timeout.
	select {
	case cr := <-resultCh:
		// Shut down the server gracefully.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		return cr.result, cr.err
	case <-time.After(timeout):
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		return nil, fmt.Errorf("browser login timed out after %s", timeout)
	}
}

// parseCookieString parses a cookie header string like
// "token=abc; ucontext=%7B%22uid%22...%7D; SCHEDsession=xyz"
// into a map of name→value pairs. It splits on "; " then on the first "="
// per pair, so values containing "=" are preserved.
func parseCookieString(raw string) map[string]string {
	cookies := make(map[string]string)
	if raw == "" {
		return cookies
	}
	for _, part := range strings.Split(raw, "; ") {
		idx := strings.Index(part, "=")
		if idx < 0 {
			continue
		}
		name := part[:idx]
		value := part[idx+1:]
		cookies[name] = value
	}
	return cookies
}
