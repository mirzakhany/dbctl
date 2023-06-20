package table

import (
	"fmt"
	"io"
	"strings"
)

type Table struct {
	w      io.Writer
	rows   [][]string
	widths []int
}

func New(w io.Writer) *Table {
	return &Table{w: w}
}

func (t *Table) AddRow(row ...string) {
	if t.widths == nil {
		t.widths = make([]int, len(row))
	} else if len(t.widths) != len(row) {
		panic(fmt.Errorf("bad size: got %d, want %d", len(row), len(t.widths)))
	}
	for i, s := range row {
		if len(s) > t.widths[i] {
			t.widths[i] = len(s)
		}
	}

	t.rows = append(t.rows, row)
}

func (t *Table) Print() {
	writeRow := func(l, m, r string, col func(i, width int) string) {
		_, _ = fmt.Fprint(t.w, l)
		for i, width := range t.widths {
			_, _ = fmt.Fprint(t.w, col(i, width))
			if i != len(t.widths)-1 {
				_, _ = fmt.Fprint(t.w, m)
			}
		}
		_, _ = fmt.Fprintln(t.w, r)
	}

	width := 1
	for _, w := range t.widths {
		width += w + 3
	}

	writeRow("╭", "┬", "╮", func(i, w int) string {
		return strings.Repeat("─", w+2)
	})

	writeRow("│", "│", "│", func(i, w int) string {
		text := t.rows[0][i]
		s := text
		return fmt.Sprintf(" %-*s ", w+len(s)-len(text), s)
	})

	writeRow("├", "┼", "┤", func(i, w int) string {
		return strings.Repeat("─", w+2)
	})

	for rid, row := range t.rows {
		if rid == 0 {
			continue
		}
		writeRow("│", "│", "│", func(i, w int) string {
			s := row[i]
			return fmt.Sprintf(" %-*s ", w+len(s)-len(s), s)
		})
	}

	writeRow("╰", "┴", "╯", func(i, w int) string {
		return strings.Repeat("─", w+2)
	})
}
