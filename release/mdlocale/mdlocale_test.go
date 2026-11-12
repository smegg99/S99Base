// release/mdlocale/mdlocale_test.go

package mdlocale

import (
	"strings"
	"testing"
)

// Unmarked prose belongs to every locale, in the fallback language.
func TestNoMarkersIsUntaggedForEveryLocale(t *testing.T) {
	const body = "## What's Changed\n* Fixed a thing by @smegg99"

	s := Split(body)

	for _, locale := range []string{"en", "pl", "de"} {
		text, lang := s.Resolve(locale, "en")
		if text != body || lang != "en" {
			t.Errorf("%s: (%q, %q), want the whole document in en", locale, text, lang)
		}
	}
}

func TestSplitsOnMarkers(t *testing.T) {
	s := Split("<!-- lang:en -->\nWhat's new\n\n<!-- lang:pl -->\nCo nowego")

	if en, lang := s.Resolve("en", "en"); en != "What's new" || lang != "en" {
		t.Errorf("en = (%q, %q)", en, lang)
	}
	if pl, lang := s.Resolve("pl", "en"); pl != "Co nowego" || lang != "pl" {
		t.Errorf("pl = (%q, %q)", pl, lang)
	}
}

// pl-PL reads the pl section.
func TestRegionalLocaleFindsItsBaseTag(t *testing.T) {
	s := Split("<!-- lang:en -->\nEnglish\n<!-- lang:pl -->\nPolski")

	if text, lang := s.Resolve("pl-PL", "en"); text != "Polski" || lang != "pl" {
		t.Errorf("pl-PL = (%q, %q), want the pl section", text, lang)
	}
}

// The language comes back as the section's own lowercase tag, however the caller spelled it.
func TestResolveReportsTheSectionsTag(t *testing.T) {
	s := Split("<!-- lang:en -->\nEnglish\n<!-- lang:pl -->\nPolski")

	if text, lang := Split("<!-- lang:pt_BR -->\nOla").Resolve("pt-BR", "en"); text != "Ola" || lang != "pt-br" {
		t.Errorf("pt_BR marker = (%q, %q), want the section as pt-br", text, lang)
	}
	for _, locale := range []string{"PL", "pl_PL", "pl-PL"} {
		if _, lang := s.Resolve(locale, "en"); lang != "pl" {
			t.Errorf("%s: lang = %q, want pl", locale, lang)
		}
	}
}

// A body that starts with a marker has no preamble, so a locale without a section reads the fallback's.
func TestMissingLocaleReadsTheFallbackSection(t *testing.T) {
	s := Split("<!-- lang:en -->\nEnglish\n<!-- lang:pl -->\nPolski")

	if text, lang := s.Resolve("de", "en"); text != "English" || lang != "en" {
		t.Errorf("de = (%q, %q), want the en section", text, lang)
	}
}

func TestUnknownLocaleFallsBackToUntagged(t *testing.T) {
	s := Split("Shared preamble.\n\n<!-- lang:en -->\nEnglish only")

	if text, lang := s.Resolve("pl", "de"); text != "Shared preamble." || lang != "de" {
		t.Errorf("pl = (%q, %q), want the untagged preamble", text, lang)
	}
}

// A marker with nothing under it is treated as absent.
func TestEmptySectionCountsAsNoSection(t *testing.T) {
	s := Split("Preamble.\n\n<!-- lang:pl -->\n\n<!-- lang:en -->\nEnglish")

	if text, lang := s.Resolve("pl", "de"); text != "Preamble." || lang != "de" {
		t.Errorf("pl = (%q, %q), want the untagged preamble", text, lang)
	}
}

func TestMarkerSyntaxIsForgiving(t *testing.T) {
	bodies := map[string]string{
		"canonical":       "<!-- lang:pl -->\nCo nowego",
		"no spaces":       "<!--lang:pl-->\nCo nowego",
		"extra spaces":    "<!--   lang:   pl   -->\nCo nowego",
		"uppercase tag":   "<!-- lang:PL -->\nCo nowego",
		"uppercase LANG":  "<!-- LANG:pl -->\nCo nowego",
		"leading indent":  "  <!-- lang:pl -->\nCo nowego",
		"trailing spaces": "<!-- lang:pl -->   \nCo nowego",
		"crlf":            "<!-- lang:pl -->\r\nCo nowego",
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			text, lang := Split(body).Resolve("pl", "en")
			if lang != "pl" || strings.TrimSpace(text) != "Co nowego" {
				t.Errorf("(%q, %q), want the pl section", text, lang)
			}
		})
	}
}

func TestMarkerMidLineIsNotAMarker(t *testing.T) {
	s := Split("Write `<!-- lang:pl -->` to start a section.")

	text, lang := s.Resolve("pl", "en")
	if lang != "en" || !strings.Contains(text, "to start a section") {
		t.Errorf("(%q, %q), want the prose kept whole and untagged", text, lang)
	}
}

func TestRepeatedLocaleAppends(t *testing.T) {
	s := Split("<!-- lang:en -->\nFirst\n\n<!-- lang:pl -->\nPolskie\n\n<!-- lang:en -->\nSecond")

	if text, _ := s.Resolve("en", "en"); text != "First\n\nSecond" {
		t.Errorf("text = %q, want both English parts", text)
	}
}

func TestEmptyDocument(t *testing.T) {
	if text, lang := Split("").Resolve("en", "en"); text != "" || lang != "en" {
		t.Errorf("(%q, %q), want empty text in the fallback language", text, lang)
	}
}
