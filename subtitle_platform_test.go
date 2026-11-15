package main

import (
	"bytes"
	"image/png"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestSubtitleStripWorkArea(t *testing.T) {
	for _, area := range []mygo.Rectangle{
		{X: 0, Y: 0, Width: 1920, Height: 1040},
		{X: -1920, Y: -200, Width: 1920, Height: 1080},
		{X: 48, Y: 0, Width: 1232, Height: 720},
		{X: 10, Y: 20, Width: 180, Height: 24},
	} {
		bounds := subtitleStripBounds(area)
		if bounds.X < area.X || bounds.Y < area.Y || bounds.X+bounds.Width > area.X+area.Width || bounds.Y+bounds.Height > area.Y+area.Height {
			t.Fatalf("strip escapes work area: %+v in %+v", bounds, area)
		}
		if bounds.Width != min(subtitleWidth, area.Width) || bounds.Height != min(subtitleHeight, area.Height) {
			t.Fatal("unexpected strip size", bounds)
		}
		if area.Height >= subtitleHeight+12 && bounds.Y != area.Y+12 {
			t.Fatal("incorrect top inset", bounds)
		}
	}
}

// This test is compiled into the Windows test executable and also exercises
// the shared renderer on macOS. Native-window runtime acceptance needs Windows.
func TestSubtitlePlatformRaster(t *testing.T) {
	var renderer subtitleRenderer
	text := strings.Repeat("Windows 中文字幕滚动 ", 16)
	first, err := renderer.prepare(text, 0)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := renderer.render(first)
	if err != nil {
		t.Fatal(err)
	}
	last, err := renderer.prepare(text, 1)
	if err != nil || last <= first {
		t.Fatal("subtitle did not scroll", last, err)
	}
	end, err := renderer.render(last)
	if err != nil || bytes.Equal(raw, end) {
		t.Fatal("seek did not change raster", err)
	}
	img, err := png.Decode(bytes.NewReader(end))
	if err != nil || img.Bounds().Dx() != subtitleWidth || img.Bounds().Dy() != subtitleHeight {
		t.Fatal("invalid strip raster", err)
	}
	if runtime.GOOS == "windows" {
		paths := subtitleFontPaths()
		if len(paths) < 3 || filepath.Base(paths[0]) != "msyh.ttc" {
			t.Fatal("missing Windows Chinese font candidates", paths)
		}
		for _, path := range paths {
			if strings.Contains(path, "/System/Library") {
				t.Fatal("macOS font on Windows", path)
			}
		}
	}
}

func TestSubtitleMonitorSelectionAndFallback(t *testing.T) {
	primary := mygo.Rectangle{X: 0, Y: 40, Width: 1920, Height: 1040}
	external := mygo.Rectangle{X: -1600, Y: -300, Width: 1600, Height: 900}
	monitors := []subtitleMonitor{
		{SubtitleDisplay: SubtitleDisplay{ID: "external"}, WorkArea: external},
		{SubtitleDisplay: SubtitleDisplay{ID: "primary", Primary: true}, WorkArea: primary},
	}
	for _, id := range []string{"", "auto", "disconnected"} {
		got, ok := selectedSubtitleWorkArea(monitors, id)
		if !ok || got != primary {
			t.Fatalf("%s: %+v", id, got)
		}
	}
	got, ok := selectedSubtitleWorkArea(monitors, "external")
	if !ok || got != external {
		t.Fatal(got, ok)
	}
	// Enumerating in a different order must not alter an explicit selection.
	monitors[0], monitors[1] = monitors[1], monitors[0]
	got, ok = selectedSubtitleWorkArea(monitors, "external")
	if !ok || got != external {
		t.Fatal(got, ok)
	}
	got, ok = selectedSubtitleWorkArea(monitors[:1], "external")
	if !ok || got != primary {
		t.Fatal("disconnect fallback", got, ok)
	}
	if _, ok := selectedSubtitleWorkArea(nil, "auto"); ok {
		t.Fatal("accepted empty topology")
	}
}
