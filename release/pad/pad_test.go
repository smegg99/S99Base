// release/pad/pad_test.go
package pad

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

var update = flag.Bool("update", false, "rewrite testdata/example.pad.xml")

const golden = "testdata/example.pad.xml"

func example() (Product, Installer) {
	return Product{
		Publisher:        "Exämple & Söns",
		PublisherURL:     "https://example.com/contact/",
		Email:            "support@example.com",
		Name:             "Example Tuner",
		ReleaseStatus:    "Minor Update",
		InstallSupport:   "Install and Uninstall",
		OS:               []string{"Windows 10", "Windows 11"},
		Languages:        []string{"English", "Polish"},
		Category:         "Audio & Multimedia::Other",
		SpecificCategory: "Audio",
		Requirements:     "64-bit Windows 10 or 11",
		Keywords:         "tuner,reeds,beat rate,café",
		Desc45:           "Measures reeds & their beats",
		Desc80:           "Measures reeds & their beat rate from a microphone or a WAV file.",
		Desc250:          "Example Tuner measures up to three reeds & their beat rate. Tested in a café.",
		Desc450:          "Example Tuner measures up to three reeds & their beat rate from a microphone or a WAV file.",
		Desc2000:         "Example Tuner measures reeds & their beat rate.\n\nIt reads \"cents\" and the publisher's own units.",
		InfoURL:          "https://example.com/",
		ScreenshotURL:    "https://example.com/screenshots/tuner.png",
		IconURL:          "https://example.com/icon.png",
		PADURL:           "https://example.com/pad/example-windows.xml",
		Permissions:      "Free to redistribute unmodified.",
	}, Installer{
		Version:   "1.4.7",
		Published: time.Date(2026, 11, 18, 21, 29, 11, 0, time.UTC),
		Size:      15139581,
		URL:       "https://downloads.example.com/v1.4.7/example-1.4.7-windows-amd64-installer.exe",
	}
}

func mustMarshal(t *testing.T, p Product, in Installer) []byte {
	t.Helper()
	doc, err := Marshal(p, in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return doc
}

// parse reads leaf elements with the XML tokenizer alone, independently of the wire structs.
func parse(t *testing.T, doc []byte) (paths []string, text map[string]string) {
	t.Helper()
	text = map[string]string{}
	dec := xml.NewDecoder(bytes.NewReader(doc))
	var stack []string
	var data strings.Builder
	leaf := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return paths, text
		}
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			stack = append(stack, tok.Name.Local)
			data.Reset()
			leaf = true
		case xml.CharData:
			data.Write(tok)
		case xml.EndElement:
			if leaf {
				path := strings.Join(stack, "/")
				paths = append(paths, path)
				text[path] = data.String()
			}
			stack = stack[:len(stack)-1]
			leaf = false
		}
	}
}

func TestMarshalMatchesGolden(t *testing.T) {
	p, in := example()
	got := mustMarshal(t, p, in)
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("document differs from %s:\n%s", golden, got)
	}
}

func TestValuesSurviveAParse(t *testing.T) {
	p, in := example()
	doc := mustMarshal(t, p, in)
	if !bytes.HasPrefix(doc, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)) {
		t.Errorf("declaration = %q", doc[:40])
	}
	if !bytes.Contains(doc, []byte("Audio &amp; Multimedia::Other")) {
		t.Error("ampersand is not escaped as &amp;")
	}

	_, text := parse(t, doc)
	const root = "XML_DIZ_INFO/"
	for path, want := range map[string]string{
		"MASTER_PAD_VERSION_INFO/MASTER_PAD_VERSION":         "3.11",
		"Company_Info/Company_Name":                          "Exämple & Söns",
		"Company_Info/Company_WebSite_URL":                   p.PublisherURL,
		"Company_Info/Contact_Info/Author_Email":             p.Email,
		"Company_Info/Contact_Info/Contact_Email":            p.Email,
		"Company_Info/Support_Info/Sales_Email":              p.Email,
		"Company_Info/Support_Info/Support_Email":            p.Email,
		"Company_Info/Support_Info/General_Email":            p.Email,
		"Company_Info/Contact_Info/Author_First_Name":        "",
		"Program_Info/Program_Version":                       "1.4.7",
		"Program_Info/Program_Release_Month":                 "11",
		"Program_Info/Program_Release_Day":                   "18",
		"Program_Info/Program_Release_Year":                  "2026",
		"Program_Info/Program_Type":                          "Freeware",
		"Program_Info/Program_OS_Support":                    "Windows 10,Windows 11",
		"Program_Info/Program_Language":                      "English,Polish",
		"Program_Info/Program_Category_Class":                "Audio & Multimedia::Other",
		"Program_Info/File_Info/File_Size_Bytes":             "15139581",
		"Program_Info/File_Info/File_Size_K":                 "14784",
		"Program_Info/File_Info/File_Size_MB":                "14.44",
		"Program_Info/Expire_Info/Has_Expire_Info":           "N",
		"Program_Descriptions/English/Keywords":              "tuner,reeds,beat rate,café",
		"Program_Descriptions/English/Char_Desc_2000":        p.Desc2000,
		"Web_Info/Application_URLs/Application_Info_URL":     p.InfoURL,
		"Web_Info/Application_URLs/Application_XML_File_URL": p.PADURL,
		"Web_Info/Download_URLs/Primary_Download_URL":        in.URL,
		"Permissions/Distribution_Permissions":               p.Permissions,
	} {
		if got := text[root+path]; got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
}

func TestNoEmailLeavesMailboxesEmpty(t *testing.T) {
	p, in := example()
	p.Email = ""
	_, text := parse(t, mustMarshal(t, p, in))
	for path, value := range text {
		if strings.HasSuffix(path, "_Email") && value != "" {
			t.Errorf("%s = %q, want empty", path, value)
		}
	}
}

func TestReleaseDateIsUTC(t *testing.T) {
	for _, tc := range []struct {
		at             time.Time
		month, day, yr string
	}{
		{time.Date(2026, 11, 18, 23, 30, 0, 0, time.FixedZone("-02", -2*3600)), "11", "19", "2026"},
		{time.Date(2026, 11, 19, 0, 30, 0, 0, time.FixedZone("+02", 2*3600)), "11", "18", "2026"},
		{time.Date(2026, 12, 31, 23, 0, 0, 0, time.FixedZone("-05", -5*3600)), "01", "01", "2027"},
	} {
		p, in := example()
		in.Published = tc.at
		_, text := parse(t, mustMarshal(t, p, in))
		got := [3]string{
			text["XML_DIZ_INFO/Program_Info/Program_Release_Month"],
			text["XML_DIZ_INFO/Program_Info/Program_Release_Day"],
			text["XML_DIZ_INFO/Program_Info/Program_Release_Year"],
		}
		if want := [3]string{tc.month, tc.day, tc.yr}; got != want {
			t.Errorf("%s: month/day/year = %v, want %v", tc.at, got, want)
		}
	}
}

func TestFileSizes(t *testing.T) {
	for _, tc := range []struct {
		n     int64
		k, mb string
	}{
		{100, "0", "0.00"},
		{1023, "0", "0.00"},
		{1024, "1", "0.00"},
		{5242, "5", "0.00"},
		{5243, "5", "0.01"},
		{1048576, "1024", "1.00"},
		// PADGen wrote winLAME's 4,337,152 bytes as 4235 K and 4.14 MB.
		{4337152, "4235", "4.14"},
		{15139581, "14784", "14.44"},
		{104857594757, "102399994", "99999.99"},
	} {
		p, in := example()
		in.Size = tc.n
		_, text := parse(t, mustMarshal(t, p, in))
		got := [2]string{
			text["XML_DIZ_INFO/Program_Info/File_Info/File_Size_K"],
			text["XML_DIZ_INFO/Program_Info/File_Info/File_Size_MB"],
		}
		if want := [2]string{tc.k, tc.mb}; got != want {
			t.Errorf("%d bytes: K/MB = %v, want %v", tc.n, got, want)
		}
	}

	for _, n := range []int64{-1, 0, 99, 104857594758, 9_999_999_999_999_999, 1 << 62} {
		p, in := example()
		in.Size = n
		if _, err := Marshal(p, in); fieldOf(err) != "Installer.Size" {
			t.Errorf("%d bytes: err = %v, want an Installer.Size error", n, err)
		}
	}
}

func fieldOf(err error) string {
	var fe *FieldError
	if errors.As(err, &fe) {
		return fe.Field
	}
	return ""
}
