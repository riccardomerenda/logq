package ui

import "testing"

func TestQueryBarHistory(t *testing.T) {
	qb := NewQueryBar()

	// Push some queries
	qb.PushHistory("level:error")
	qb.PushHistory("service:auth")
	qb.PushHistory("latency>500")

	// Browse up through history
	qb.Focus()
	qb.SetValue("current draft")

	if !qb.HistoryUp() {
		t.Fatal("HistoryUp should return true")
	}
	if qb.Value() != "latency>500" {
		t.Errorf("After first Up, got %q, want %q", qb.Value(), "latency>500")
	}

	if !qb.HistoryUp() {
		t.Fatal("HistoryUp should return true")
	}
	if qb.Value() != "service:auth" {
		t.Errorf("After second Up, got %q, want %q", qb.Value(), "service:auth")
	}

	if !qb.HistoryUp() {
		t.Fatal("HistoryUp should return true")
	}
	if qb.Value() != "level:error" {
		t.Errorf("After third Up, got %q, want %q", qb.Value(), "level:error")
	}

	// At top, stays on oldest
	qb.HistoryUp()
	if qb.Value() != "level:error" {
		t.Errorf("At top, got %q, want %q", qb.Value(), "level:error")
	}

	// Navigate back down
	qb.HistoryDown()
	if qb.Value() != "service:auth" {
		t.Errorf("After Down, got %q, want %q", qb.Value(), "service:auth")
	}

	qb.HistoryDown()
	if qb.Value() != "latency>500" {
		t.Errorf("After second Down, got %q, want %q", qb.Value(), "latency>500")
	}

	// Past end restores draft
	qb.HistoryDown()
	if qb.Value() != "current draft" {
		t.Errorf("Past end, got %q, want %q", qb.Value(), "current draft")
	}
}

func TestQueryBarHistoryEmpty(t *testing.T) {
	qb := NewQueryBar()
	qb.Focus()

	if qb.HistoryUp() {
		t.Error("HistoryUp on empty history should return false")
	}
	if qb.HistoryDown() {
		t.Error("HistoryDown without entering history should return false")
	}
}

func TestQueryBarHistoryDedup(t *testing.T) {
	qb := NewQueryBar()

	qb.PushHistory("level:error")
	qb.PushHistory("level:error") // duplicate, should be ignored
	qb.PushHistory("service:auth")

	if len(qb.history) != 2 {
		t.Errorf("History length = %d, want 2 (dedup consecutive)", len(qb.history))
	}
}

func TestQueryBarHistoryIgnoresEmpty(t *testing.T) {
	qb := NewQueryBar()

	qb.PushHistory("")
	if len(qb.history) != 0 {
		t.Error("Empty string should not be added to history")
	}
}

func TestQueryBarHistoryCap(t *testing.T) {
	qb := NewQueryBar()

	for i := 0; i < 150; i++ {
		qb.PushHistory(string(rune('a' + i%26)))
	}

	if len(qb.history) > 100 {
		t.Errorf("History length = %d, should be capped at 100", len(qb.history))
	}
}

func TestQueryBarBlurResetsHistory(t *testing.T) {
	qb := NewQueryBar()
	qb.PushHistory("level:error")
	qb.Focus()
	qb.HistoryUp()

	qb.Blur()
	if qb.historyIdx != -1 {
		t.Errorf("Blur should reset historyIdx, got %d", qb.historyIdx)
	}
}
