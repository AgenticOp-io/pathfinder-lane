//go:build !windows

package crtapp

func CreateShortcuts(installerExe string) error { return nil }

func RemoveShortcuts() {}
