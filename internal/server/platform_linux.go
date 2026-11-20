//go:build linux && !android

package server

import (
	"bufio"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// detectPlatform reports the Linux distribution. PRETTY_NAME is the
// distribution's own display name, for example "Ubuntu 24.04.1 LTS".
func detectPlatform() PlatformInfo {
	info := PlatformInfo{Kind: "computer", OS: "linux", Name: "Linux"}

	values := readOSRelease()
	prettyName := strings.TrimSpace(values["PRETTY_NAME"])
	name := strings.TrimSpace(values["NAME"])
	version := strings.TrimSpace(values["VERSION_ID"])
	if version == "" {
		version = strings.TrimSpace(values["VERSION"])
	}
	switch {
	case prettyName != "":
		info.Name = prettyName
	case name != "":
		info.Name = name
	}
	// PRETTY_NAME normally already includes the release number. Only show the
	// version separately when it adds information.
	if version != "" && !strings.Contains(info.Name, version) {
		info.Version = version
	}
	if info.Version == "" && info.Name == "Linux" {
		var uts unix.Utsname
		if err := unix.Uname(&uts); err == nil {
			info.Version = charsToString(uts.Release[:])
		}
	}
	return info
}

func readOSRelease() map[string]string {
	values := map[string]string{}
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
		_ = file.Close()
		if len(values) > 0 {
			return values
		}
	}
	return values
}

// charsToString converts a NUL terminated byte array from unix.Utsname.
func charsToString(value []byte) string {
	if index := strings.IndexByte(string(value), 0); index >= 0 {
		return string(value[:index])
	}
	return string(value)
}
