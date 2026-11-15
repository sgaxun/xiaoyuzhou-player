package main

import "github.com/egoist/mygo"

// Screen work areas are DIPs and may have negative origins. Keep the strip
// wholly inside even unusually small work areas, with a top inset if it fits.
func subtitleStripBounds(area mygo.Rectangle) mygo.Rectangle {
	width, height := min(subtitleWidth, area.Width), min(subtitleHeight, area.Height)
	return mygo.Rectangle{X: area.X + (area.Width-width)/2, Y: area.Y + min(12, max(0, area.Height-height)), Width: width, Height: height}
}
