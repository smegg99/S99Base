// release/pad/validate.go
package pad

import (
	"fmt"
	"net/mail"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// maxURL counts characters between "://" and any required extension.
const maxURL = 120

var (
	imageExts    = []string{"gif", "jpg", "png"}
	padExts      = []string{"xml", "cgi", "php", "asp"}
	downloadExts = []string{"air", "apk", "app", "bin", "com", "exe", "dmg", "hqx", "gz", "gzip", "jar", "msi", "rar", "sea", "sh", "sis", "sit", "sitx", "tar", "taz", "tgx", "tgz", "uu", "z", "zip"}

	releaseStatuses    = []string{"Major Update", "Minor Update", "New Release", "Beta", "Alpha", "Media Only"}
	installSupports    = []string{"Install and Uninstall", "Install Only", "No Install Support", "Uninstall Only"}
	specificCategories = []string{"Audio", "Business", "Development Tools", "Education", "Games", "Graphics", "Home/Hobby", "Internet", "Miscellaneous", "Screen Savers", "Utilities"}

	versionChars = regexp.MustCompile(`^[a-zA-Z0-9_.\-]{1,15}$`)
	emailParts   = regexp.MustCompile(`^.{2,30}@.{2,63}\..{2,20}$`)
)

// check returns the first failure in field order.
func check(p Product, in Installer) error {
	for _, err := range []error{
		checkText("Product.Publisher", p.Publisher, 2, 40, false),
		checkURL("Product.PublisherURL", p.PublisherURL, nil),
		checkEmail("Product.Email", p.Email),
		checkText("Product.Name", p.Name, 1, 40, false),
		checkEnum("Product.ReleaseStatus", p.ReleaseStatus, releaseStatuses, false),
		checkEnum("Product.InstallSupport", p.InstallSupport, installSupports, false),
		checkList("Product.OS", p.OS),
		checkList("Product.Languages", p.Languages),
		checkCategory("Product.Category", p.Category),
		checkEnum("Product.SpecificCategory", p.SpecificCategory, specificCategories, true),
		checkText("Product.Requirements", p.Requirements, 0, 100, false),
		checkText("Product.Keywords", p.Keywords, 0, 250, false),
		checkText("Product.Desc45", p.Desc45, 1, 45, false),
		checkText("Product.Desc80", p.Desc80, 1, 80, false),
		checkText("Product.Desc250", p.Desc250, 1, 250, false),
		checkText("Product.Desc450", p.Desc450, 1, 450, false),
		checkText("Product.Desc2000", p.Desc2000, 1, 2000, true),
		checkURL("Product.InfoURL", p.InfoURL, nil),
		checkURL("Product.ScreenshotURL", p.ScreenshotURL, imageExts),
		checkURL("Product.IconURL", p.IconURL, imageExts),
		checkURL("Product.PADURL", p.PADURL, padExts),
		checkText("Product.Permissions", p.Permissions, 0, 2000, true),
		checkVersion("Installer.Version", in.Version),
		checkDate("Installer.Published", in.Published),
		checkSize("Installer.Size", in.Size),
		checkURL("Installer.URL", in.URL, downloadExts),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func fail(field, format string, args ...any) error {
	return &FieldError{Field: field, Reason: fmt.Sprintf(format, args...)}
}

func checkText(field, s string, min, max int, multiline bool) error {
	if err := checkChars(field, s, multiline); err != nil {
		return err
	}
	n := utf8.RuneCountInString(s)
	switch {
	case min > 0 && strings.TrimSpace(s) == "":
		return fail(field, "is required")
	case n < min:
		return fail(field, "has %d characters, minimum %d", n, min)
	case n > max:
		return fail(field, "has %d characters, maximum %d", n, max)
	}
	return nil
}

// checkChars applies 3.11's "no < and no tab" rule plus what XML 1.0 cannot encode.
func checkChars(field, s string, multiline bool) error {
	if !utf8.ValidString(s) {
		return fail(field, "is not valid UTF-8")
	}
	for _, r := range s {
		switch {
		case r == '<':
			return fail(field, "contains <")
		case r == '\t':
			return fail(field, "contains a tab")
		case r == '\n' || r == '\r':
			if !multiline {
				return fail(field, "must be one line")
			}
			if r == '\r' {
				return fail(field, "must break lines with LF only")
			}
		case r < 0x20 || r == 0xFFFE || r == 0xFFFF:
			return fail(field, "contains U+%04X", r)
		}
	}
	if multiline && strings.Contains(s, "\n\n\n") {
		return fail(field, "has more than one blank line in a row")
	}
	return nil
}

func checkEnum(field, s string, allowed []string, optional bool) error {
	switch {
	case slices.Contains(allowed, s), optional && s == "":
		return nil
	case s == "":
		return fail(field, "is required")
	}
	return fail(field, "%q is not one of %s", s, strings.Join(allowed, ", "))
}

// checkList checks names that are joined with commas, such as OS and language names.
func checkList(field string, items []string) error {
	if len(items) == 0 {
		return fail(field, "is required")
	}
	for i, item := range items {
		name := fmt.Sprintf("%s[%d]", field, i)
		if err := checkChars(name, item, false); err != nil {
			return err
		}
		if strings.TrimSpace(item) == "" || strings.Contains(item, ",") {
			return fail(name, "must be one non-empty name without commas")
		}
	}
	return nil
}

func checkCategory(field, s string) error {
	if err := checkChars(field, s, false); err != nil {
		return err
	}
	class, sub, ok := strings.Cut(s, "::")
	if !ok || strings.TrimSpace(class) == "" || strings.TrimSpace(sub) == "" {
		return fail(field, `must be "Class::Subclass"`)
	}
	return nil
}

// checkEmail takes a bare address shaped like 3.11's 2-30@2-63.2-20 pattern.
func checkEmail(field, s string) error {
	if s == "" {
		return nil
	}
	if err := checkChars(field, s, false); err != nil {
		return err
	}
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Name != "" || addr.Address != s || !emailParts.MatchString(s) {
		return fail(field, "must be a bare address of 2-30 characters @ 2-63 . 2-20")
	}
	return nil
}

func checkVersion(field, v string) error {
	if !versionChars.MatchString(v) {
		return fail(field, "must be 1-15 of A-Z a-z 0-9 . _ -")
	}
	return nil
}

func checkDate(field string, t time.Time) error {
	if t.IsZero() {
		return fail(field, "is required")
	}
	if y := t.UTC().Year(); y < 1900 || y > 2199 {
		return fail(field, "year %d is outside 1900-2199", y)
	}
	return nil
}

// checkSize keeps every representation inside 3.11: 3-16 digits of bytes and 8 characters of MB.
func checkSize(field string, n int64) error {
	switch {
	case n < 100:
		return fail(field, "must be at least 100 bytes")
	case n > 9_999_999_999_999_999:
		return fail(field, "must have at most 16 digits")
	}
	if _, _, mb := sizes(n); len(mb) > 8 {
		return fail(field, "is %s MB, longer than 8 characters", mb)
	}
	return nil
}

func checkURL(field, raw string, exts []string) error {
	if raw == "" {
		return fail(field, "is required")
	}
	if err := checkChars(field, raw, false); err != nil {
		return err
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		!strings.HasPrefix(raw, u.Scheme+"://") || strings.ContainsRune(raw, ' ') {
		return fail(field, "must be an absolute http or https URL")
	}
	if u.User != nil {
		return fail(field, "must not carry credentials")
	}
	if !publicHost(u.Hostname()) {
		return fail(field, "host %q is not public", u.Hostname())
	}
	rest := strings.TrimPrefix(raw, u.Scheme+"://")
	if len(exts) > 0 {
		i := slices.IndexFunc(exts, func(ext string) bool { return strings.HasSuffix(rest, "."+ext) })
		if i < 0 {
			return fail(field, "must end in .%s", strings.Join(exts, ", ."))
		}
		rest = strings.TrimSuffix(rest, "."+exts[i])
	}
	if n := utf8.RuneCountInString(rest); n < 2 || n > maxURL {
		return fail(field, "has %d characters after the scheme, allowed 2-%d", n, maxURL)
	}
	return nil
}

func publicHost(host string) bool {
	if addr, err := netip.ParseAddr(host); err == nil {
		addr = addr.Unmap()
		return addr.IsGlobalUnicast() && !addr.IsPrivate()
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return strings.Contains(host, ".") && !strings.HasSuffix(host, ".localhost")
}
