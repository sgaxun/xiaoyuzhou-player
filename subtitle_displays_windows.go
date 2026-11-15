//go:build windows

package main

import (
	"fmt"
	"math"
	"strings"
	"unsafe"

	"github.com/egoist/mygo"
	"golang.org/x/sys/windows"
)

// DISPLAY_DEVICEW is fixed-size UTF-16, also on 64-bit Windows.
type subtitleDisplayDevice struct {
	Size        uint32
	Name        [32]uint16
	Description [128]uint16
	Flags       uint32
	ID          [128]uint16
	Key         [128]uint16
}

var enumSubtitleDisplayDevices = windows.NewLazySystemDLL("user32.dll").NewProc("EnumDisplayDevicesW")

func windowsSubtitleMonitors() ([]subtitleMonitor, error) {
	displays := mygo.Screen.Displays()
	monitors := make([]subtitleMonitor, 0, len(displays))
	for _, display := range displays {
		adapter, err := windows.UTF16PtrFromString(display.Label)
		if err != nil {
			return nil, err
		}
		device := subtitleDisplayDevice{}
		device.Size = uint32(unsafe.Sizeof(device))
		// Request the monitor interface path, not an HMONITOR or DISPLAY1
		// ordinal, so arrangement changes/restarts preserve the selection.
		ok, _, _ := enumSubtitleDisplayDevices.Call(uintptr(unsafe.Pointer(adapter)), 0, uintptr(unsafe.Pointer(&device)), 1)
		id, name := "", display.Label
		if ok != 0 {
			id = windows.UTF16ToString(device.ID[:])
			if description := windows.UTF16ToString(device.Description[:]); description != "" {
				name = description + " · " + display.Label
			}
		}
		// Virtual/remote displays may not expose a monitor interface.
		if id == "" {
			id = display.Label
		}
		monitors = append(monitors, subtitleMonitor{
			SubtitleDisplay: SubtitleDisplay{
				ID: "windows:" + strings.ToUpper(id), Name: name,
				Primary: display.Primary,
				Width:   int(math.Round(float64(display.Bounds.Width) * display.ScaleFactor)),
				Height:  int(math.Round(float64(display.Bounds.Height) * display.ScaleFactor)),
			},
			WorkArea: display.WorkArea,
		})
	}
	if len(monitors) == 0 {
		return nil, fmt.Errorf("未找到可用显示器")
	}
	return monitors, nil
}

func platformSubtitleDisplays() ([]SubtitleDisplay, error) {
	monitors, err := windowsSubtitleMonitors()
	if err != nil {
		return nil, err
	}
	result := make([]SubtitleDisplay, 0, len(monitors))
	for _, monitor := range monitors {
		result = append(result, monitor.SubtitleDisplay)
	}
	return result, nil
}
