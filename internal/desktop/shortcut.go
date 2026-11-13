package desktop

import (
	"os"
	"path/filepath"
)

var (
	configuredDataDir string
	iconData          []byte
)

// Configure remembers the portable data directory and the icon bytes used for
// desktop shortcuts.
func Configure(dataDir string, icon []byte) {
	configuredDataDir = dataDir
	iconData = icon
}

// ensureIconFile writes the Windows shortcut icon into the portable data
// directory once so .lnk files do not depend on the executable's resource
// section.
func ensureIconFile() (string, error) {
	if len(iconData) == 0 {
		return "", nil
	}
	dir := configuredDataDir
	if dir == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", err
		}
		dir = filepath.Dir(executable)
	}
	path := filepath.Join(dir, "resume-helper.ico")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, iconData, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
