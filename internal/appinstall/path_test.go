package appinstall

import "testing"

func TestSameFileIgnoresExtendedPathPrefix(t *testing.T) {
	a := `\\?\C:\Users\david\AppData\Local\PathfinderSSH-MSP\bin\pathfinder.exe`
	b := `C:\Users\david\AppData\Local\PathfinderSSH-MSP\bin\pathfinder.exe`
	if !SameFile(a, b) {
		t.Fatal("expected same file")
	}
}

func TestSameFileSlashNormalized(t *testing.T) {
	a := `C:/Users/david/AppData/Local/PathfinderSSH-MSP/bin/pathfinder.exe`
	b := `C:\Users\david\AppData\Local\PathfinderSSH-MSP\bin\pathfinder.exe`
	if !SameFile(a, b) {
		t.Fatal("expected same file")
	}
}
