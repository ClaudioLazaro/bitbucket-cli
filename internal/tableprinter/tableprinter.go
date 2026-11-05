// Package tableprinter renders aligned tables on terminals and tab-separated
// rows when output is not a terminal.
package tableprinter

import (
	"fmt"
	"io"
	"strings"

	"github.com/ClaudioLazaro/bitbucket-cli/internal/iostreams"
	"github.com/ClaudioLazaro/bitbucket-cli/internal/text"
)

// Table collects rows and renders them on demand.
type Table struct {
	streams *iostreams.IOStreams
	header  []string
	rows    [][]string
}

// New returns an empty table bound to the given streams.
func New(streams *iostreams.IOStreams) *Table {
	return &Table{streams: streams}
}

// Header sets the header row.
func (t *Table) Header(cols ...string) *Table {
	t.header = cols
	return t
}

// AddRow appends a row.
func (t *Table) AddRow(cols ...string) *Table {
	t.rows = append(t.rows, cols)
	return t
}

// Render writes the table to the output stream.
func (t *Table) Render() error {
	if t.streams.IsStdoutTTY() {
		return t.renderTTY(t.streams.Out)
	}
	return t.renderPlain(t.streams.Out)
}

func (t *Table) renderPlain(w io.Writer) error {
	for _, row := range t.rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func (t *Table) renderTTY(w io.Writer) error {
	columns := len(t.header)
	for _, row := range t.rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return nil
	}

	widths := make([]int, columns)
	for i, cell := range t.header {
		widths[i] = text.DisplayWidth(cell)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= columns {
				break
			}
			if w := text.DisplayWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// Shrink the widest columns until the table fits the terminal.
	total := 0
	for _, width := range widths {
		total += width + 2
	}
	maxWidth := t.streams.TerminalWidth() - 2
	for total > maxWidth {
		widest := 0
		for i, width := range widths {
			if width > widths[widest] {
				widest = i
			}
		}
		if widths[widest] <= 4 {
			break
		}
		widths[widest]--
		total--
	}

	if len(t.header) > 0 {
		if err := writeRow(w, t.header, widths, true); err != nil {
			return err
		}
	}
	for _, row := range t.rows {
		if err := writeRow(w, row, widths, false); err != nil {
			return err
		}
	}
	return nil
}

func writeRow(w io.Writer, row []string, widths []int, header bool) error {
	var b strings.Builder
	for i, width := range widths {
		cell := ""
		if i < len(row) {
			cell = row[i]
		}
		cell = text.Truncate(cell, width)
		if i == len(widths)-1 {
			b.WriteString(cell)
		} else {
			b.WriteString(text.PadRight(cell, width))
			b.WriteString("  ")
		}
	}
	if header {
		line := strings.TrimRight(b.String(), " ")
		b.Reset()
		b.WriteString(strings.ToUpper(line))
	}
	_, err := fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	return err
}
