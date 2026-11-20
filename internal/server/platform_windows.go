//go:build windows

package server

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// detectPlatform reads the Windows release, edition and build number. The
// registry still reports Windows 11 as "Windows 10" in ProductName, so the
// display name is corrected from the build number.
func detectPlatform() PlatformInfo {
	info := PlatformInfo{Kind: "computer", OS: "windows", Name: "Windows"}

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return info
	}
	defer key.Close()

	productName, _, _ := key.GetStringValue("ProductName")
	displayVersion, _, _ := key.GetStringValue("DisplayVersion")
	build, _, _ := key.GetStringValue("CurrentBuildNumber")
	if build == "" {
		build, _, _ = key.GetStringValue("CurrentBuild")
	}
	buildNumber, _ := strconv.Atoi(build)

	if buildNumber >= 22000 && strings.HasPrefix(productName, "Windows 10") {
		productName = "Windows 11" + strings.TrimPrefix(productName, "Windows 10")
	}
	if productName == "" {
		switch {
		case buildNumber >= 22000:
			productName = "Windows 11"
		case buildNumber >= 10240:
			productName = "Windows 10"
		}
	}
	if productName != "" {
		info.Name = productName
	}

	major, minor := uint64(10), uint64(0)
	if value, _, err := key.GetIntegerValue("CurrentMajorVersionNumber"); err == nil && value > 0 {
		major = value
	}
	if value, _, err := key.GetIntegerValue("CurrentMinorVersionNumber"); err == nil && value > 0 {
		minor = value
	}
	revision, _, _ := key.GetIntegerValue("UBR")

	parts := make([]string, 0, 3)
	if displayVersion != "" {
		parts = append(parts, displayVersion)
	}
	if build != "" {
		version := fmt.Sprintf("%d.%d.%s", major, minor, build)
		if revision > 0 {
			version += fmt.Sprintf(".%d", revision)
		}
		parts = append(parts, "("+version+")")
	}
	info.Version = strings.Join(parts, " ")
	return info
}
