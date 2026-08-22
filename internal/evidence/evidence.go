// Ticket evidence pack: zip open-terminal scrollbacks for change tickets.
package evidence

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File is one scrollback capture to include.
type File struct {
	Name    string // path inside the zip
	Content []byte
}

// WriteZip creates destZip with the given files plus a manifest.txt.
func WriteZip(destZip string, ticket string, files []File) error {
	if len(files) == 0 {
		return fmt.Errorf("no scrollbacks to pack")
	}
	if err := os.MkdirAll(filepath.Dir(destZip), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()

	var manifest strings.Builder
	fmt.Fprintf(&manifest, "PathfinderSSH MSP evidence pack\n")
	fmt.Fprintf(&manifest, "Created: %s\n", time.Now().Format(time.RFC3339))
	if t := strings.TrimSpace(ticket); t != "" {
		fmt.Fprintf(&manifest, "Ticket: %s\n", t)
	}
	fmt.Fprintf(&manifest, "Files: %d\n\n", len(files))
	for _, file := range files {
		name := sanitizeName(file.Name)
		fmt.Fprintf(&manifest, "- %s (%d bytes)\n", name, len(file.Content))
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(file.Content); err != nil {
			return err
		}
	}
	mw, err := zw.Create("manifest.txt")
	if err != nil {
		return err
	}
	_, err = mw.Write([]byte(manifest.String()))
	return err
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, `\`, `_`)
	name = strings.ReplaceAll(name, `/`, `_`)
	name = strings.TrimSpace(name)
	if name == "" {
		name = "scrollback.txt"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".txt") {
		name += ".txt"
	}
	return name
}
