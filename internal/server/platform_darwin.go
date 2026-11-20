//go:build darwin && !ios

package server

import (
	"strings"

	"golang.org/x/sys/unix"
)

// detectPlatform reports the macOS product version (for example 15.1) and its
// build number (for example 24B83).
func detectPlatform() PlatformInfo {
	info := PlatformInfo{Kind: "computer", OS: "macos", Name: "macOS"}

	product, _ := unix.Sysctl("kern.osproductversion")
	build, _ := unix.Sysctl("kern.osversion")
	product = strings.TrimSpace(product)
	build = strings.TrimSpace(build)

	switch {
	case product != "" && build != "":
		info.Version = product + " (" + build + ")"
	case product != "":
		info.Version = product
	case build != "":
		info.Version = build
	}
	return info
}
