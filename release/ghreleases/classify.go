// release/ghreleases/classify.go

package ghreleases

import (
	"path/filepath"
	"regexp"
	"strings"
)

// webMarker matches "web" as a whole name part, so "webview" is no marker.
var webMarker = regexp.MustCompile(`(^|[-_.])web([-_.]|$)`)

// kindPlatform maps kinds whose extension alone names the platform; .zip is absent.
var kindPlatform = map[string]string{
	"appimage":    "linux",
	"deb":         "linux",
	"rpm":         "linux",
	"pkg.tar.zst": "linux",
	"tar.gz":      "linux",
	"installer":   "windows",
	"exe":         "windows",
	"msix":        "windows",
	"dmg":         "macos",
	"apk":         "android",
	"aab":         "android",
}

// classify infers platform, arch and kind from an asset filename.
func classify(name string) (platform, arch, kind string) {
	n := strings.ToLower(name)
	kind = kindOf(n)
	return platformOf(n, kind), archOf(n), kind
}

func kindOf(n string) string {
	switch {
	case strings.HasSuffix(n, ".appimage"):
		return "appimage"
	case strings.HasSuffix(n, ".deb"):
		return "deb"
	case strings.HasSuffix(n, ".rpm"):
		return "rpm"
	case strings.HasSuffix(n, ".pkg.tar.zst"):
		return "pkg.tar.zst"
	case strings.HasSuffix(n, ".tar.zst"):
		return "tar.zst"
	case strings.HasSuffix(n, ".apk"):
		return "apk"
	case strings.HasSuffix(n, ".aab"):
		return "aab"
	case strings.HasSuffix(n, ".tar.gz"), strings.HasSuffix(n, ".tgz"):
		return "tar.gz"
	// The NSIS installer is <app>-<ver>-windows-<arch>-installer.exe; a plain
	// .exe is the portable build.
	case strings.HasSuffix(n, "-installer.exe"):
		return "installer"
	case strings.HasSuffix(n, ".exe"):
		return "exe"
	case strings.HasSuffix(n, ".msix"):
		return "msix"
	case strings.HasSuffix(n, ".dmg"):
		return "dmg"
	case strings.HasSuffix(n, ".zip"):
		return "zip"
	default:
		ext := strings.TrimPrefix(filepath.Ext(n), ".")
		// For <app>-<ver>-<os>-<arch>, Ext returns ".0-linux-amd64"; a real extension has no dash.
		if strings.Contains(ext, "-") {
			return "binary"
		}
		return ext
	}
}

// platformOf prefers a filename marker over the extension's default.
func platformOf(n, kind string) string {
	if p := platformMarker(n); p != "" {
		return p
	}
	if p, ok := kindPlatform[kind]; ok {
		return p
	}
	if kind == "zip" {
		// An unmarked zip is a Windows portable build by convention.
		return "windows"
	}
	return PlatformOther
}

// platformMarker checks macOS, then Windows, android, web and linux.
func platformMarker(n string) string {
	switch {
	case strings.Contains(n, "darwin"), strings.Contains(n, "macos"),
		strings.Contains(n, "osx"), strings.Contains(n, "mac"):
		return "macos"
	case strings.Contains(n, "windows"), strings.Contains(n, "win"):
		return "windows"
	case strings.Contains(n, "android"):
		return "android"
	case webMarker.MatchString(n):
		return "web"
	case strings.Contains(n, "linux"):
		return "linux"
	}
	return ""
}

// archOf checks x86_64 before arm64, so apple-intel is x86_64 and apple alone is no arch.
func archOf(n string) string {
	switch {
	case strings.Contains(n, "universal"):
		return "universal"
	case strings.Contains(n, "x86_64"), strings.Contains(n, "amd64"),
		strings.Contains(n, "x64"), strings.Contains(n, "intel"):
		return "x86_64"
	case strings.Contains(n, "arm64"), strings.Contains(n, "aarch64"),
		strings.Contains(n, "silicon"):
		return "arm64"
	}
	return ""
}
