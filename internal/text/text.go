// Package text contains small helpers for displaying values in terminals.
package text

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// DisplayWidth returns the number of runes in s. It is a deliberately simple
// width calculation: wide runes count as one column, which is good enough for
// the tables this CLI renders.
func DisplayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// Truncate shortens s to at most width runes, appending "..." when truncated.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(s) <= width {
		return s
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	runes := []rune(s)
	return string(runes[:width-3]) + "..."
}

// PadRight pads s with spaces up to width runes.
func PadRight(s string, width int) string {
	if n := width - DisplayWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// RelativeTime formats t compactly relative to now.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// Date formats t as a compact UTC date and time.
func Date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02 15:04")
}

// Plural returns singular when n is 1, plural otherwise.
func Plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

// Indent prefixes every line of s with prefix.
func Indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}
