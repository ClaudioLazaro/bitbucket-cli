// Package iostreams provides the standard input, output, and error streams
// used by every command, together with terminal detection and color support.
package iostreams

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// IOStreams groups the streams a command writes to and reads from.
type IOStreams struct {
	In     io.Reader
	Out    io.Writer
	ErrOut io.Writer

	stdinTTY  bool
	stdoutTTY bool
	stderrTTY bool
	color     bool
	width     int
}

// System returns streams bound to the process standard file descriptors.
func System() *IOStreams {
	stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
	stdoutTTY := term.IsTerminal(int(os.Stdout.Fd()))
	stderrTTY := term.IsTerminal(int(os.Stderr.Fd()))
	s := &IOStreams{
		In:        os.Stdin,
		Out:       os.Stdout,
		ErrOut:    os.Stderr,
		stdinTTY:  stdinTTY,
		stdoutTTY: stdoutTTY,
		stderrTTY: stderrTTY,
		width:     80,
	}
	if stdoutTTY {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			s.width = w
		}
	}
	s.color = stdoutTTY && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return s
}

// Test returns streams backed by buffers, with TTY detection disabled.
func Test() (io *IOStreams, out, errOut, in *bytes.Buffer) {
	in = &bytes.Buffer{}
	out = &bytes.Buffer{}
	errOut = &bytes.Buffer{}
	return &IOStreams{In: in, Out: out, ErrOut: errOut, width: 80}, out, errOut, in
}

// IsStdinTTY reports whether standard input is attached to a terminal.
func (s *IOStreams) IsStdinTTY() bool { return s.stdinTTY }

// IsStdoutTTY reports whether standard output is attached to a terminal.
func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutTTY }

// IsStderrTTY reports whether standard error is attached to a terminal.
func (s *IOStreams) IsStderrTTY() bool { return s.stderrTTY }

// ColorEnabled reports whether ANSI color should be emitted.
func (s *IOStreams) ColorEnabled() bool { return s.color }

// SetColorEnabled overrides color detection. It is mostly useful in tests.
func (s *IOStreams) SetColorEnabled(enabled bool) { s.color = enabled }

// SetTTY overrides terminal detection. It is mostly useful in tests.
func (s *IOStreams) SetTTY(stdin, stdout, stderr bool) {
	s.stdinTTY = stdin
	s.stdoutTTY = stdout
	s.stderrTTY = stderr
}

// TerminalWidth returns the detected terminal width, defaulting to 80.
func (s *IOStreams) TerminalWidth() int {
	if s.width <= 0 {
		return 80
	}
	return s.width
}

// SetTerminalWidth overrides the detected terminal width.
func (s *IOStreams) SetTerminalWidth(width int) { s.width = width }

// ReadLine writes prompt to standard error and reads one line from standard input.
func (s *IOStreams) ReadLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(s.ErrOut, prompt)
	}
	r := bufio.NewReader(io.LimitReader(s.In, 1<<20))
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// ReadPassword writes prompt to standard error and reads a secret without echo
// when standard input is a terminal.
func (s *IOStreams) ReadPassword(prompt string) (string, error) {
	if prompt != "" {
		fmt.Fprint(s.ErrOut, prompt)
	}
	if f, ok := s.In.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(s.ErrOut)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	line, err := bufio.NewReader(io.LimitReader(s.In, 1<<20)).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

const (
	ansiReset   = "\x1b[0m"
	ansiBold    = "\x1b[1m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
)

// ColorScheme renders strings with ANSI attributes when color is enabled.
type ColorScheme struct {
	enabled bool
}

// ColorScheme returns the color scheme associated with these streams.
func (s *IOStreams) ColorScheme() *ColorScheme {
	return &ColorScheme{enabled: s.color}
}

func (c *ColorScheme) wrap(code, s string) string {
	if !c.enabled || s == "" {
		return s
	}
	return code + s + ansiReset
}

// Bold renders s in bold.
func (c *ColorScheme) Bold(s string) string { return c.wrap(ansiBold, s) }

// Dim renders s dimmed.
func (c *ColorScheme) Dim(s string) string { return c.wrap(ansiDim, s) }

// Red renders s in red.
func (c *ColorScheme) Red(s string) string { return c.wrap(ansiRed, s) }

// Green renders s in green.
func (c *ColorScheme) Green(s string) string { return c.wrap(ansiGreen, s) }

// Yellow renders s in yellow.
func (c *ColorScheme) Yellow(s string) string { return c.wrap(ansiYellow, s) }

// Magenta renders s in magenta.
func (c *ColorScheme) Magenta(s string) string { return c.wrap(ansiMagenta, s) }

// Cyan renders s in cyan.
func (c *ColorScheme) Cyan(s string) string { return c.wrap(ansiCyan, s) }

// Greenf renders a formatted string in green.
func (c *ColorScheme) Greenf(format string, a ...any) string {
	return c.Green(fmt.Sprintf(format, a...))
}

// Redf renders a formatted string in red.
func (c *ColorScheme) Redf(format string, a ...any) string {
	return c.Red(fmt.Sprintf(format, a...))
}

// Yellowf renders a formatted string in yellow.
func (c *ColorScheme) Yellowf(format string, a ...any) string {
	return c.Yellow(fmt.Sprintf(format, a...))
}
