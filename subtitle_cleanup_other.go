//go:build !darwin

package main

func closePlatformSubtitle() {}

func platformSubtitleDisplays() ([]SubtitleDisplay, error) { return []SubtitleDisplay{}, nil }
