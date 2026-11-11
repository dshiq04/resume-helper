//go:build !windows

package desktop

import "errors"

func ShortcutSupported() bool {
	return false
}

func ShortcutPath() (string, error) {
	return "", nil
}

func ShortcutExists() (bool, error) {
	return false, nil
}

func CreateShortcut() error {
	return errors.New("desktop shortcuts are only supported on Windows")
}
