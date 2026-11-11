//go:build windows

package desktop

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

const shortcutFileName = "履途 ResumeTrail.lnk"

func ShortcutSupported() bool {
	return true
}

func ShortcutPath() (string, error) {
	desktopDir, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return "", fmt.Errorf("locate desktop folder: %w", err)
	}
	return filepath.Join(desktopDir, shortcutFileName), nil
}

func ShortcutExists() (bool, error) {
	path, err := ShortcutPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CreateShortcut() error {
	shortcutPath, err := ShortcutPath()
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	executable = filepath.Clean(executable)
	workingDir := filepath.Dir(executable)

	iconLocation := executable + ",0"
	if iconPath, iconErr := ensureIconFile(); iconErr == nil && iconPath != "" {
		iconLocation = iconPath
	}

	script := fmt.Sprintf(
		`$ErrorActionPreference='Stop'; try { $shell = New-Object -ComObject WScript.Shell; $shortcut = $shell.CreateShortcut(%s); $shortcut.TargetPath = %s; $shortcut.WorkingDirectory = %s; $shortcut.Description = 'ResumeTrail'; $shortcut.IconLocation = %s; $shortcut.Save(); exit 0 } catch { Write-Error $_.Exception.Message; exit 1 }`,
		powerShellQuote(shortcutPath),
		powerShellQuote(executable),
		powerShellQuote(workingDir),
		powerShellQuote(iconLocation),
	)
	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-Command", script,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("create desktop shortcut: %s", message)
	}
	return nil
}

func powerShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
