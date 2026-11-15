package main

import "fmt"

// The player service serializes partial preference patches with all other
// state writes. Never read/modify/write state.json outside that lock.
func (b *Backend) SubtitlePreference() (map[string]any, error) {
	result, err := b.service.Request("state", nil)
	if err != nil {
		return nil, err
	}
	response, ok := result.(map[string]any)
	if !ok || response["ok"] != true {
		return nil, fmt.Errorf("读取字幕设置失败：%v", result)
	}
	state, ok := response["state"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("字幕设置格式错误")
	}
	// Null distinguishes a legacy installation from an explicitly disabled
	// preference, so the UI can migrate its existing localStorage switch once.
	var enabled any
	if value, ok := state["menubarTranscript"].(bool); ok {
		enabled = value
	}
	mode, _ := state["subtitleMode"].(string)
	if mode != "menubar" {
		mode = "notch"
	}
	display, _ := state["subtitleDisplayId"].(string)
	if display == "" {
		display = "auto"
	}
	name, _ := state["subtitleDisplayName"].(string)
	return map[string]any{"enabled": enabled, "mode": mode, "displayId": display, "displayName": name}, nil
}

func (b *Backend) SaveSubtitleMode(mode string) error {
	if mode != "notch" && mode != "menubar" {
		return fmt.Errorf("不支持的字幕位置")
	}
	b.mediaMu.Lock()
	defer b.mediaMu.Unlock()
	result, err := b.service.Request("savePrefs", map[string]any{"subtitleMode": mode})
	if err != nil {
		return err
	}
	if response, ok := result.(map[string]any); !ok || response["ok"] != true {
		return fmt.Errorf("保存字幕位置失败")
	}
	if err := b.platformSubtitle("", 0); err != nil {
		return err
	}
	if b.tray != nil {
		if err := b.subtitleRenderer.update("", 0, &b.subtitle, &b.subtitleOffset, func(raw []byte) error {
			b.tray.SetTitle("")
			return b.tray.SetIcon(raw, true)
		}); err != nil {
			return err
		}
	}
	b.subtitleMode = mode
	return b.renderSubtitle(b.subtitleText, b.subtitleProgress)
}

func (b *Backend) SaveSubtitlePreference(enabled bool) error {
	result, err := b.service.Request("savePrefs", map[string]any{"menubarTranscript": enabled})
	if err != nil {
		return err
	}
	response, ok := result.(map[string]any)
	if !ok || response["ok"] != true {
		return fmt.Errorf("保存字幕设置失败：%v", result)
	}
	return nil
}

// SubtitleDisplay uses a stable platform identity, never an array position.
type SubtitleDisplay struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BuiltIn bool   `json:"builtIn"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

func (b *Backend) SubtitleDisplays() ([]SubtitleDisplay, error) {
	return platformSubtitleDisplays()
}

func (b *Backend) SaveSubtitleDisplay(id string) error {
	b.mediaMu.Lock()
	defer b.mediaMu.Unlock()
	name := ""
	if id != "auto" {
		displays, err := platformSubtitleDisplays()
		if err != nil {
			return err
		}
		found := false
		for _, display := range displays {
			if display.ID == id {
				name = display.Name
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("所选显示屏已断开，请刷新后重试")
		}
	}
	result, err := b.service.Request("savePrefs", map[string]any{"subtitleDisplayId": id, "subtitleDisplayName": name})
	if err != nil {
		return err
	}
	if response, ok := result.(map[string]any); !ok || response["ok"] != true {
		return fmt.Errorf("保存字幕显示屏失败")
	}
	b.subtitleDisplay = id
	// Reposition even when paused; the scheduler deduplicates unchanged text.
	return b.renderSubtitle(b.subtitleText, b.subtitleProgress)
}
