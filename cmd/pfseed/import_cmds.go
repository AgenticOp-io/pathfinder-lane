package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

func defaultCRTConfig() string {
	return crtimport.DefaultConfig()
}

func cmdImportSecureCRT(args []string) error {
	fs := flag.NewFlagSet("import-securecrt", flag.ContinueOnError)
	config := fs.String("config", defaultCRTConfig(), "VanDyke Config directory (contains Sessions/)")
	path := fs.String("sessions", "", "Pathfinder sessions.yaml path")
	dry := fs.Bool("dry-run", false, "parse and report without writing")
	preview := fs.Bool("preview", false, "print JSON summary only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*config) == "" {
		return fmt.Errorf("-config is required (VanDyke Config folder)")
	}

	list, err := crtimport.Import(*config)
	if err != nil {
		return err
	}
	folders, supported, skipped := crtimport.Folders(list)

	summary := map[string]any{
		"config":    *config,
		"found":     len(list),
		"supported": supported,
		"skipped":   skipped,
		"folders":   len(folders),
		"dryRun":    *dry || *preview,
	}

	if *preview || *dry {
		counts := make([]map[string]any, 0, len(folders))
		for _, f := range folders {
			counts = append(counts, map[string]any{"folder": f.Name, "sessions": len(f.Sessions)})
		}
		summary["folderList"] = counts
		return json.NewEncoder(os.Stdout).Encode(summary)
	}

	p := *path
	if p == "" {
		p = defaultSessionsPath()
	}
	tr, err := sessions.LoadFile(p)
	if err != nil {
		return err
	}
	imp := tr.ImportFolders(folders)
	if err := sessions.SaveFile(p, tr); err != nil {
		return err
	}
	summary["path"] = p
	summary["added"] = imp.Added
	summary["skippedExisting"] = imp.Skipped
	summary["describe"] = imp.Describe()
	return json.NewEncoder(os.Stdout).Encode(summary)
}

func cmdImportCSV(args []string) error {
	fs := flag.NewFlagSet("import-csv", flag.ContinueOnError)
	file := fs.String("file", "", "CSV path (VanDyke-style header)")
	path := fs.String("sessions", "", "Pathfinder sessions.yaml path")
	dry := fs.Bool("dry-run", false, "parse without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("-file is required")
	}
	f, err := os.Open(*file)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}
	if len(rows) < 2 {
		return fmt.Errorf("csv needs a header and at least one row")
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	col := func(row []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	byFolder := map[string][]sessions.Node{}
	skipped := 0
	for _, row := range rows[1:] {
		host := col(row, "hostname")
		if host == "" {
			skipped++
			continue
		}
		name := col(row, "session_name")
		if name == "" {
			name = host
		}
		folder := col(row, "folder")
		if folder == "" {
			folder = "CSV"
		} else {
			folder = strings.ReplaceAll(folder, "/", " / ")
			folder = strings.ReplaceAll(folder, "\\", " / ")
		}
		proto := strings.ToLower(col(row, "protocol"))
		user := col(row, "username")
		portStr := col(row, "port")
		notes := col(row, "description")

		n := sessions.Defaults()
		n.Name = name
		n.Host = host
		n.Username = user
		n.Notes = notes
		if notes == "" {
			n.Notes = "Imported from CSV"
		}
		switch {
		case strings.Contains(proto, "telnet"):
			n.Transport = sessions.TransportTelnet
			n.Port = 23
		default:
			n.Transport = sessions.TransportSSH
			n.Port = 22
			n.AuthType = sessions.AuthPassword
		}
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				n.Port = p
			}
		}
		byFolder[folder] = append(byFolder[folder], n.Normalize())
	}

	summary := map[string]any{
		"file":      *file,
		"supported": len(rows) - 1 - skipped,
		"skipped":   skipped,
		"folders":   len(byFolder),
		"dryRun":    *dry,
	}
	if *dry {
		return json.NewEncoder(os.Stdout).Encode(summary)
	}
	p := *path
	if p == "" {
		p = defaultSessionsPath()
	}
	tr, err := sessions.LoadFile(p)
	if err != nil {
		return err
	}
	folders := make([]sessions.Folder, 0, len(byFolder))
	for name, nodes := range byFolder {
		folders = append(folders, sessions.Folder{Name: name, Sessions: nodes})
	}
	imp := tr.ImportFolders(folders)
	if err := sessions.SaveFile(p, tr); err != nil {
		return err
	}
	summary["path"] = p
	summary["added"] = imp.Added
	summary["skippedExisting"] = imp.Skipped
	summary["describe"] = imp.Describe()
	return json.NewEncoder(os.Stdout).Encode(summary)
}
