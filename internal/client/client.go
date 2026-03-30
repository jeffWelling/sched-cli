package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jeff/sched-cli/internal/store"
)

// defaultRetryDelay is the duration to wait before retrying a failed request.
var defaultRetryDelay = 2 * time.Second

// CookieSet holds the auth cookies needed for Sched API requests.
type CookieSet struct {
	Token    string
	UContext string
}

// EventSetResponse represents the response from /event-set add/del.
type EventSetResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Raw     string `json:"-"`
}

// Client handles HTTP communication with Sched.com.
type Client struct {
	eventURL   string // e.g., "https://srecon26americas.sched.com"
	cookies    CookieSet
	httpClient *http.Client
	retryDelay time.Duration
}

// New creates a Client for the given event URL with auth cookies.
func New(eventURL string, cookies CookieSet) *Client {
	return &Client{
		eventURL:   strings.TrimRight(eventURL, "/"),
		cookies:    cookies,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		retryDelay: defaultRetryDelay,
	}
}

// NewWithHTTPClient creates a Client with a custom http.Client (for testing).
func NewWithHTTPClient(eventURL string, cookies CookieSet, httpClient *http.Client) *Client {
	return &Client{
		eventURL:   strings.TrimRight(eventURL, "/"),
		cookies:    cookies,
		httpClient: httpClient,
		retryDelay: defaultRetryDelay,
	}
}

// FetchAllSessions fetches and parses /all.ics for the complete event schedule.
func (c *Client) FetchAllSessions() ([]store.Session, error) {
	data, err := c.get("/all.ics")
	if err != nil {
		return nil, fmt.Errorf("fetching all.ics: %w", err)
	}
	return ParseICalFeed(data, "")
}

// FetchUserSchedule fetches and parses /{username}.ics for a user's schedule.
func (c *Client) FetchUserSchedule(username string) ([]store.Session, error) {
	data, err := c.get("/" + username + ".ics")
	if err != nil {
		return nil, fmt.Errorf("fetching %s.ics: %w", username, err)
	}
	return ParseICalFeed(data, username)
}

// AddToSchedule adds sessions to the user's Sched schedule.
func (c *Client) AddToSchedule(hexIDs ...string) (*EventSetResponse, error) {
	params := url.Values{}
	for _, id := range hexIDs {
		params.Add("add", id)
	}
	data, err := c.get("/event-set?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("adding to schedule: %w", err)
	}
	return parseEventSetResponse(data), nil
}

// RemoveFromSchedule removes sessions from the user's Sched schedule.
func (c *Client) RemoveFromSchedule(hexIDs ...string) (*EventSetResponse, error) {
	params := url.Values{}
	for _, id := range hexIDs {
		params.Add("del", id)
	}
	data, err := c.get("/event-set?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("removing from schedule: %w", err)
	}
	return parseEventSetResponse(data), nil
}

// Login performs a POST /login to sched.com and returns the auth cookies.
func Login(email, password string) (*CookieSet, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow redirects
		},
	}

	form := url.Values{
		"username": {email},
		"password": {password},
		"login":    {""},
	}

	resp, err := client.PostForm("https://sched.com/login", form)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return nil, fmt.Errorf("unexpected login response: %d", resp.StatusCode)
	}

	var cs CookieSet
	for _, cookie := range resp.Cookies() {
		switch cookie.Name {
		case "token":
			cs.Token = cookie.Value
		case "ucontext":
			cs.UContext = cookie.Value
		}
	}

	if cs.Token == "" {
		return nil, fmt.Errorf("login succeeded but no token cookie received")
	}

	return &cs, nil
}

func (c *Client) get(path string) ([]byte, error) {
	data, err, statusCode := c.doGet(path)
	if err == nil {
		return data, nil
	}

	// Retry once if the error is retryable.
	if isRetryable(err, statusCode) {
		time.Sleep(c.retryDelay)
		data, err, _ = c.doGet(path)
		return data, err
	}

	return nil, err
}

// doGet performs a single GET request and returns the body, error, and status code.
// Status code is 0 if the request failed before receiving a response.
func (c *Client) doGet(path string) ([]byte, error, int) {
	req, err := http.NewRequest("GET", c.eventURL+path, nil)
	if err != nil {
		return nil, err, 0
	}
	req.AddCookie(&http.Cookie{Name: "token", Value: c.cookies.Token})
	req.AddCookie(&http.Cookie{Name: "ucontext", Value: c.cookies.UContext})
	req.Header.Set("User-Agent", "sched-cli/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err, 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, path), resp.StatusCode
	}

	body, err := io.ReadAll(resp.Body)
	return body, err, resp.StatusCode
}

// isRetryable returns true for errors that warrant a retry:
// connection refused, timeouts, and HTTP 502/503/504.
func isRetryable(err error, statusCode int) bool {
	if statusCode == 502 || statusCode == 503 || statusCode == 504 {
		return true
	}

	if err == nil {
		return false
	}

	// net.Error covers timeouts and temporary network errors.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Connection refused shows up as *net.OpError.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	return false
}

func parseEventSetResponse(data []byte) *EventSetResponse {
	raw := strings.TrimSpace(string(data))
	resp := &EventSetResponse{Raw: raw}

	// Simple "normal" response
	if raw == `"normal"` || raw == "normal" {
		resp.Status = "normal"
		return resp
	}

	// TODO: parse JSON response for seat-limited sessions
	resp.Status = raw
	return resp
}
