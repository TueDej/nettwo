package views

import (
	"strings"
	"testing"
)

// The dashboard fills the terminal exactly: a view taller or shorter than the
// window causes the flicker/overflow the user reported, and it gets worse when
// the status bar wraps its key hints onto extra lines.
func TestDashboardViewFitsWindowExactly(t *testing.T) {
	sizes := []struct{ w, h int }{
		{120, 40},
		{100, 30},
		{90, 25},
		{80, 24},
		{70, 22},
		{60, 20},
		{50, 20},
		{40, 19},
	}

	for _, s := range sizes {
		m := NewDashboardModel()
		m.UpdateAccounts([]string{"user@sharif.edu", "second@sharif.edu"},
			map[string]string{"user@sharif.edu": "12.500 GB"})
		m.AddActivity("Logged in", true)
		m.SetSize(s.w, s.h)

		got := strings.Count(m.View(), "\n") + 1
		if got != s.h {
			t.Errorf("size %dx%d: view is %d lines tall", s.w, s.h, got)
		}
	}
}

// A narrower terminal needs more height, because the key hints wrap.
func TestDashboardMinHeightGrowsWhenNarrower(t *testing.T) {
	m := NewDashboardModel()
	if wide, narrow := m.MinHeight(120), m.MinHeight(40); narrow <= wide {
		t.Errorf("MinHeight(40) = %d should exceed MinHeight(120) = %d", narrow, wide)
	}
}

// Panel height must account for the wrapped status bar, so the panels shrink
// instead of pushing the bar off the screen.
func TestDashboardPanelHeightAccountForStatusBar(t *testing.T) {
	m := NewDashboardModel()
	m.SetSize(120, 40)
	wide := m.panelHeight()

	m.SetSize(45, 40) // narrow: key hints wrap, so the bar is taller
	narrow := m.panelHeight()

	if narrow >= wide {
		t.Errorf("narrow terminal panel height %d should be smaller than wide %d", narrow, wide)
	}
}
