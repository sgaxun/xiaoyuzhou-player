//go:build darwin

package main

import (
	"os"
	"testing"
	"xiaoyuzhou-menubar/internal/player"
)

// Opt-in: opens the actual native panel, without loading an account or audio.
func TestNotchNativeLifecycle(t *testing.T) {
	if os.Getenv("XIAOYUZHOU_TEST_NOTCH") != "1" {
		t.Skip("requires a macOS desktop and the packaged native helper")
	}
	defer closePlatformSubtitle()
	b := &Backend{}
	for _, progress := range []float64{0, .5, .5, 1, .2} {
		if err := b.MenuSubtitle("刘海字幕检查：长句跟随播放进度滚动，暂停停住，向前跳转和回退立即同步。", progress); err != nil {
			t.Fatal(err)
		}
		if err := b.verifyPlatformSubtitle(); err != nil {
			t.Fatal(err)
		}
		if notchSubtitle.status.Progress != progress {
			t.Fatalf("seek/pause progress: got %v, want %v", notchSubtitle.status.Progress, progress)
		}
	}
	if err := b.MenuSubtitle("", 0); err != nil {
		t.Fatal(err)
	}
	if notchSubtitle.status.Visible {
		t.Fatal("clearing subtitles left the panel visible")
	}
	// A stopped helper can be recreated, including after application shutdown.
	closePlatformSubtitle()
	if err := b.MenuSubtitle("刘海字幕重新启动检查", .1); err != nil {
		t.Fatal(err)
	}
	if err := b.verifyPlatformSubtitle(); err != nil {
		t.Fatal(err)
	}
}

func TestNotchNativeDisplaySelection(t *testing.T) {
	if os.Getenv("XIAOYUZHOU_TEST_NOTCH") != "1" {
		t.Skip("requires native desktop helper")
	}
	defer closePlatformSubtitle()
	displays, err := platformSubtitleDisplays()
	if err != nil || len(displays) == 0 {
		t.Fatal(displays, err)
	}
	service, err := player.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	b := &Backend{service: service}
	if err := b.MenuSubtitle("显示屏选择测试：暂停时切换也应立即移动", .4); err != nil {
		t.Fatal(err)
	}
	for _, display := range displays {
		if err := b.SaveSubtitleDisplay(display.ID); err != nil {
			t.Fatal(err)
		}
		if notchSubtitle.status.DisplayID != display.ID {
			t.Fatalf("selected %s, active %s", display.ID, notchSubtitle.status.DisplayID)
		}
		prefs, err := b.SubtitlePreference()
		if err != nil || prefs["displayId"] != display.ID || prefs["displayName"] != display.Name {
			t.Fatal(prefs, err)
		}
		t.Logf("selected %s (%s)", display.Name, display.ID)
	}
	// Simulate the requested screen being absent without changing real monitors.
	b.subtitleDisplay = "disconnected-screen"
	if err := b.MenuSubtitle("显示屏断开回退检查", .4); err != nil {
		t.Fatal(err)
	}
	if notchSubtitle.status.DisplayID == "" || notchSubtitle.status.RequestedDisplayID != "disconnected-screen" {
		t.Fatal("fallback lost requested identity", notchSubtitle.status)
	}
	if err := b.SaveSubtitleDisplay("not-a-connected-display"); err == nil {
		t.Fatal("accepted disconnected display")
	}
	if err := b.SaveSubtitleDisplay("auto"); err != nil {
		t.Fatal(err)
	}
	if notchSubtitle.status.RequestedDisplayID != "auto" {
		t.Fatal("auto not applied")
	}
}
