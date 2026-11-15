package main

import "github.com/egoist/mygo"

type subtitleMonitor struct {
	SubtitleDisplay
	WorkArea mygo.Rectangle
}

// Falling back never changes the saved ID, allowing a reconnected display to
// take over again, including while playback is paused.
func selectedSubtitleWorkArea(monitors []subtitleMonitor, id string) (mygo.Rectangle, bool) {
	for _, monitor := range monitors {
		if id != "" && id != "auto" && monitor.ID == id {
			return monitor.WorkArea, true
		}
	}
	for _, monitor := range monitors {
		if monitor.Primary {
			return monitor.WorkArea, true
		}
	}
	if len(monitors) > 0 {
		return monitors[0].WorkArea, true
	}
	return mygo.Rectangle{}, false
}
