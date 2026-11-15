//go:build darwin

package main

import (
	"os"
	"testing"
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
