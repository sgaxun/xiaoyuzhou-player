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
	return map[string]any{"enabled": enabled, "mode": mode}, nil
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
