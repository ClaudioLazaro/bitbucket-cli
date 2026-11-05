package bitbucket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// HTTPError is a non-2xx response from Bitbucket.
type HTTPError struct {
	StatusCode int
	Method     string
	URL        string
	Message    string
	Detail     string
	RequestID  string
}

// Error implements error.
func (e *HTTPError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	out := fmt.Sprintf("HTTP %d: %s", e.StatusCode, msg)
	if e.Detail != "" && e.Detail != msg {
		out += ": " + e.Detail
	}
	if id := e.RequestID; id != "" {
		out += fmt.Sprintf(" (request id %s)", id)
	}
	return out
}

// Hint returns actionable guidance for common failures.
func (e *HTTPError) Hint() string {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return "credentials are missing, invalid, or expired. Bitbucket Cloud app passwords were removed in July 2026; create an API token (email + token) or an access token and run `bk auth login` again"
	case http.StatusForbidden:
		return "the credential lacks the scopes required for this operation; check the token's Bitbucket permissions"
	case http.StatusNotFound:
		return "resource not found; check the workspace/project, repository slug, and identifier"
	case http.StatusConflict:
		return "the resource changed concurrently or already exists; refresh and try again"
	}
	return ""
}

// IsNotFound reports whether err is a 404 response.
func IsNotFound(err error) bool { return statusIs(err, http.StatusNotFound) }

// IsUnauthorized reports whether err is a 401 response.
func IsUnauthorized(err error) bool { return statusIs(err, http.StatusUnauthorized) }

// IsForbidden reports whether err is a 403 response.
func IsForbidden(err error) bool { return statusIs(err, http.StatusForbidden) }

func statusIs(err error, status int) bool {
	var httpErr *HTTPError
	return errors.As(err, &httpErr) && httpErr.StatusCode == status
}

// NewHTTPError extracts Bitbucket's error envelope from a response body.
func NewHTTPError(resp *http.Response, method, url string, body []byte) *HTTPError {
	e := &HTTPError{
		StatusCode: resp.StatusCode,
		Method:     method,
		URL:        url,
		RequestID:  firstHeader(resp, "x-request-id", "x-b3-traceid"),
	}
	var cloud struct {
		Error struct {
			Message string         `json:"message"`
			Detail  string         `json:"detail"`
			Fields  map[string]any `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &cloud); err == nil && cloud.Error.Message != "" {
		e.Message = cloud.Error.Message
		e.Detail = cloud.Error.Detail
		if e.Detail == "" && len(cloud.Error.Fields) > 0 {
			parts := make([]string, 0, len(cloud.Error.Fields))
			for key, value := range cloud.Error.Fields {
				parts = append(parts, fmt.Sprintf("%s: %v", key, value))
			}
			e.Detail = strings.Join(parts, "; ")
		}
		return e
	}
	var dc struct {
		Errors []struct {
			Message       string `json:"message"`
			ExceptionName string `json:"exceptionName"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &dc); err == nil && len(dc.Errors) > 0 {
		e.Message = dc.Errors[0].Message
		if e.Message == "" {
			e.Message = dc.Errors[0].ExceptionName
		}
		if len(dc.Errors) > 1 {
			extra := make([]string, 0, len(dc.Errors)-1)
			for _, item := range dc.Errors[1:] {
				extra = append(extra, item.Message)
			}
			e.Detail = strings.Join(extra, "; ")
		}
		return e
	}
	// Fall back to a short, single-line excerpt of the body.
	text := strings.TrimSpace(string(body))
	if len(text) > 200 {
		text = text[:200] + "..."
	}
	e.Detail = strings.ReplaceAll(text, "\n", " ")
	return e
}

func firstHeader(resp *http.Response, names ...string) string {
	for _, name := range names {
		if value := resp.Header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

// ErrUnsupported marks an operation that is not available on this host kind.
var ErrUnsupported = errors.New("operation not supported on this Bitbucket host")

// UnsupportedError describes an operation missing for a host kind.
type UnsupportedError struct {
	Operation string
	Kind      string
}

// Error implements error.
func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("%s is not supported on %s hosts yet", e.Operation, e.Kind)
}

// Is lets errors.Is match ErrUnsupported.
func (e *UnsupportedError) Is(target error) bool { return target == ErrUnsupported }

func unsupported(operation string) error {
	return &UnsupportedError{Operation: operation, Kind: "Data Center"}
}
