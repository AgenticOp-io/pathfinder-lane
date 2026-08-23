package sessions

import "strings"

// Subtree returns a tree containing only the folder at folderPath as root.
func (t Tree) Subtree(folderPath string) (Tree, error) {
	f, err := t.FolderAt(folderPath)
	if err != nil {
		return Tree{}, err
	}
	copy := *f
	return Tree{Version: FileVersion, Folders: []Folder{copy}}, nil
}

// SessionInFolder finds a session by name within one folder (not nested subfolders).
func (t Tree) SessionInFolder(folder, name string) (Node, bool) {
	f, err := t.FolderAt(folder)
	if err != nil {
		return Node{}, false
	}
	name = strings.TrimSpace(name)
	for _, n := range f.Sessions {
		if strings.EqualFold(strings.TrimSpace(n.Name), name) || strings.EqualFold(n.Label(), name) {
			return n, true
		}
	}
	return Node{}, false
}
