package bitbucket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/config"
)

func newTestClient(t *testing.T, base string) *Client {
	t.Helper()
	baseURL, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse base: %v", err)
	}
	return &Client{
		BaseURL:    baseURL,
		HTTP:       http.DefaultClient,
		Token:      "secret",
		Username:   "me@example.com",
		AuthMethod: config.AuthBasic,
		Kind:       config.KindCloud,
		UserAgent:  "bk-test",
	}
}

func TestGetUsesBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "me@example.com" || pass != "secret" {
			t.Errorf("unexpected basic auth: %q %q %v", user, pass, ok)
		}
		if r.URL.Path != "/repositories/acme/widgets" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"slug":"widgets","full_name":"acme/widgets"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	var repo Repository
	if err := client.Get(context.Background(), "/repositories/acme/widgets", nil, &repo); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if repo.Slug != "widgets" {
		t.Fatalf("slug = %q", repo.Slug)
	}
}

func TestBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer pat" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"slug":"widgets"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	client.AuthMethod = config.AuthBearer
	client.Username = ""
	client.Token = "pat"
	var repo Repository
	if err := client.Get(context.Background(), "/repositories/acme/widgets", nil, &repo); err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestPaginationCloudFollowsNext(t *testing.T) {
	var requests int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		if r.URL.Path != "/items" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("page") == "2" {
			_, _ = w.Write([]byte(`{"values":[{"id":2}]}`))
			return
		}
		if r.URL.Query().Get("pagelen") != "100" {
			t.Errorf("pagelen = %q", r.URL.Query().Get("pagelen"))
		}
		_, _ = w.Write([]byte(`{"values":[{"id":1}],"next":"` + server.URL + `/items?page=2"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	items, err := GetAll[struct {
		ID int `json:"id"`
	}](context.Background(), client, "/items", nil, 0)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 {
		t.Fatalf("items = %+v", items)
	}
	if atomic.LoadInt32(&requests) != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestPaginationDataCenterUsesNextPageStart(t *testing.T) {
	var starts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		starts = append(starts, r.URL.Query().Get("start"))
		if r.URL.Query().Get("start") == "2" {
			_, _ = w.Write([]byte(`{"values":[{"id":2}],"isLastPage":true}`))
			return
		}
		_, _ = w.Write([]byte(`{"values":[{"id":1}],"isLastPage":false,"nextPageStart":2}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	client.Kind = config.KindDataCenter
	items, err := GetAll[struct {
		ID int `json:"id"`
	}](context.Background(), client, "/repos", nil, 0)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if len(starts) != 2 || starts[1] != "2" {
		t.Fatalf("starts = %v", starts)
	}
}

func TestHTTPErrorParsingAndHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "abc123")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Forbidden","detail":"Token lacks scope"}}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	var repo Repository
	err := client.Get(context.Background(), "/repositories/acme/widgets", nil, &repo)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsForbidden(err) {
		t.Fatalf("expected 403, got %v", err)
	}
	if !strings.Contains(err.Error(), "Token lacks scope") {
		t.Fatalf("error = %v", err)
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("not an HTTPError: %v", err)
	}
	if httpErr.RequestID != "abc123" {
		t.Fatalf("request id = %q", httpErr.RequestID)
	}
	if hint := httpErr.Hint(); !strings.Contains(hint, "scopes") {
		t.Fatalf("hint = %q", hint)
	}
}

func TestRetryOnRateLimit(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"slug":"widgets"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	var repo Repository
	if err := client.Get(context.Background(), "/repositories/acme/widgets", nil, &repo); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestRepoPathShapes(t *testing.T) {
	cloud := newTestClient(t, "https://api.bitbucket.org/2.0")
	if got := cloud.RepoPath("acme", "widgets"); got != "/repositories/acme/widgets" {
		t.Fatalf("cloud path = %q", got)
	}
	dc := newTestClient(t, "https://dc.example.com/rest/api/1.0")
	dc.Kind = config.KindDataCenter
	if got := dc.RepoPath("PROJ", "widgets"); got != "/projects/PROJ/repos/widgets" {
		t.Fatalf("dc path = %q", got)
	}
}

func TestGetRawUsesTextAcceptAndFollowsRedirect(t *testing.T) {
	var storageAuth, storageAccept string
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		storageAuth = r.Header.Get("Authorization")
		storageAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte("step log"))
	}))
	defer storage.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "text/plain" {
			t.Errorf("api Accept = %q, want text/plain", got)
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		http.Redirect(w, r, storage.URL+"/blob", http.StatusTemporaryRedirect)
	}))
	defer api.Close()

	client := newTestClient(t, api.URL)
	data, err := client.GetRaw(context.Background(), "/log", nil, "text/plain")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(data) != "step log" {
		t.Fatalf("data = %q", data)
	}
	if storageAuth != "" {
		t.Fatalf("authorization leaked to storage: %q", storageAuth)
	}
	if storageAccept != "*/*" {
		t.Fatalf("storage Accept = %q, want */*", storageAccept)
	}
}

func TestGetRawRetriesWildcardOn406(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if r.Header.Get("Accept") != "*/*" {
			w.WriteHeader(http.StatusNotAcceptable)
			return
		}
		_, _ = w.Write([]byte("log body"))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	data, err := client.GetRaw(context.Background(), "/log", nil, "text/plain")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if string(data) != "log body" {
		t.Fatalf("data = %q", data)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestCreateSnippetUsesMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/snippets/acme" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return
		}
		if got := r.FormValue("title"); got != "Example" {
			t.Errorf("title = %q", got)
		}
		if got := r.FormValue("is_private"); got != "true" {
			t.Errorf("is_private = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("form file: %v", err)
			return
		}
		defer file.Close()
		if header.Filename != "hello.txt" {
			t.Errorf("filename = %q", header.Filename)
		}
		data := make([]byte, 64)
		n, _ := file.Read(data)
		if string(data[:n]) != "hi" {
			t.Errorf("content = %q", string(data[:n]))
		}
		_, _ = w.Write([]byte(`{"id":1,"title":"Example"}`))
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	snippet, err := client.CreateSnippet(context.Background(), "acme", CreateSnippetInput{
		Title:     "Example",
		IsPrivate: true,
		Files:     map[string]string{"hello.txt": "hi"},
	})
	if err != nil {
		t.Fatalf("CreateSnippet: %v", err)
	}
	if snippet.Title != "Example" {
		t.Fatalf("snippet = %+v", snippet)
	}
}

func TestFlexStringAcceptsStringAndNumber(t *testing.T) {
	var snippet Snippet
	if err := json.Unmarshal([]byte(`{"id":42,"title":"x"}`), &snippet); err != nil {
		t.Fatalf("unmarshal number: %v", err)
	}
	if snippet.ID.String() != "42" {
		t.Fatalf("id = %q", snippet.ID.String())
	}
	if err := json.Unmarshal([]byte(`{"id":"42","title":"x"}`), &snippet); err != nil {
		t.Fatalf("unmarshal string: %v", err)
	}
	if snippet.ID.String() != "42" {
		t.Fatalf("id = %q", snippet.ID.String())
	}
}
