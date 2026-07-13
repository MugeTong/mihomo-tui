package app

import "github.com/charmbracelet/x/ansi"

func truncateCells(text string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(text, width, "…")
}
