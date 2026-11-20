//go:build android

package server

import (
	"os/exec"
	"strings"
)

// detectPlatform reports the Android release and API level. The form factor
// comes from the build characteristics, which contains "tablet" on tablets.
func detectPlatform() PlatformInfo {
	info := PlatformInfo{Kind: "phone", OS: "android", Name: "Android"}

	if strings.Contains(strings.ToLower(androidProperty("ro.build.characteristics")), "tablet") {
		info.Kind = "tablet"
	}

	release := androidProperty("ro.build.version.release")
	sdk := androidProperty("ro.build.version.sdk")
	switch {
	case release != "" && sdk != "":
		info.Version = release + " (SDK " + sdk + ")"
	case release != "":
		info.Version = release
	case sdk != "":
		info.Version = "SDK " + sdk
	}
	return info
}

func androidProperty(name string) string {
	for _, binary := range []string{"getprop", "/system/bin/getprop"} {
		output, err := exec.Command(binary, name).Output()
		if err != nil {
			continue
		}
		return strings.TrimSpace(string(output))
	}
	return ""
}
