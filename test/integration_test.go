package test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/cmdutil"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
	"github.com/ClaudioLazaro/bitbucket-cli/pkg/cmd/root"
)

type envMap map[string]string

func runCommand(t *testing.T, server *httptest.Server, overrides envMap, args ...string) (string, string, error) {
	t.Helper()
	streams, out, errOut, _ := iostreams.Test()
	values := envMap{
		"BK_CONFIG_DIR":  t.TempDir(),
		"BK_HOST":        "bitbucket.org",
		"BK_TOKEN":       "token",
		"BK_USERNAME":    "me@example.com",
		"BK_AUTH_METHOD": "basic",
		"BK_API_URL":     server.URL + "/2.0",
	}
	for key, value := range overrides {
		values[key] = value
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cmd := root.NewCmdRoot(cmdutil.NewFactory(streams))
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func newServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func runCommandWithInput(t *testing.T, server *httptest.Server, overrides envMap, stdin string, args ...string) (string, string, error) {
	t.Helper()
	streams, out, errOut, in := iostreams.Test()
	_, _ = in.WriteString(stdin)
	values := envMap{
		"BK_CONFIG_DIR":  t.TempDir(),
		"BK_HOST":        "bitbucket.org",
		"BK_API_URL":     server.URL + "/2.0",
		"BK_TOKEN":       "",
		"BK_USERNAME":    "",
		"BK_AUTH_METHOD": "",
	}
	for key, value := range overrides {
		values[key] = value
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cmd := root.NewCmdRoot(cmdutil.NewFactory(streams))
	cmd.SetOut(streams.Out)
	cmd.SetErr(streams.ErrOut)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, body)
}

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = io.WriteString(w, `{"error":{"message":"not found"}}`)
}

func TestRepoList(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/acme" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "me@example.com" || pass != "token" {
			t.Errorf("unexpected basic auth: %q %q %v", user, pass, ok)
		}
		writeJSON(w, `{"values":[{"slug":"widgets","full_name":"acme/widgets","is_private":true,"description":"Widgets","updated_on":"2026-01-02T03:04:05+00:00"}],"pagelen":1}`)
	})

	out, _, err := runCommand(t, server, nil, "repo", "list", "acme")
	if err != nil {
		t.Fatalf("repo list: %v", err)
	}
	if !strings.Contains(out, "widgets") || !strings.Contains(out, "private") {
		t.Fatalf("output = %q", out)
	}

	jsonOut, _, err := runCommand(t, server, nil, "repo", "list", "acme", "--json=slug,is_private")
	if err != nil {
		t.Fatalf("repo list --json: %v", err)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &items); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonOut, err)
	}
	if len(items) != 1 || items[0]["slug"] != "widgets" || items[0]["is_private"] != true {
		t.Fatalf("items = %+v", items)
	}
	if len(items[0]) != 2 {
		t.Fatalf("field filtering failed: %+v", items[0])
	}
}

func TestRepoView(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/acme/widgets" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		writeJSON(w, `{
			"slug":"widgets","full_name":"acme/widgets","is_private":false,"scm":"git",
			"description":"Widgets for everyone","mainbranch":{"name":"main"},
			"links":{
				"html":{"href":"https://bitbucket.org/acme/widgets"},
				"clone":[{"href":"https://bitbucket.org/acme/widgets.git","name":"https"}]
			}
		}`)
	})

	out, _, err := runCommand(t, server, nil, "repo", "view", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("repo view: %v", err)
	}
	for _, want := range []string{"acme/widgets", "Widgets for everyone", "Default branch: main", "https://bitbucket.org/acme/widgets"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPRList(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/acme/widgets/pullrequests" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		if state := r.URL.Query().Get("state"); state != "OPEN" {
			t.Errorf("state = %q", state)
		}
		writeJSON(w, `{"values":[{
			"id":7,"title":"Add widgets","state":"OPEN",
			"author":{"display_name":"Me"},
			"source":{"branch":{"name":"feature"}},
			"destination":{"branch":{"name":"main"}},
			"updated_on":"2026-01-01T00:00:00+00:00"
		}]}`)
	})

	out, _, err := runCommand(t, server, nil, "pr", "list", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pr list: %v", err)
	}
	if !strings.Contains(out, "7") || !strings.Contains(out, "Add widgets") || !strings.Contains(out, "open") {
		t.Fatalf("output = %q", out)
	}
}

func TestPRView(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/repositories/acme/widgets/pullrequests/7" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		writeJSON(w, `{
			"id":7,"title":"Add widgets","state":"OPEN","description":"Adds the widget list",
			"author":{"display_name":"Me"},
			"source":{"branch":{"name":"feature"}},
			"destination":{"branch":{"name":"main"}},
			"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/7"}}
		}`)
	})

	out, _, err := runCommand(t, server, nil, "pr", "view", "7", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pr view: %v", err)
	}
	for _, want := range []string{"#7", "Add widgets", "Adds the widget list", "https://bitbucket.org/acme/widgets/pull-requests/7"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPRCreate(t *testing.T) {
	var createdBody map[string]any
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets":
			writeJSON(w, `{"slug":"widgets","full_name":"acme/widgets","mainbranch":{"name":"main"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests":
			writeJSON(w, `{"values":[],"pagelen":1}`)
		case r.Method == http.MethodPost && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests":
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &createdBody); err != nil {
				t.Errorf("decode body: %v", err)
			}
			writeJSON(w, `{"id":9,"title":"Add widgets","state":"OPEN","links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/9"}}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pr", "create", "-R", "acme/widgets", "--head", "feature", "--title", "Add widgets", "--body", "Details")
	if err != nil {
		t.Fatalf("pr create: %v", err)
	}
	if !strings.Contains(out, "Created pull request #9") {
		t.Fatalf("output = %q", out)
	}
	if createdBody["title"] != "Add widgets" || createdBody["description"] != "Details" {
		t.Fatalf("body = %+v", createdBody)
	}
	source, _ := createdBody["source"].(map[string]any)
	branch, _ := source["branch"].(map[string]any)
	if branch["name"] != "feature" {
		t.Fatalf("source = %+v", createdBody["source"])
	}
	destination, _ := createdBody["destination"].(map[string]any)
	destBranch, _ := destination["branch"].(map[string]any)
	if destBranch["name"] != "main" {
		t.Fatalf("destination = %+v", createdBody["destination"])
	}
}

func TestPRMerge(t *testing.T) {
	var mergeBody map[string]any
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/9":
			writeJSON(w, `{"id":9,"title":"Add widgets","state":"OPEN","source":{"branch":{"name":"feature"}},"destination":{"branch":{"name":"main"}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/9/merge":
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &mergeBody); err != nil {
				t.Errorf("decode body: %v", err)
			}
			writeJSON(w, `{"id":9,"title":"Add widgets","state":"MERGED","links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/9"}}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pr", "merge", "9", "-R", "acme/widgets", "--strategy", "squash")
	if err != nil {
		t.Fatalf("pr merge: %v", err)
	}
	if !strings.Contains(out, "Merged pull request #9") {
		t.Fatalf("output = %q", out)
	}
	if mergeBody["merge_strategy"] != "squash" {
		t.Fatalf("merge body = %+v", mergeBody)
	}
}

func TestPRChecksFailedExitsOne(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/acme/widgets/pullrequests/9":
			writeJSON(w, `{"id":9,"state":"OPEN","source":{"branch":{"name":"feature"}},"destination":{"branch":{"name":"main"}}}`)
		case "/2.0/repositories/acme/widgets/pullrequests/9/statuses":
			writeJSON(w, `{"values":[{"key":"CI","name":"build","state":"FAILED","description":"unit tests"}]}`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pr", "checks", "9", "-R", "acme/widgets")
	if err == nil {
		t.Fatal("expected a non-zero exit error")
	}
	var exitErr *cmdutil.ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(out, "FAILED") {
		t.Fatalf("output = %q", out)
	}
}

func TestPRSummaryObject(t *testing.T) {
	const payload = `{
		"id": 8,
		"title": "Summary object",
		"state": "OPEN",
		"summary": {"raw": "Body from summary", "markup": "markdown", "html": "<p>Body from summary</p>"},
		"author": {"display_name": "Me"},
		"source": {"branch": {"name": "feature"}},
		"destination": {"branch": {"name": "main"}},
		"updated_on": "2026-01-01T00:00:00+00:00",
		"links": {"html": {"href": "https://bitbucket.org/acme/widgets/pull-requests/8"}}
	}`
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/acme/widgets/pullrequests":
			writeJSON(w, `{"values":[`+payload+`]}`)
		case "/2.0/repositories/acme/widgets/pullrequests/8":
			writeJSON(w, payload)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pr", "list", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pr list with object summary: %v", err)
	}
	if !strings.Contains(out, "Summary object") {
		t.Fatalf("output = %q", out)
	}

	view, _, err := runCommand(t, server, nil, "pr", "view", "8", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pr view with object summary: %v", err)
	}
	if !strings.Contains(view, "Body from summary") {
		t.Fatalf("view = %q", view)
	}

	jsonOut, _, err := runCommand(t, server, nil, "pr", "view", "8", "-R", "acme/widgets", "--json")
	if err != nil {
		t.Fatalf("pr view --json: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &decoded); err != nil {
		t.Fatalf("invalid JSON %q: %v", jsonOut, err)
	}
	if decoded["summary"] != "Body from summary" {
		t.Fatalf("summary JSON = %#v, want a string", decoded["summary"])
	}
}

func TestPipelineListAndLogs(t *testing.T) {
	const pipelineUUID = "{11111111-1111-1111-1111-111111111111}"
	const buildStepUUID = "{22222222-2222-2222-2222-222222222222}"
	const deployStepUUID = "{33333333-3333-3333-3333-333333333333}"
	const buildLogPath = "/storage/build.log"
	const deployLogPath = "/storage/deploy.log"

	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/acme/widgets/pipelines":
			writeJSON(w, `{"values":[{
				"uuid":"`+pipelineUUID+`","build_number":12,
				"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},
				"target":{"ref_name":"main"},
				"trigger":{"name":"push"},
				"created_on":"2026-01-01T00:00:00+00:00","build_seconds_used":90
			}]}`)
		case "/2.0/repositories/acme/widgets/pipelines/" + pipelineUUID:
			writeJSON(w, `{"uuid":"`+pipelineUUID+`","build_number":12,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"ref_name":"main"}}`)
		case "/2.0/repositories/acme/widgets/pipelines/" + pipelineUUID + "/steps":
			writeJSON(w, `{"values":[
				{"uuid":"`+buildStepUUID+`","name":"Build","state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}}},
				{"uuid":"`+deployStepUUID+`","name":"Deploy","state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}}}
			]}`)
		case "/2.0/repositories/acme/widgets/pipelines/" + pipelineUUID + "/steps/" + buildStepUUID + "/log":
			if got := r.Header.Get("Accept"); got != "text/plain" {
				t.Errorf("log Accept = %q, want text/plain", got)
				w.WriteHeader(http.StatusNotAcceptable)
				return
			}
			http.Redirect(w, r, buildLogPath, http.StatusTemporaryRedirect)
		case "/2.0/repositories/acme/widgets/pipelines/" + pipelineUUID + "/steps/" + deployStepUUID + "/log":
			if got := r.Header.Get("Accept"); got != "text/plain" {
				t.Errorf("log Accept = %q, want text/plain", got)
				w.WriteHeader(http.StatusNotAcceptable)
				return
			}
			http.Redirect(w, r, deployLogPath, http.StatusTemporaryRedirect)
		case buildLogPath:
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("authorization leaked to log storage: %q", got)
			}
			_, _ = io.WriteString(w, "build log\n")
		case deployLogPath:
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("authorization leaked to log storage: %q", got)
			}
			_, _ = io.WriteString(w, "deploy log\n")
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pipeline", "list", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pipeline list: %v", err)
	}
	if !strings.Contains(out, "12") || !strings.Contains(out, "SUCCESSFUL") {
		t.Fatalf("output = %q", out)
	}

	view, _, err := runCommand(t, server, nil, "pipeline", "view", "12", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pipeline view: %v", err)
	}
	if !strings.Contains(view, "Build") || !strings.Contains(view, "Deploy") {
		t.Fatalf("view = %q", view)
	}

	logs, _, err := runCommand(t, server, nil, "pipeline", "logs", "12", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pipeline logs: %v", err)
	}
	for _, want := range []string{"=== step 1: Build", "build log", "=== step 2: Deploy", "deploy log"} {
		if !strings.Contains(logs, want) {
			t.Fatalf("logs missing %q:\n%s", want, logs)
		}
	}

	buildLogs, _, err := runCommand(t, server, nil, "pipeline", "logs", "12", "-R", "acme/widgets", "--step", "Build")
	if err != nil {
		t.Fatalf("pipeline logs --step Build: %v", err)
	}
	if !strings.Contains(buildLogs, "build log") || strings.Contains(buildLogs, "deploy log") {
		t.Fatalf("build logs = %q", buildLogs)
	}

	deployLogs, _, err := runCommand(t, server, nil, "pipeline", "logs", "12", "-R", "acme/widgets", "--step", "2")
	if err != nil {
		t.Fatalf("pipeline logs --step 2: %v", err)
	}
	if !strings.Contains(deployLogs, "deploy log") || strings.Contains(deployLogs, "build log") {
		t.Fatalf("deploy logs = %q", deployLogs)
	}

	uuidLogs, _, err := runCommand(t, server, nil, "pipeline", "logs", "12", "-R", "acme/widgets", "--step", deployStepUUID)
	if err != nil {
		t.Fatalf("pipeline logs --step UUID: %v", err)
	}
	if !strings.Contains(uuidLogs, "deploy log") {
		t.Fatalf("uuid logs = %q", uuidLogs)
	}

	for _, selector := range []string{"Missing", "9"} {
		_, _, err := runCommand(t, server, nil, "pipeline", "logs", "12", "-R", "acme/widgets", "--step", selector)
		if err == nil {
			t.Fatalf("--step %s: expected an error", selector)
		}
	}
}

func TestAPIPassthrough(t *testing.T) {
	var hookBody map[string]any
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/user":
			writeJSON(w, `{"display_name":"Me","uuid":"{u-1}"}`)
		case r.Method == http.MethodPost && r.URL.Path == "/2.0/repositories/acme/widgets/hooks":
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &hookBody)
			writeJSON(w, `{"uuid":"{h-1}"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "api", "/user")
	if err != nil {
		t.Fatalf("api: %v", err)
	}
	if !strings.Contains(out, `"display_name": "Me"`) {
		t.Fatalf("output = %q", out)
	}

	_, _, err = runCommand(t, server, nil, "api", "POST", "/repositories/acme/widgets/hooks", "-f", "description=ci", "-f", "url=https://ci.example.com")
	if err != nil {
		t.Fatalf("api POST: %v", err)
	}
	if hookBody["description"] != "ci" || hookBody["url"] != "https://ci.example.com" {
		t.Fatalf("body = %+v", hookBody)
	}
}

func TestAuthStatus(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/user" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		writeJSON(w, `{"display_name":"Me","uuid":"{u-1}"}`)
	})

	out, _, err := runCommand(t, server, nil, "auth", "status")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if !strings.Contains(out, "bitbucket.org") || !strings.Contains(out, "ok (Me)") {
		t.Fatalf("output = %q", out)
	}
}

func TestDataCenterRepoList(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer pat" {
			t.Errorf("authorization = %q", got)
		}
		writeJSON(w, `{"values":[{"slug":"repo1","name":"repo1","project":{"key":"PROJ"}}],"isLastPage":true}`)
	})

	out, _, err := runCommand(t, server, envMap{
		"BK_HOST":        "https://dc.example.com",
		"BK_API_URL":     server.URL + "/rest/api/1.0",
		"BK_TOKEN":       "pat",
		"BK_USERNAME":    "",
		"BK_AUTH_METHOD": "bearer",
	}, "repo", "list", "PROJ")
	if err != nil {
		t.Fatalf("dc repo list: %v", err)
	}
	if !strings.Contains(out, "repo1") {
		t.Fatalf("output = %q", out)
	}
}

func TestBrowsePrintsURL(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		notFound(w)
	})
	out, _, err := runCommand(t, server, nil, "browse", "-R", "acme/widgets", "--pr", "7", "--no-open")
	if err != nil {
		t.Fatalf("browse: %v", err)
	}
	if strings.TrimSpace(out) != "https://bitbucket.org/acme/widgets/pull-requests/7" {
		t.Fatalf("output = %q", out)
	}
}

func TestAuthLoginStoresCredentialAndStatusVerifies(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/user" {
			t.Errorf("path = %q", r.URL.Path)
			notFound(w)
			return
		}
		writeJSON(w, `{"display_name":"Me","uuid":"{u-1}"}`)
	})
	dir := t.TempDir()
	env := envMap{
		"BK_CONFIG_DIR":       dir,
		"BK_FORCE_FILE_STORE": "1",
	}

	out, errOut, err := runCommandWithInput(t, server, env, "token\n", "auth", "login", "--hostname", "bitbucket.org", "--username", "me@example.com", "--with-token")
	if err != nil {
		t.Fatalf("auth login: %v\n%s", err, errOut)
	}
	if !strings.Contains(out, "Logged in to bitbucket.org") {
		t.Fatalf("login output = %q", out)
	}

	credentialPath := dir + "/credentials.json"
	info, err := os.Stat(credentialPath)
	if err != nil {
		t.Fatalf("credential file: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("credential permissions = %v", info.Mode().Perm())
	}

	statusOut, _, err := runCommandWithInput(t, server, env, "", "auth", "status")
	if err != nil {
		t.Fatalf("auth status: %v", err)
	}
	if !strings.Contains(statusOut, "ok (Me)") {
		t.Fatalf("status output = %q", statusOut)
	}
}

func TestConfigSetGet(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		notFound(w)
	})
	dir := t.TempDir()
	env := envMap{"BK_CONFIG_DIR": dir}

	if _, _, err := runCommand(t, server, env, "config", "set", "active_host", "bitbucket.org"); err != nil {
		t.Fatalf("set active_host: %v", err)
	}
	if _, _, err := runCommand(t, server, env, "config", "set", "workspace", "acme"); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	out, _, err := runCommand(t, server, env, "config", "get", "workspace")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if strings.TrimSpace(out) != "acme" {
		t.Fatalf("output = %q", out)
	}
}

// Bitbucket Cloud answers a POST for a source/destination pair that already has
// an open pull request by updating it in place. pr create must refuse instead of
// overwriting someone's title and description.
func TestPRCreateRefusesWhenPullRequestIsOpen(t *testing.T) {
	var query string
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets":
			writeJSON(w, `{"slug":"widgets","full_name":"acme/widgets","mainbranch":{"name":"main"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests":
			query = r.URL.Query().Get("q")
			writeJSON(w, `{"values":[{"id":7,"title":"Existing","state":"OPEN","summary":{"raw":"x"},"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/7"}}}]}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	_, _, err := runCommand(t, server, nil, "pr", "create", "-R", "acme/widgets", "--head", "feature", "--title", "teste", "--body", "teste")
	if err == nil || !strings.Contains(err.Error(), "#7 is already open") || !strings.Contains(err.Error(), "--update") {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(query, `source.branch.name="feature"`) || !strings.Contains(query, `destination.branch.name="main"`) {
		t.Fatalf("q = %q", query)
	}
}

func TestPRCreateUpdateEditsTheOpenPullRequest(t *testing.T) {
	var updated map[string]any
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets":
			writeJSON(w, `{"slug":"widgets","full_name":"acme/widgets","mainbranch":{"name":"main"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests":
			writeJSON(w, `{"values":[{"id":7,"title":"Old","state":"OPEN"}]}`)
		case r.Method == http.MethodPut && r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/7":
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &updated); err != nil {
				t.Errorf("decode body: %v", err)
			}
			writeJSON(w, `{"id":7,"title":"New","state":"OPEN","links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/7"}}}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pr", "create", "-R", "acme/widgets", "--head", "feature", "--title", "New", "--body", "Body", "--update")
	if err != nil {
		t.Fatalf("pr create --update: %v", err)
	}
	if !strings.Contains(out, "Updated pull request #7") {
		t.Fatalf("output = %q", out)
	}
	if updated["title"] != "New" || updated["description"] != "Body" {
		t.Fatalf("body = %+v", updated)
	}
	if _, ok := updated["reviewers"]; ok {
		t.Fatalf("reviewers must be left untouched: %+v", updated)
	}
}

func TestPipelineListShowsPullRequestSourceBranch(t *testing.T) {
	server := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/2.0/repositories/acme/widgets/pipelines":
			writeJSON(w, `{"values":[{"build_number":965,"state":{"name":"COMPLETED","result":{"name":"SUCCESSFUL"}},"target":{"type":"pipeline_pullrequest_target","ref_name":null,"source":"feature/x","destination":"main"},"trigger":{"name":"PUSH"}}]}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			notFound(w)
		}
	})

	out, _, err := runCommand(t, server, nil, "pipeline", "list", "-R", "acme/widgets")
	if err != nil {
		t.Fatalf("pipeline list: %v", err)
	}
	if !strings.Contains(out, "feature/x") {
		t.Fatalf("output = %q", out)
	}
}
