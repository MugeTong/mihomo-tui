package app

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTruncateCellsPreservesEmojiGrapheme(t *testing.T) {
	const value = "🇯🇵 Tokyo"

	if got := truncateCells(value, 3); got != "🇯🇵…" {
		t.Fatalf("truncateCells() = %q, want %q", got, "🇯🇵…")
	}
	if got := lipgloss.Width(truncateCells(value, 3)); got != 3 {
		t.Fatalf("truncated width = %d, want 3", got)
	}
}

func TestTruncateCellsUsesTerminalCellWidth(t *testing.T) {
	const value = "日本节点"

	if got := truncateCells(value, 5); got != "日本…" {
		t.Fatalf("truncateCells() = %q, want %q", got, "日本…")
	}
}
