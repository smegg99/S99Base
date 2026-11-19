// release/pad/pad.go

// Package pad writes one release as a self-hosted PAD 3.11 document (XML_DIZ_INFO).
//
// Departures from 3.11: URLs may be https, as PAD 4.0 allows; OS names are written
// as given, so modern Windows names fit; Program_Type is always Freeware; one
// optional mailbox fills every email field; names, address, cost and expiry stay empty.
package pad

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Product is the listing copy that stays the same from release to release.
type Product struct {
	Publisher    string // Company_Name, 2-40 characters
	PublisherURL string // Company_WebSite_URL; a public contact page fits
	Email        string // optional public mailbox written to every email field
	Name         string // Program_Name without a version, 1-40 characters

	ReleaseStatus    string   // Program_Release_Status, such as "Minor Update"
	InstallSupport   string   // Program_Install_Support, such as "Install and Uninstall"
	OS               []string // Program_OS_Support names
	Languages        []string // Program_Language names, such as "English"
	Category         string   // Program_Category_Class, "Class::Subclass"
	SpecificCategory string   // Program_Specific_Category, optional
	Requirements     string   // Program_System_Requirements, optional, up to 100 characters

	Keywords string // up to 250 characters
	Desc45   string // one line each, within the length in its name
	Desc80   string
	Desc250  string
	Desc450  string
	Desc2000 string // paragraphs separated by LF

	InfoURL       string // Application_Info_URL
	ScreenshotURL string // .png, .jpg or .gif
	IconURL       string // .png, .jpg or .gif
	PADURL        string // Application_XML_File_URL, where this document is served

	Permissions string // Distribution_Permissions, optional
}

// Installer is the one file a release is listed with.
type Installer struct {
	Version   string    // Program_Version, 1-15 of A-Z a-z 0-9 . _ -
	Published time.Time // written as its UTC date
	Size      int64     // exact byte count
	URL       string    // Primary_Download_URL of this version's file
}

// FieldError names the Product or Installer field that PAD 3.11 cannot carry.
type FieldError struct {
	Field  string // such as "Product.Desc45" or "Installer.Size"
	Reason string
}

func (e *FieldError) Error() string { return "pad: " + e.Field + " " + e.Reason }

// Marshal validates both values and returns the UTF-8 document; it never truncates.
func Marshal(product Product, installer Installer) ([]byte, error) {
	if err := check(product, installer); err != nil {
		return nil, err
	}
	body, err := xml.MarshalIndent(newDocument(product, installer), "", "\t")
	if err != nil {
		return nil, fmt.Errorf("pad: %w", err)
	}
	out := append([]byte(xml.Header), body...)
	return append(out, '\n'), nil
}

func newDocument(p Product, in Installer) *document {
	released := in.Published.UTC()
	bytes, k, mb := sizes(in.Size)
	return &document{
		Version: versionInfo{Version: "3.11", Editor: "github.com/smegg99/s99base/release/pad"},
		Company: companyInfo{
			Name:    p.Publisher,
			Website: p.PublisherURL,
			Contact: contactInfo{AuthorEmail: p.Email, ContactEmail: p.Email},
			Support: supportInfo{SalesEmail: p.Email, SupportEmail: p.Email, GeneralEmail: p.Email},
		},
		Program: programInfo{
			Name:             p.Name,
			Version:          in.Version,
			ReleaseMonth:     fmt.Sprintf("%02d", int(released.Month())),
			ReleaseDay:       fmt.Sprintf("%02d", released.Day()),
			ReleaseYear:      strconv.Itoa(released.Year()),
			Type:             "Freeware",
			ReleaseStatus:    p.ReleaseStatus,
			InstallSupport:   p.InstallSupport,
			OS:               strings.Join(p.OS, ","),
			Language:         strings.Join(p.Languages, ","),
			SpecificCategory: p.SpecificCategory,
			Category:         p.Category,
			Requirements:     p.Requirements,
			File:             fileInfo{Bytes: bytes, K: k, MB: mb},
			Expire:           expireInfo{Has: "N"},
		},
		Descriptions: descriptions{English: english{
			Keywords: p.Keywords,
			Desc45:   p.Desc45,
			Desc80:   p.Desc80,
			Desc250:  p.Desc250,
			Desc450:  p.Desc450,
			Desc2000: p.Desc2000,
		}},
		Web: webInfo{
			Application: applicationURLs{
				Info:       p.InfoURL,
				Screenshot: p.ScreenshotURL,
				Icon:       p.IconURL,
				PAD:        p.PADURL,
			},
			Download: downloadURLs{Primary: in.URL},
		},
		Permissions: permissions{Distribution: p.Permissions},
	}
}

const (
	kib = 1 << 10
	mib = 1 << 20
)

// sizes follows PADGen: kilobytes truncate, megabytes round half up to hundredths.
func sizes(n int64) (bytes, k, mb string) {
	hundredths := (n*100 + mib/2) / mib
	return strconv.FormatInt(n, 10), strconv.FormatInt(n/kib, 10), fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100)
}
