package appinstall

import (
	"fmt"
	"os"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/product"
)

// Uninstall removes Desktop/Start Menu shortcuts and the LocalAppData install tree.
func Uninstall() error {
	_ = RemoveShortcuts()
	root := Root()
	base := strings.ToLower(filepathBase(root))
	if root == "" || root == "." || (base != strings.ToLower(product.InstallDir) && !strings.Contains(base, "pathfinderssh")) {
		return fmt.Errorf("refusing to remove ambiguous install root %q", root)
	}
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func filepathBase(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}

// RemoveShortcuts deletes Pathfinder .lnk files from Desktop and Start Menu.
func RemoveShortcuts() error {
	return removeShortcuts()
}
