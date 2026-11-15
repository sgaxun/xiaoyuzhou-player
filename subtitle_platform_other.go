//go:build !windows && !darwin

package main

// Platforms without a native subtitle implementation leave the overlay hidden.
func (b *Backend) platformSubtitle(text string, progress float64) error { return nil }

func (b *Backend) verifyPlatformSubtitle() error { return nil }

func platformSubtitleDisplays() ([]SubtitleDisplay, error) { return []SubtitleDisplay{}, nil }
