package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"nettwo/internal/tui/styles"
)

// renderedHint is one key hint exactly as the status bar lays it out.
func renderedHint(kb KeyBinding) string {
	return styles.StatusBarKey.Render(kb.Key) +
		styles.StatusBarSep.Render(hintKeySep) +
		kb.Description
}

// singleLineWidth is the terminal width at which every hint just fits on one
// line: the hints themselves, the separators between them, and the bar's
// horizontal padding (4 cells).
func singleLineWidth(bar StatusBarModel) int {
	total := 4
	for i, kb := range bar.keys {
		if i > 0 {
			total += hintSeparatorWidth
		}
		total += lipgloss.Width(renderedHint(kb))
	}
	return total
}

func TestStatusBarHintsWrapInsteadOfTruncating(t *testing.T) {
	tests := []struct {
		name     string
		width    func(total int) int
		minLines int
	}{
		{
			name:     "roomy terminal keeps one line",
			width:    func(total int) int { return total },
			minLines: 1,
		},
		{
			name:     "narrow terminal wraps onto several lines",
			width:    func(total int) int { return total / 2 },
			minLines: 2,
		},
		{
			name:     "very narrow terminal still shows every hint",
			width:    func(total int) int { return total / 4 },
			minLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := NewStatusBarModel()
			bar.SetWidth(tt.width(singleLineWidth(bar)))

			lines := bar.hintLines()
			if len(lines) < tt.minLines {
				t.Fatalf("got %d line(s), want at least %d: %q", len(lines), tt.minLines, lines)
			}

			joined := stripANSI(strings.Join(lines, "\n"))
			for _, kb := range bar.keys {
				if !strings.Contains(joined, kb.Description) {
					t.Errorf("hint %q was dropped:\n%s", kb.Description, joined)
				}
			}

			// Callers reserve space using Height(); it must match View().
			if bar.Height() != 2+len(lines) {
				t.Errorf("Height() = %d, want %d", bar.Height(), 2+len(lines))
			}
		})
	}
}

func TestStatusBarViewLineCountMatchesHeight(t *testing.T) {
	for _, width := range []int{120, 60, 45, 40} {
		bar := NewStatusBarModel()
		bar.SetWidth(width)
		got := strings.Count(bar.View(), "\n") + 1
		if got != bar.Height() {
			t.Errorf("width %d: View() rendered %d lines, Height() reports %d", width, got, bar.Height())
		}
	}
}

// A hint wider than the whole bar is the only case that may be elided.
func TestStatusBarElidesOnlyWhenAHintCannotFit(t *testing.T) {
	bar := NewStatusBarModel()
	bar.SetWidth(4) // contentWidth 0: nothing fits

	lines := bar.hintLines()
	if len(lines) != len(bar.keys) {
		t.Fatalf("got %d line(s), want one elided hint per line (%d)", len(lines), len(bar.keys))
	}
	for _, line := range lines {
		if lipgloss.Width(line) != 0 {
			t.Errorf("line %q is not elided at width 4", line)
		}
	}
}

func TestStatusBarSurvivesDegenerateWidths(t *testing.T) {
	for _, width := range []int{0, 1, 5, 20} {
		bar := NewStatusBarModel()
		bar.SetWidth(width)
		if len(bar.hintLines()) == 0 {
			t.Fatalf("width %d: no lines", width)
		}
		if bar.View() == "" {
			t.Errorf("width %d: empty view", width)
		}
	}
}
