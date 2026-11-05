package cmdutil

import (
	"errors"
	"fmt"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/bitbucket"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
)

// ExitError carries a specific process exit code.
type ExitError struct {
	Code int
	Err  error
}

// Error implements error.
func (e *ExitError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit status %d", e.Code)
	}
	return e.Err.Error()
}

// Unwrap exposes the underlying error.
func (e *ExitError) Unwrap() error { return e.Err }

// PrintError writes err and any actionable hint to standard error.
func PrintError(io *iostreams.IOStreams, err error) {
	fmt.Fprintf(io.ErrOut, "Error: %v\n", err)
	var httpErr *bitbucket.HTTPError
	if errors.As(err, &httpErr) {
		if hint := httpErr.Hint(); hint != "" {
			fmt.Fprintf(io.ErrOut, "Hint: %s\n", hint)
		}
	}
}
