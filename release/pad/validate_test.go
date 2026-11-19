// release/pad/validate_test.go
package pad

import (
	"strings"
	"testing"
	"time"
)

type edit struct {
	name    string
	product func(*Product)
	install func(*Installer)
	field   string // empty when the edit is valid
}

func run(t *testing.T, cases []edit) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, in := example()
			if tc.product != nil {
				tc.product(&p)
			}
			if tc.install != nil {
				tc.install(&in)
			}
			doc, err := Marshal(p, in)
			if got := fieldOf(err); got != tc.field {
				t.Fatalf("field = %q (err %v), want %q", got, err, tc.field)
			}
			if tc.field != "" && doc != nil {
				t.Fatal("returned a document with an error")
			}
		})
	}
}

func TestTextLimits(t *testing.T) {
	repeat := strings.Repeat
	paragraphs := repeat("é", 999) + "\n\n" + repeat("ł", 999)
	run(t, []edit{
		{name: "45 accented characters", product: func(p *Product) { p.Desc45 = repeat("é", 45) }},
		{name: "46 accented characters", product: func(p *Product) { p.Desc45 = repeat("é", 46) }, field: "Product.Desc45"},
		{name: "80 characters", product: func(p *Product) { p.Desc80 = repeat("a", 80) }},
		{name: "81 characters", product: func(p *Product) { p.Desc80 = repeat("a", 81) }, field: "Product.Desc80"},
		{name: "250 characters", product: func(p *Product) { p.Desc250 = repeat("a", 250) }},
		{name: "251 characters", product: func(p *Product) { p.Desc250 = repeat("a", 251) }, field: "Product.Desc250"},
		{name: "450 characters", product: func(p *Product) { p.Desc450 = repeat("a", 450) }},
		{name: "451 characters", product: func(p *Product) { p.Desc450 = repeat("a", 451) }, field: "Product.Desc450"},
		{name: "2000 characters in paragraphs", product: func(p *Product) { p.Desc2000 = paragraphs }},
		{name: "2001 characters", product: func(p *Product) { p.Desc2000 = paragraphs + "a" }, field: "Product.Desc2000"},
		{name: "keywords at 250", product: func(p *Product) { p.Keywords = repeat("k", 250) }},
		{name: "keywords at 251", product: func(p *Product) { p.Keywords = repeat("k", 251) }, field: "Product.Keywords"},
		{name: "publisher at 40", product: func(p *Product) { p.Publisher = repeat("ż", 40) }},
		{name: "publisher at 41", product: func(p *Product) { p.Publisher = repeat("ż", 41) }, field: "Product.Publisher"},
		{name: "publisher at 1", product: func(p *Product) { p.Publisher = "A" }, field: "Product.Publisher"},
		{name: "name at 41", product: func(p *Product) { p.Name = repeat("n", 41) }, field: "Product.Name"},
		{name: "blank name", product: func(p *Product) { p.Name = "  " }, field: "Product.Name"},
		{name: "requirements at 101", product: func(p *Product) { p.Requirements = repeat("r", 101) }, field: "Product.Requirements"},
		{name: "permissions at 2001", product: func(p *Product) { p.Permissions = repeat("p", 2001) }, field: "Product.Permissions"},
		{name: "line break in a one-line field", product: func(p *Product) { p.Desc450 = "one\ntwo" }, field: "Product.Desc450"},
		{name: "CRLF in a paragraph field", product: func(p *Product) { p.Desc2000 = "one\r\ntwo" }, field: "Product.Desc2000"},
		{name: "two blank lines", product: func(p *Product) { p.Desc2000 = "one\n\n\ntwo" }, field: "Product.Desc2000"},
		{name: "markup", product: func(p *Product) { p.Desc80 = "<b>bold</b>" }, field: "Product.Desc80"},
		{name: "tab", product: func(p *Product) { p.Keywords = "a\tb" }, field: "Product.Keywords"},
		{name: "control character", product: func(p *Product) { p.Desc250 = "a\x00b" }, field: "Product.Desc250"},
		{name: "noncharacter", product: func(p *Product) { p.Desc250 = "a￾b" }, field: "Product.Desc250"},
		{name: "invalid UTF-8", product: func(p *Product) { p.Name = "Tuner\xff" }, field: "Product.Name"},
		{name: "ampersands and quotes", product: func(p *Product) { p.Desc80 = `Reeds & beats, "cents" & 'units'` }},
	})
}

func TestTextIsNotTruncated(t *testing.T) {
	p, in := example()
	p.Desc45 = strings.Repeat("é", 44) + "ł"
	_, text := parse(t, mustMarshal(t, p, in))
	if got := text["XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_45"]; got != p.Desc45 {
		t.Errorf("Char_Desc_45 = %q, want %q", got, p.Desc45)
	}
}

func TestRequiredFields(t *testing.T) {
	run(t, []edit{
		{name: "publisher", product: func(p *Product) { p.Publisher = "" }, field: "Product.Publisher"},
		{name: "publisher URL", product: func(p *Product) { p.PublisherURL = "" }, field: "Product.PublisherURL"},
		{name: "name", product: func(p *Product) { p.Name = "" }, field: "Product.Name"},
		{name: "release status", product: func(p *Product) { p.ReleaseStatus = "" }, field: "Product.ReleaseStatus"},
		{name: "install support", product: func(p *Product) { p.InstallSupport = "" }, field: "Product.InstallSupport"},
		{name: "OS", product: func(p *Product) { p.OS = nil }, field: "Product.OS"},
		{name: "languages", product: func(p *Product) { p.Languages = nil }, field: "Product.Languages"},
		{name: "category", product: func(p *Product) { p.Category = "" }, field: "Product.Category"},
		{name: "desc45", product: func(p *Product) { p.Desc45 = "" }, field: "Product.Desc45"},
		{name: "desc2000", product: func(p *Product) { p.Desc2000 = "" }, field: "Product.Desc2000"},
		{name: "info URL", product: func(p *Product) { p.InfoURL = "" }, field: "Product.InfoURL"},
		{name: "screenshot URL", product: func(p *Product) { p.ScreenshotURL = "" }, field: "Product.ScreenshotURL"},
		{name: "icon URL", product: func(p *Product) { p.IconURL = "" }, field: "Product.IconURL"},
		{name: "PAD URL", product: func(p *Product) { p.PADURL = "" }, field: "Product.PADURL"},
		{name: "version", install: func(in *Installer) { in.Version = "" }, field: "Installer.Version"},
		{name: "zero date", install: func(in *Installer) { in.Published = time.Time{} }, field: "Installer.Published"},
		{name: "download URL", install: func(in *Installer) { in.URL = "" }, field: "Installer.URL"},
		{name: "optional fields empty", product: func(p *Product) {
			p.SpecificCategory, p.Requirements, p.Keywords, p.Permissions = "", "", "", ""
		}},
	})
}

func TestNamesAndVersions(t *testing.T) {
	run(t, []edit{
		{name: "unknown status", product: func(p *Product) { p.ReleaseStatus = "Patch" }, field: "Product.ReleaseStatus"},
		{name: "unknown install support", product: func(p *Product) { p.InstallSupport = "Portable" }, field: "Product.InstallSupport"},
		{name: "unknown specific category", product: func(p *Product) { p.SpecificCategory = "Music" }, field: "Product.SpecificCategory"},
		{name: "category without subclass", product: func(p *Product) { p.Category = "Audio" }, field: "Product.Category"},
		{name: "empty OS name", product: func(p *Product) { p.OS = []string{"Windows 10", ""} }, field: "Product.OS[1]"},
		{name: "comma in a language", product: func(p *Product) { p.Languages = []string{"English,Polish"} }, field: "Product.Languages[0]"},
		{name: "v prefix", install: func(in *Installer) { in.Version = "v1.4.7" }},
		{name: "prerelease suffix", install: func(in *Installer) { in.Version = "1.4.7-rc.1" }},
		{name: "15 characters", install: func(in *Installer) { in.Version = "1.4.7-rc.123456" }},
		{name: "16 characters", install: func(in *Installer) { in.Version = "1.4.7-rc.1234567" }, field: "Installer.Version"},
		{name: "space", install: func(in *Installer) { in.Version = "1.4.7 beta" }, field: "Installer.Version"},
		{name: "build metadata", install: func(in *Installer) { in.Version = "1.4.7+abc" }, field: "Installer.Version"},
		{name: "year 1899", install: func(in *Installer) { in.Published = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC) }, field: "Installer.Published"},
		{name: "year 2200", install: func(in *Installer) { in.Published = time.Date(2200, 1, 1, 0, 0, 0, 0, time.UTC) }, field: "Installer.Published"},
	})
}

func TestURLs(t *testing.T) {
	info := func(raw string) func(*Product) { return func(p *Product) { p.InfoURL = raw } }
	host120 := "https://" + strings.Repeat("a", 116) + ".com"
	run(t, []edit{
		{name: "https", product: info("https://example.com/")},
		{name: "http", product: info("http://example.com/")},
		{name: "public address", product: info("https://93.184.215.14/")},
		{name: "120 characters", product: info(host120)},
		{name: "121 characters", product: info(host120 + "m"), field: "Product.InfoURL"},
		{name: "relative", product: info("/download"), field: "Product.InfoURL"},
		{name: "no scheme", product: info("example.com/"), field: "Product.InfoURL"},
		{name: "ftp", product: info("ftp://example.com/"), field: "Product.InfoURL"},
		{name: "mailto", product: info("mailto:someone@example.com"), field: "Product.InfoURL"},
		{name: "opaque", product: info("https:example.com"), field: "Product.InfoURL"},
		{name: "no host", product: info("https:///path"), field: "Product.InfoURL"},
		{name: "upper-case scheme", product: info("HTTPS://example.com/"), field: "Product.InfoURL"},
		{name: "space", product: info("https://example.com/a b"), field: "Product.InfoURL"},
		{name: "credentials", product: info("https://user:secret@example.com/"), field: "Product.InfoURL"},
		{name: "localhost", product: info("https://localhost/"), field: "Product.InfoURL"},
		{name: "localhost subdomain", product: info("https://app.localhost/"), field: "Product.InfoURL"},
		{name: "single label", product: info("https://intranet/"), field: "Product.InfoURL"},
		{name: "loopback", product: info("https://127.0.0.1/"), field: "Product.InfoURL"},
		{name: "private", product: info("https://192.168.1.10/"), field: "Product.InfoURL"},
		{name: "link-local", product: info("https://169.254.1.1/"), field: "Product.InfoURL"},
		{name: "IPv6 loopback", product: info("https://[::1]/"), field: "Product.InfoURL"},
		{name: "mapped loopback", product: info("https://[::ffff:127.0.0.1]/"), field: "Product.InfoURL"},
		{name: "screenshot webp", product: func(p *Product) { p.ScreenshotURL = "https://example.com/s.webp" }, field: "Product.ScreenshotURL"},
		{name: "icon upper-case extension", product: func(p *Product) { p.IconURL = "https://example.com/i.PNG" }, field: "Product.IconURL"},
		{name: "PAD URL as json", product: func(p *Product) { p.PADURL = "https://example.com/pad.json" }, field: "Product.PADURL"},
		{name: "download page", install: func(in *Installer) { in.URL = "https://example.com/download" }, field: "Installer.URL"},
		{name: "msi", install: func(in *Installer) { in.URL = "https://example.com/setup.msi" }},
		{name: "120 characters before .exe", install: func(in *Installer) { in.URL = host120 + ".exe" }},
		{name: "121 characters before .exe", install: func(in *Installer) { in.URL = host120 + "m.exe" }, field: "Installer.URL"},
	})
}
