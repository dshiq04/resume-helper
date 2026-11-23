//go:build !windows && !linux && !darwin && !android && !ios

package server

import "runtime"

// detectPlatform covers the remaining Go ports so the about card always has a
// value to show.
func detectPlatform() PlatformInfo {
	return PlatformInfo{Kind: "computer", OS: runtime.GOOS, Name: runtime.GOOS}
}
