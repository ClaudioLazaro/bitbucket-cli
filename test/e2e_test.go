package test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary build in -short mode")
	}

	bin := filepath.Join(t.TempDir(), "bk")
	build := exec.Command("go", "build", "-o", bin, "../cmd/bk")
	build.Stderr = os.Stderr
	if output, err := build.Output(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, output)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2.0/repositories/acme":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"values":[{"slug":"widgets","full_name":"acme/widgets","is_private":true}]}`)
		case "/2.0/repositories/acme/widgets/pullrequests/9":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":9,"state":"OPEN","source":{"branch":{"name":"feature"}},"destination":{"branch":{"name":"main"}}}`)
		case "/2.0/repositories/acme/widgets/pullrequests/9/statuses":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"values":[{"key":"CI","name":"build","state":"FAILED"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"message":"not found"}}`)
		}
	}))
	defer server.Close()

	run := func(args ...string) (string, string, int) {
		cmd := exec.Command(bin, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Env = append(os.Environ(),
			"BK_CONFIG_DIR="+t.TempDir(),
			"BK_HOST=bitbucket.org",
			"BK_TOKEN=token",
			"BK_USERNAME=me@example.com",
			"BK_AUTH_METHOD=basic",
			"BK_API_URL="+server.URL+"/2.0",
		)
		err := cmd.Run()
		code := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				code = exitErr.ExitCode()
			} else {
				t.Fatalf("run %v: %v", args, err)
			}
		}
		return stdout.String(), stderr.String(), code
	}

	stdout, stderr, code := run("version")
	if code != 0 || !strings.Contains(stdout, "bk version") {
		t.Fatalf("version: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	stdout, stderr, code = run("repo", "list", "acme")
	if code != 0 || !strings.Contains(stdout, "widgets") {
		t.Fatalf("repo list: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}

	stdout, stderr, code = run("pr", "checks", "9", "-R", "acme/widgets")
	if code != 1 {
		t.Fatalf("pr checks: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "FAILED") {
		t.Fatalf("pr checks output = %q", stdout)
	}

	stdout, stderr, code = run("--help")
	if code != 0 || !strings.Contains(stdout, "Available Commands") {
		t.Fatalf("help: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
