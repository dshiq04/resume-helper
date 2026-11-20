//go:build ios

package server

import (
	"strings"

	"golang.org/x/sys/unix"
)

// detectPlatform reports iOS or iPadOS with the product version. The hardware
// identifier distinguishes an iPad ("iPad13,1") from an iPhone.
func detectPlatform() PlatformInfo {
	info := PlatformInfo{Kind: "phone", OS: "ios", Name: "iOS"}

	model, _ := unix.Sysctl("hw.machine")
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "ipad") {
		info.Kind = "tablet"
		info.OS = "ipados"
		info.Name = "iPadOS"
	}

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
