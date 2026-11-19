// release/pad/reference_test.go
package pad

import (
	"slices"
	"testing"
)

// PADGen 3.1.1.50 output, leaf elements in order:
// https://github.com/vividos/winLAME/blob/e1920c2499ee857d97164ec0b6b3177744dec0fe/docs/winLAME.pad.xml
var padgenLeaves = []string{
	"XML_DIZ_INFO/MASTER_PAD_VERSION_INFO/MASTER_PAD_VERSION",
	"XML_DIZ_INFO/MASTER_PAD_VERSION_INFO/MASTER_PAD_EDITOR",
	"XML_DIZ_INFO/MASTER_PAD_VERSION_INFO/MASTER_PAD_INFO",
	"XML_DIZ_INFO/Company_Info/Company_Name",
	"XML_DIZ_INFO/Company_Info/Address_1",
	"XML_DIZ_INFO/Company_Info/Address_2",
	"XML_DIZ_INFO/Company_Info/City_Town",
	"XML_DIZ_INFO/Company_Info/State_Province",
	"XML_DIZ_INFO/Company_Info/Zip_Postal_Code",
	"XML_DIZ_INFO/Company_Info/Country",
	"XML_DIZ_INFO/Company_Info/Company_WebSite_URL",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Author_First_Name",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Author_Last_Name",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Author_Email",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Contact_First_Name",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Contact_Last_Name",
	"XML_DIZ_INFO/Company_Info/Contact_Info/Contact_Email",
	"XML_DIZ_INFO/Company_Info/Support_Info/Sales_Email",
	"XML_DIZ_INFO/Company_Info/Support_Info/Support_Email",
	"XML_DIZ_INFO/Company_Info/Support_Info/General_Email",
	"XML_DIZ_INFO/Company_Info/Support_Info/Sales_Phone",
	"XML_DIZ_INFO/Company_Info/Support_Info/Support_Phone",
	"XML_DIZ_INFO/Company_Info/Support_Info/General_Phone",
	"XML_DIZ_INFO/Company_Info/Support_Info/Fax_Phone",
	"XML_DIZ_INFO/Program_Info/Program_Name",
	"XML_DIZ_INFO/Program_Info/Program_Version",
	"XML_DIZ_INFO/Program_Info/Program_Release_Month",
	"XML_DIZ_INFO/Program_Info/Program_Release_Day",
	"XML_DIZ_INFO/Program_Info/Program_Release_Year",
	"XML_DIZ_INFO/Program_Info/Program_Cost_Dollars",
	"XML_DIZ_INFO/Program_Info/Program_Cost_Other_Code",
	"XML_DIZ_INFO/Program_Info/Program_Cost_Other",
	"XML_DIZ_INFO/Program_Info/Program_Type",
	"XML_DIZ_INFO/Program_Info/Program_Release_Status",
	"XML_DIZ_INFO/Program_Info/Program_Install_Support",
	"XML_DIZ_INFO/Program_Info/Program_OS_Support",
	"XML_DIZ_INFO/Program_Info/Program_Language",
	"XML_DIZ_INFO/Program_Info/Program_Change_Info",
	"XML_DIZ_INFO/Program_Info/Program_Specific_Category",
	"XML_DIZ_INFO/Program_Info/Program_Category_Class",
	"XML_DIZ_INFO/Program_Info/Program_System_Requirements",
	"XML_DIZ_INFO/Program_Info/File_Info/File_Size_Bytes",
	"XML_DIZ_INFO/Program_Info/File_Info/File_Size_K",
	"XML_DIZ_INFO/Program_Info/File_Info/File_Size_MB",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Has_Expire_Info",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Count",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Based_On",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Other_Info",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Month",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Day",
	"XML_DIZ_INFO/Program_Info/Expire_Info/Expire_Year",
	"XML_DIZ_INFO/Program_Descriptions/English/Keywords",
	"XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_45",
	"XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_80",
	"XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_250",
	"XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_450",
	"XML_DIZ_INFO/Program_Descriptions/English/Char_Desc_2000",
	"XML_DIZ_INFO/Web_Info/Application_URLs/Application_Info_URL",
	"XML_DIZ_INFO/Web_Info/Application_URLs/Application_Order_URL",
	"XML_DIZ_INFO/Web_Info/Application_URLs/Application_Screenshot_URL",
	"XML_DIZ_INFO/Web_Info/Application_URLs/Application_Icon_URL",
	"XML_DIZ_INFO/Web_Info/Application_URLs/Application_XML_File_URL",
	"XML_DIZ_INFO/Web_Info/Download_URLs/Primary_Download_URL",
	"XML_DIZ_INFO/Web_Info/Download_URLs/Secondary_Download_URL",
	"XML_DIZ_INFO/Web_Info/Download_URLs/Additional_Download_URL_1",
	"XML_DIZ_INFO/Web_Info/Download_URLs/Additional_Download_URL_2",
	"XML_DIZ_INFO/Permissions/Distribution_Permissions",
	"XML_DIZ_INFO/Permissions/EULA",
}

func TestLayoutMatchesPADGen(t *testing.T) {
	p, in := example()
	paths, _ := parse(t, mustMarshal(t, p, in))
	if !slices.Equal(paths, padgenLeaves) {
		t.Errorf("leaf elements = %q\nwant %q", paths, padgenLeaves)
	}
}
