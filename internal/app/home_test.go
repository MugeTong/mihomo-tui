package app

import "testing"

func TestNodeWindowKeepsCursorVisible(t *testing.T) {
	tests := []struct {
		name        string
		bodyHeight  int
		total       int
		cursor      int
		previous    int
		wantVisible bool
	}{
		{name: "top", bodyHeight: 8, total: 12, cursor: 0, previous: 0, wantVisible: true},
		{name: "middle", bodyHeight: 8, total: 12, cursor: 6, previous: 1, wantVisible: true},
		{name: "end", bodyHeight: 8, total: 12, cursor: 11, previous: 6, wantVisible: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			window := nodeWindow(tt.bodyHeight, tt.total, tt.cursor, tt.previous)
			gotVisible := tt.cursor >= window.start && tt.cursor < window.end
			if gotVisible != tt.wantVisible {
				t.Fatalf("cursor visibility = %v, want %v; window = %+v", gotVisible, tt.wantVisible, window)
			}
		})
	}
}

func TestNodeWindowFitsAvailableRows(t *testing.T) {
	window := nodeWindow(8, 12, 6, 1)
	renderedRows := window.end - window.start + 1 // position line
	if window.hasAbove {
		renderedRows++
	}
	if window.hasBelow {
		renderedRows++
	}

	if renderedRows > 8 {
		t.Fatalf("rendered rows = %d, want <= 8; window = %+v", renderedRows, window)
	}
}
