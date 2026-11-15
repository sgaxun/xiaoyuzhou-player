//go:build darwin

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

var notchSubtitle struct {
	sync.Mutex
	cmd     *exec.Cmd
	input   io.WriteCloser
	replies chan []byte
	status  struct {
		Visible            bool    `json:"visible"`
		IgnoresMouseEvents bool    `json:"ignoresMouseEvents"`
		Progress           float64 `json:"progress"`
	}
}

func stopNotchSubtitleLocked() {
	s := &notchSubtitle
	if s.input != nil {
		_ = s.input.Close()
	}
	if s.cmd != nil {
		_ = s.cmd.Process.Kill()
	}
	s.cmd, s.input, s.replies = nil, nil, nil
}

func closePlatformSubtitle() {
	notchSubtitle.Lock()
	defer notchSubtitle.Unlock()
	stopNotchSubtitleLocked()
}

func (b *Backend) platformSubtitle(text string, progress float64) error {
	s := &notchSubtitle
	s.Lock()
	defer s.Unlock()
	text = normalizeSubtitle(text)
	if text == "" && s.cmd == nil {
		return nil
	}
	if math.IsNaN(progress) || math.IsInf(progress, 0) {
		progress = 0
	}
	progress = max(0, min(1, progress))
	if s.cmd == nil {
		exe, err := os.Executable()
		if err != nil {
			return err
		}
		path := filepath.Join(filepath.Dir(exe), "../Resources/bin/notch-helper")
		if _, err := os.Stat(path); err != nil {
			path = "resources/bin/notch-helper"
		}
		cmd := exec.Command(path)
		input, err := cmd.StdinPipe()
		if err != nil {
			return err
		}
		output, err := cmd.StdoutPipe()
		if err != nil {
			_ = input.Close()
			return err
		}
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			_ = input.Close()
			_ = output.Close()
			return fmt.Errorf("启动刘海字幕: %w", err)
		}
		replies := make(chan []byte, 1)
		s.cmd, s.input, s.replies = cmd, input, replies
		go func() {
			scanner := bufio.NewScanner(output)
			for scanner.Scan() {
				replies <- append([]byte(nil), scanner.Bytes()...)
			}
			close(replies)
			_ = cmd.Wait()
		}()
	}
	raw, err := json.Marshal(map[string]any{"text": text, "progress": progress})
	if err == nil {
		_, err = s.input.Write(append(raw, '\n'))
	}
	if err != nil {
		stopNotchSubtitleLocked()
		return err
	}
	select {
	case reply, ok := <-s.replies:
		if !ok {
			stopNotchSubtitleLocked()
			return fmt.Errorf("刘海字幕进程已退出")
		}
		if err := json.Unmarshal(reply, &s.status); err != nil {
			stopNotchSubtitleLocked()
			return err
		}
		if s.status.Visible != (text != "") {
			return fmt.Errorf("刘海字幕可见状态异常")
		}
		return nil
	case <-time.After(3 * time.Second):
		stopNotchSubtitleLocked()
		return fmt.Errorf("刘海字幕响应超时")
	}
}

func (b *Backend) verifyPlatformSubtitle() error {
	notchSubtitle.Lock()
	defer notchSubtitle.Unlock()
	if !notchSubtitle.status.Visible || !notchSubtitle.status.IgnoresMouseEvents {
		return fmt.Errorf("刘海字幕未显示或未启用鼠标穿透")
	}
	return nil
}
