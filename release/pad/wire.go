// release/pad/wire.go
package pad

import "encoding/xml"

// Element order follows PADGen 3.1.1.50 output for PAD 3.11.
type document struct {
	XMLName      xml.Name     `xml:"XML_DIZ_INFO"`
	Version      versionInfo  `xml:"MASTER_PAD_VERSION_INFO"`
	Company      companyInfo  `xml:"Company_Info"`
	Program      programInfo  `xml:"Program_Info"`
	Descriptions descriptions `xml:"Program_Descriptions"`
	Web          webInfo      `xml:"Web_Info"`
	Permissions  permissions  `xml:"Permissions"`
}

// blank is an element the 3.11 layout carries and this writer leaves empty.
type blank struct{}

type versionInfo struct {
	Version string `xml:"MASTER_PAD_VERSION"`
	Editor  string `xml:"MASTER_PAD_EDITOR"`
	Info    blank  `xml:"MASTER_PAD_INFO"`
}

type companyInfo struct {
	Name     string      `xml:"Company_Name"`
	Address1 blank       `xml:"Address_1"`
	Address2 blank       `xml:"Address_2"`
	City     blank       `xml:"City_Town"`
	State    blank       `xml:"State_Province"`
	Zip      blank       `xml:"Zip_Postal_Code"`
	Country  blank       `xml:"Country"`
	Website  string      `xml:"Company_WebSite_URL"`
	Contact  contactInfo `xml:"Contact_Info"`
	Support  supportInfo `xml:"Support_Info"`
}

type contactInfo struct {
	AuthorFirstName  blank  `xml:"Author_First_Name"`
	AuthorLastName   blank  `xml:"Author_Last_Name"`
	AuthorEmail      string `xml:"Author_Email"`
	ContactFirstName blank  `xml:"Contact_First_Name"`
	ContactLastName  blank  `xml:"Contact_Last_Name"`
	ContactEmail     string `xml:"Contact_Email"`
}

type supportInfo struct {
	SalesEmail   string `xml:"Sales_Email"`
	SupportEmail string `xml:"Support_Email"`
	GeneralEmail string `xml:"General_Email"`
	SalesPhone   blank  `xml:"Sales_Phone"`
	SupportPhone blank  `xml:"Support_Phone"`
	GeneralPhone blank  `xml:"General_Phone"`
	FaxPhone     blank  `xml:"Fax_Phone"`
}

type programInfo struct {
	Name             string     `xml:"Program_Name"`
	Version          string     `xml:"Program_Version"`
	ReleaseMonth     string     `xml:"Program_Release_Month"`
	ReleaseDay       string     `xml:"Program_Release_Day"`
	ReleaseYear      string     `xml:"Program_Release_Year"`
	CostDollars      blank      `xml:"Program_Cost_Dollars"`
	CostOtherCode    blank      `xml:"Program_Cost_Other_Code"`
	CostOther        blank      `xml:"Program_Cost_Other"`
	Type             string     `xml:"Program_Type"`
	ReleaseStatus    string     `xml:"Program_Release_Status"`
	InstallSupport   string     `xml:"Program_Install_Support"`
	OS               string     `xml:"Program_OS_Support"`
	Language         string     `xml:"Program_Language"`
	ChangeInfo       blank      `xml:"Program_Change_Info"`
	SpecificCategory string     `xml:"Program_Specific_Category"`
	Category         string     `xml:"Program_Category_Class"`
	Requirements     string     `xml:"Program_System_Requirements"`
	File             fileInfo   `xml:"File_Info"`
	Expire           expireInfo `xml:"Expire_Info"`
}

type fileInfo struct {
	Bytes string `xml:"File_Size_Bytes"`
	K     string `xml:"File_Size_K"`
	MB    string `xml:"File_Size_MB"`
}

type expireInfo struct {
	Has     string `xml:"Has_Expire_Info"`
	Count   blank  `xml:"Expire_Count"`
	BasedOn blank  `xml:"Expire_Based_On"`
	Other   blank  `xml:"Expire_Other_Info"`
	Month   blank  `xml:"Expire_Month"`
	Day     blank  `xml:"Expire_Day"`
	Year    blank  `xml:"Expire_Year"`
}

type descriptions struct {
	English english `xml:"English"`
}

type english struct {
	Keywords string `xml:"Keywords"`
	Desc45   string `xml:"Char_Desc_45"`
	Desc80   string `xml:"Char_Desc_80"`
	Desc250  string `xml:"Char_Desc_250"`
	Desc450  string `xml:"Char_Desc_450"`
	Desc2000 string `xml:"Char_Desc_2000"`
}

type webInfo struct {
	Application applicationURLs `xml:"Application_URLs"`
	Download    downloadURLs    `xml:"Download_URLs"`
}

type applicationURLs struct {
	Info       string `xml:"Application_Info_URL"`
	Order      blank  `xml:"Application_Order_URL"`
	Screenshot string `xml:"Application_Screenshot_URL"`
	Icon       string `xml:"Application_Icon_URL"`
	PAD        string `xml:"Application_XML_File_URL"`
}

type downloadURLs struct {
	Primary     string `xml:"Primary_Download_URL"`
	Secondary   blank  `xml:"Secondary_Download_URL"`
	Additional1 blank  `xml:"Additional_Download_URL_1"`
	Additional2 blank  `xml:"Additional_Download_URL_2"`
}

type permissions struct {
	Distribution string `xml:"Distribution_Permissions"`
	EULA         blank  `xml:"EULA"`
}
