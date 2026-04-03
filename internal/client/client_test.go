package client

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// --- Minimal iCal fixtures ---

const validICalFeed = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Sched//Test//EN
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Understanding eBPF
LOCATION:Room 101
CATEGORIES:Networking
UID:abc123def456
URL:https://srecon26.sched.com/event/abc123def456
END:VEVENT
BEGIN:VEVENT
DTSTART:20260324T160000Z
DTEND:20260324T170000Z
SUMMARY:Capacity Planning at Scale
LOCATION:Room 202
CATEGORIES:SRE
UID:789fed012345
URL:https://srecon26.sched.com/event/cplan
END:VEVENT
END:VCALENDAR
`

const userScheduleICalFeed = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
DTSTART:20260324T143000Z
DTEND:20260324T153000Z
SUMMARY:Understanding eBPF
UID:abc123def456@jsmith
URL:https://srecon26.sched.com/event/abc123def456
END:VEVENT
END:VCALENDAR
`

// --- Client Construction ---

func TestNew_SetsEventURL(t *testing.T) {
	c := New("https://srecon26.sched.com", CookieSet{Token: "tok", UContext: "ctx"})
	if c.eventURL != "https://srecon26.sched.com" {
		t.Errorf("eventURL = %q, want %q", c.eventURL, "https://srecon26.sched.com")
	}
}

func TestNew_StripsTrailingSlash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://srecon26.sched.com/", "https://srecon26.sched.com"},
		{"https://srecon26.sched.com///", "https://srecon26.sched.com"},
		{"https://srecon26.sched.com", "https://srecon26.sched.com"},
	}
	for _, tt := range tests {
		c := New(tt.input, CookieSet{})
		if c.eventURL != tt.want {
			t.Errorf("New(%q).eventURL = %q, want %q", tt.input, c.eventURL, tt.want)
		}
	}
}

func TestNewWithHTTPClient_UsesProvidedClient(t *testing.T) {
	custom := &http.Client{Timeout: 99 * time.Second}
	c := NewWithHTTPClient("https://example.sched.com", CookieSet{}, custom)
	if c.httpClient != custom {
		t.Error("expected NewWithHTTPClient to use the provided http.Client")
	}
}

func TestNewWithHTTPClient_StripsTrailingSlash(t *testing.T) {
	c := NewWithHTTPClient("https://example.sched.com/", CookieSet{}, &http.Client{})
	if c.eventURL != "https://example.sched.com" {
		t.Errorf("eventURL = %q, want trailing slash stripped", c.eventURL)
	}
}

// --- Request Mechanics ---

func TestGet_IncludesAuthCookies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCookie, err := r.Cookie("token")
		if err != nil {
			t.Errorf("missing token cookie: %v", err)
			http.Error(w, "missing token", 400)
			return
		}
		if tokenCookie.Value != "my-token" {
			t.Errorf("token = %q, want %q", tokenCookie.Value, "my-token")
		}

		uctxCookie, err := r.Cookie("ucontext")
		if err != nil {
			t.Errorf("missing ucontext cookie: %v", err)
			http.Error(w, "missing ucontext", 400)
			return
		}
		if uctxCookie.Value != "my-context" {
			t.Errorf("ucontext = %q, want %q", uctxCookie.Value, "my-context")
		}

		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "my-token", UContext: "my-context"}, srv.Client())
	_, err := c.get("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet_IncludesUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "Mozilla/5.0 (compatible; sched-cli/1.0; +https://github.com/jeff/sched-cli)" {
			t.Errorf("User-Agent = %q, want %q", ua, "Mozilla/5.0 (compatible; sched-cli/1.0; +https://github.com/jeff/sched-cli)")
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	_, err := c.get("/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGet_RequestsCorrectURL(t *testing.T) {
	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.RequestURI()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	_, err := c.get("/all.ics")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestedPath != "/all.ics" {
		t.Errorf("request path = %q, want %q", requestedPath, "/all.ics")
	}
}

// --- FetchAllSessions ---

func TestFetchAllSessions_ParsesValidICalResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/all.ics" {
			t.Errorf("path = %q, want /all.ics", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(validICalFeed))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	sessions, err := c.FetchAllSessions()
	if err != nil {
		t.Fatalf("FetchAllSessions() error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].Title != "Understanding eBPF" {
		t.Errorf("sessions[0].Title = %q, want %q", sessions[0].Title, "Understanding eBPF")
	}
	if sessions[0].HexID != "abc123def456" {
		t.Errorf("sessions[0].HexID = %q, want %q", sessions[0].HexID, "abc123def456")
	}
	if sessions[0].Location != "Room 101" {
		t.Errorf("sessions[0].Location = %q, want %q", sessions[0].Location, "Room 101")
	}
	if sessions[0].Category != "Networking" {
		t.Errorf("sessions[0].Category = %q, want %q", sessions[0].Category, "Networking")
	}
	if sessions[1].Title != "Capacity Planning at Scale" {
		t.Errorf("sessions[1].Title = %q, want %q", sessions[1].Title, "Capacity Planning at Scale")
	}
	// Second session has a short URL so it gets a ShortID
	if sessions[1].ShortID != "cplan" {
		t.Errorf("sessions[1].ShortID = %q, want %q", sessions[1].ShortID, "cplan")
	}
}

func TestFetchAllSessions_ErrorOnHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	_, err := c.FetchAllSessions()
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q should mention status 500", err.Error())
	}
}

func TestFetchAllSessions_ErrorOnNon200(t *testing.T) {
	codes := []int{301, 403, 404, 502, 503}
	for _, code := range codes {
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer srv.Close()

			c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
			_, err := c.FetchAllSessions()
			if err == nil {
				t.Fatalf("expected error on HTTP %d, got nil", code)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", code)) {
				t.Errorf("error %q should mention status %d", err.Error(), code)
			}
		})
	}
}

// --- FetchUserSchedule ---

func TestFetchUserSchedule_FetchesCorrectPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jsmith.ics" {
			t.Errorf("path = %q, want /jsmith.ics", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(userScheduleICalFeed))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	sessions, err := c.FetchUserSchedule("jsmith")
	if err != nil {
		t.Fatalf("FetchUserSchedule() error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions, want 1", len(sessions))
	}
	// UID "abc123def456@jsmith" should be parsed to just the hex part
	if sessions[0].HexID != "abc123def456" {
		t.Errorf("HexID = %q, want %q (@ username stripped)", sessions[0].HexID, "abc123def456")
	}
	if sessions[0].Title != "Understanding eBPF" {
		t.Errorf("Title = %q, want %q", sessions[0].Title, "Understanding eBPF")
	}
}

// --- AddToSchedule ---

func TestAddToSchedule_SingleHexID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event-set" {
			t.Errorf("path = %q, want /event-set", r.URL.Path)
		}
		addValues := r.URL.Query()["add"]
		if len(addValues) != 1 {
			t.Errorf("expected 1 add param, got %d", len(addValues))
		}
		if addValues[0] != "abc123" {
			t.Errorf("add = %q, want %q", addValues[0], "abc123")
		}
		w.WriteHeader(200)
		w.Write([]byte(`"normal"`))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	resp, err := c.AddToSchedule("abc123")
	if err != nil {
		t.Fatalf("AddToSchedule() error: %v", err)
	}
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

func TestAddToSchedule_BatchMultipleHexIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addValues := r.URL.Query()["add"]
		if len(addValues) != 3 {
			t.Fatalf("expected 3 add params, got %d: %v", len(addValues), addValues)
		}
		// Verify repeated add= params, NOT comma-separated
		expected := []string{"aaa", "bbb", "ccc"}
		for i, want := range expected {
			if addValues[i] != want {
				t.Errorf("add[%d] = %q, want %q", i, addValues[i], want)
			}
		}
		// Verify the raw query does NOT contain commas in add values
		rawQuery := r.URL.RawQuery
		if strings.Contains(rawQuery, "add=aaa%2Cbbb") || strings.Contains(rawQuery, "add=aaa,bbb") {
			t.Errorf("batch should use repeated add= params, not comma-separated; query = %s", rawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(`"normal"`))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	resp, err := c.AddToSchedule("aaa", "bbb", "ccc")
	if err != nil {
		t.Fatalf("AddToSchedule() error: %v", err)
	}
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

func TestAddToSchedule_HandlesNormalResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`"normal"`))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	resp, err := c.AddToSchedule("abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
	if resp.Raw != `"normal"` {
		t.Errorf("Raw = %q, want %q", resp.Raw, `"normal"`)
	}
}

// --- RemoveFromSchedule ---

func TestRemoveFromSchedule_SingleHexID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event-set" {
			t.Errorf("path = %q, want /event-set", r.URL.Path)
		}
		delValues := r.URL.Query()["del"]
		if len(delValues) != 1 {
			t.Errorf("expected 1 del param, got %d", len(delValues))
		}
		if delValues[0] != "abc123" {
			t.Errorf("del = %q, want %q", delValues[0], "abc123")
		}
		w.WriteHeader(200)
		w.Write([]byte(`"normal"`))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	resp, err := c.RemoveFromSchedule("abc123")
	if err != nil {
		t.Fatalf("RemoveFromSchedule() error: %v", err)
	}
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

func TestRemoveFromSchedule_BatchMultipleHexIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delValues := r.URL.Query()["del"]
		if len(delValues) != 2 {
			t.Fatalf("expected 2 del params, got %d: %v", len(delValues), delValues)
		}
		if delValues[0] != "xxx" || delValues[1] != "yyy" {
			t.Errorf("del params = %v, want [xxx, yyy]", delValues)
		}
		w.WriteHeader(200)
		w.Write([]byte("normal"))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	resp, err := c.RemoveFromSchedule("xxx", "yyy")
	if err != nil {
		t.Fatalf("RemoveFromSchedule() error: %v", err)
	}
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

// --- EventSetResponse Parsing ---

func TestParseEventSetResponse_QuotedNormal(t *testing.T) {
	resp := parseEventSetResponse([]byte(`"normal"`))
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
	if resp.Raw != `"normal"` {
		t.Errorf("Raw = %q, want %q", resp.Raw, `"normal"`)
	}
}

func TestParseEventSetResponse_UnquotedNormal(t *testing.T) {
	resp := parseEventSetResponse([]byte("normal"))
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

func TestParseEventSetResponse_NormalWithWhitespace(t *testing.T) {
	resp := parseEventSetResponse([]byte("  \"normal\"  \n"))
	if resp.Status != "normal" {
		t.Errorf("Status = %q, want %q", resp.Status, "normal")
	}
}

func TestParseEventSetResponse_SeatStatus(t *testing.T) {
	// Non-"normal" responses are stored as-is in Status
	resp := parseEventSetResponse([]byte(`{"status":"full","seats_remaining":0}`))
	if resp.Status != `{"status":"full","seats_remaining":0}` {
		t.Errorf("Status = %q, want raw JSON string", resp.Status)
	}
}

func TestParseEventSetResponse_RawFieldAlwaysSet(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"quoted normal", `"normal"`},
		{"unquoted normal", "normal"},
		{"seat status", `{"status":"full"}`},
		{"unknown response", "something unexpected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := parseEventSetResponse([]byte(tt.input))
			if resp.Raw != tt.input {
				t.Errorf("Raw = %q, want %q", resp.Raw, tt.input)
			}
		})
	}
}

// --- Login Function ---

// loginWithURL duplicates the Login() logic but accepts a custom URL for testing.
// This avoids hitting the hardcoded https://sched.com/login.
func loginWithURL(loginURL, email, password string) (*CookieSet, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	form := url.Values{
		"username": {email},
		"password": {password},
		"login":    {""},
	}

	resp, err := client.PostForm(loginURL, form)
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

func TestLogin_SendsCorrectFormData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want POST", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
			t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", contentType)
		}

		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))

		if form.Get("username") != "user@example.com" {
			t.Errorf("username = %q, want %q", form.Get("username"), "user@example.com")
		}
		if form.Get("password") != "s3cret" {
			t.Errorf("password = %q, want %q", form.Get("password"), "s3cret")
		}
		if _, ok := form["login"]; !ok {
			t.Error("missing 'login' field in form data")
		}

		http.SetCookie(w, &http.Cookie{Name: "token", Value: "tok123"})
		http.SetCookie(w, &http.Cookie{Name: "ucontext", Value: "ctx456"})
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	cs, err := loginWithURL(srv.URL+"/login", "user@example.com", "s3cret")
	if err != nil {
		t.Fatalf("loginWithURL() error: %v", err)
	}
	if cs.Token != "tok123" {
		t.Errorf("Token = %q, want %q", cs.Token, "tok123")
	}
	if cs.UContext != "ctx456" {
		t.Errorf("UContext = %q, want %q", cs.UContext, "ctx456")
	}
}

func TestLogin_ExtractsTokenCookie(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "token", Value: "the-token-value"})
		http.SetCookie(w, &http.Cookie{Name: "ucontext", Value: "the-ucontext-value"})
		http.SetCookie(w, &http.Cookie{Name: "irrelevant", Value: "ignored"})
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	cs, err := loginWithURL(srv.URL, "e@x.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Token != "the-token-value" {
		t.Errorf("Token = %q, want %q", cs.Token, "the-token-value")
	}
	if cs.UContext != "the-ucontext-value" {
		t.Errorf("UContext = %q, want %q", cs.UContext, "the-ucontext-value")
	}
}

func TestLogin_ErrorWhenNoTokenCookie(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 302 but no token cookie
		http.SetCookie(w, &http.Cookie{Name: "ucontext", Value: "ctx"})
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	_, err := loginWithURL(srv.URL, "e@x.com", "pw")
	if err == nil {
		t.Fatal("expected error when no token cookie, got nil")
	}
	if !strings.Contains(err.Error(), "no token cookie") {
		t.Errorf("error %q should mention missing token cookie", err.Error())
	}
}

func TestLogin_ErrorOnNon302Response(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{"200 OK", 200},
		{"401 Unauthorized", 401},
		{"403 Forbidden", 403},
		{"500 Internal Server Error", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.code)
			}))
			defer srv.Close()

			_, err := loginWithURL(srv.URL, "e@x.com", "pw")
			if err == nil {
				t.Fatalf("expected error on HTTP %d, got nil", tt.code)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", tt.code)) {
				t.Errorf("error %q should mention status %d", err.Error(), tt.code)
			}
		})
	}
}

func TestLogin_DoesNotFollowRedirects(t *testing.T) {
	redirectCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		if redirectCount == 1 {
			http.SetCookie(w, &http.Cookie{Name: "token", Value: "tok"})
			http.SetCookie(w, &http.Cookie{Name: "ucontext", Value: "ctx"})
			w.Header().Set("Location", "/dashboard")
			w.WriteHeader(http.StatusFound)
			return
		}
		// If we get here, a redirect was followed
		t.Error("login should not follow redirects")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	_, err := loginWithURL(srv.URL, "e@x.com", "pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if redirectCount != 1 {
		t.Errorf("expected exactly 1 request (no redirect following), got %d", redirectCount)
	}
}

// --- Error Handling ---

func TestGet_NetworkTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang for longer than client timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	shortTimeout := &http.Client{Timeout: 50 * time.Millisecond}
	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, shortTimeout)
	_, err := c.FetchAllSessions()
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestGet_InvalidURL(t *testing.T) {
	c := NewWithHTTPClient("://not-a-url", CookieSet{Token: "t", UContext: "u"}, &http.Client{})
	_, err := c.FetchAllSessions()
	if err == nil {
		t.Fatal("expected error on invalid URL, got nil")
	}
}

// --- Retry Logic ---

func TestGet_RetriesOnServerError(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.WriteHeader(503)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	c.retryDelay = 10 * time.Millisecond // fast retry for tests

	data, err := c.get("/test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if string(data) != "ok" {
		t.Errorf("body = %q, want %q", string(data), "ok")
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests (1 fail + 1 retry), got %d", requestCount)
	}
}

func TestGet_DoesNotRetryOn404(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	c.retryDelay = 10 * time.Millisecond

	_, err := c.get("/missing")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if requestCount != 1 {
		t.Errorf("expected exactly 1 request (no retry on 404), got %d", requestCount)
	}
}

func TestGet_RetriesOnTimeout(t *testing.T) {
	var requestCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count == 1 {
			// Hang to trigger timeout on first request.
			time.Sleep(500 * time.Millisecond)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	// Use a very short timeout so the first request times out.
	shortTimeout := &http.Client{Timeout: 50 * time.Millisecond}
	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, shortTimeout)
	c.retryDelay = 10 * time.Millisecond

	data, err := c.get("/test")
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if string(data) != "recovered" {
		t.Errorf("body = %q, want %q", string(data), "recovered")
	}
	if requestCount != 2 {
		t.Errorf("expected 2 requests (1 timeout + 1 retry), got %d", requestCount)
	}
}

// --- Integration-style: full request verification ---

func TestAddToSchedule_VerifiesFullRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify method
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		// Verify cookies
		if _, err := r.Cookie("token"); err != nil {
			t.Errorf("missing token cookie: %v", err)
		}
		if _, err := r.Cookie("ucontext"); err != nil {
			t.Errorf("missing ucontext cookie: %v", err)
		}
		// Verify User-Agent
		if r.Header.Get("User-Agent") != "Mozilla/5.0 (compatible; sched-cli/1.0; +https://github.com/jeff/sched-cli)" {
			t.Errorf("User-Agent = %q, want Mozilla/5.0 (compatible; sched-cli/1.0; +https://github.com/jeff/sched-cli)", r.Header.Get("User-Agent"))
		}
		// Verify path and query
		if r.URL.Path != "/event-set" {
			t.Errorf("path = %q, want /event-set", r.URL.Path)
		}
		addVals := r.URL.Query()["add"]
		if len(addVals) != 1 || addVals[0] != "deadbeef" {
			t.Errorf("add params = %v, want [deadbeef]", addVals)
		}

		w.WriteHeader(200)
		w.Write([]byte(`"normal"`))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	_, err := c.AddToSchedule("deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveFromSchedule_VerifiesFullRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.URL.Path != "/event-set" {
			t.Errorf("path = %q, want /event-set", r.URL.Path)
		}
		delVals := r.URL.Query()["del"]
		if len(delVals) != 1 || delVals[0] != "cafebabe" {
			t.Errorf("del params = %v, want [cafebabe]", delVals)
		}

		w.WriteHeader(200)
		w.Write([]byte("normal"))
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, CookieSet{Token: "t", UContext: "u"}, srv.Client())
	_, err := c.RemoveFromSchedule("cafebabe")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
