// release/mdlocale/mdlocale.go

// Package mdlocale splits markdown into per-locale sections opened by <!-- lang:xx --> lines.
package mdlocale

import (
	"regexp"
	"strings"
)

// marker must sit on its own line; the trailing class admits the \r of GitHub's CRLF bodies.
var marker = regexp.MustCompile(`(?im)^[ \t]*<!--[ \t]*lang:[ \t]*([A-Za-z0-9_-]+)[ \t]*-->[ \t\r]*$`)

type Sections struct {
	// byLocale is keyed by lowercased tag.
	byLocale map[string]string
	// untagged is the text before the first marker.
	untagged string
}

// Split never fails; a document with no markers is untagged in full.
func Split(md string) Sections {
	s := Sections{byLocale: map[string]string{}}

	loc := marker.FindAllStringSubmatchIndex(md, -1)
	if len(loc) == 0 {
		s.untagged = strings.TrimSpace(md)
		return s
	}

	s.untagged = strings.TrimSpace(md[:loc[0][0]])

	for i, m := range loc {
		tag := strings.ToLower(strings.ReplaceAll(md[m[2]:m[3]], "_", "-"))

		end := len(md)
		if i+1 < len(loc) {
			end = loc[i+1][0]
		}
		body := strings.TrimSpace(md[m[1]:end])

		// A repeated tag appends rather than overwrites.
		if prev, ok := s.byLocale[tag]; ok && prev != "" {
			if body != "" {
				body = prev + "\n\n" + body
			} else {
				body = prev
			}
		}
		s.byLocale[tag] = body
	}

	return s
}

// Resolve returns the locale's section, its base tag's (pl-PL reads pl), the fallback's, or the untagged text in fallback.
func (s Sections) Resolve(locale, fallback string) (text, lang string) {
	for _, tag := range []string{locale, baseTag(locale), fallback, baseTag(fallback)} {
		tag = strings.ToLower(strings.ReplaceAll(tag, "_", "-"))
		if body := s.byLocale[tag]; tag != "" && body != "" {
			return body, tag
		}
	}
	return s.untagged, fallback
}

// baseTag is the language part of a tag such as pl-PL or pt_BR.
func baseTag(tag string) string {
	base, _, _ := strings.Cut(strings.ReplaceAll(tag, "_", "-"), "-")
	return base
}
