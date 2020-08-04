package maltego

import (
	"encoding/xml"
	"fmt"
	"github.com/dreadl0ck/netcap/decoder"
	"github.com/mgutz/ansi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// additional entities that are not actual NETCAP audit records
var entities = []EntityCoreInfo{
	{"CaptureProcess", "remove_red_eye", "An operating system NETCAP process that captures traffic from a network interface", "", nil},
	{"ContentType", "category", "A MIME type describes different multi-media formats", "", nil},
	{"Credentials", "security", "Credentials for accessing services that require user authentication", "netcap.IPAddr", nil},
	{"Email", "mail_outline", "An email message", "maltego.Email", nil},
	{"Interface", "router", "A network interface", "", []PropertyField{newRequiredStringField("name", "Name of the network interface")}},
	{"PCAP", "sd_storage", "A packet capture dump file", "", []PropertyField{newRequiredStringField("path", "Absolute path to the PCAP file")}},
	{"Device", "devices", "A device seen on the network", "", nil},
	{"FileType", "insert_chart", "The type of file based on its contents", "", nil},
	{"IPAddr", "router", "An internet protocol (IP) network address", "maltego.IPv4Address", nil},

	// TODO: icons
	{"InternalSourceIP", "cloud_upload", "An internal source address", "netcap.IPAddr", nil},
	{"ExternalSourceIP", "cloud_upload", "An external source address", "netcap.IPAddr", nil},
	{"InternalDestinationIP", "cloud_download", "An internal destination address", "netcap.IPAddr", nil},
	{"ExternalDestinationIP", "cloud_download", "An external destination address", "netcap.IPAddr", nil},

	{"DHCPClient", "cast_connected", "A DHCP client", "", nil},
	{"DHCPResult", "fingerprint", "A DHCP fingerprint result", "", nil},
	{"DestinationPort", "local_convenience_store", "A TCP / UDP destination port", "", nil},
	{"SourcePort", "local_convenience_store", "A TCP / UDP source port", "", nil},
	{"HTTPCookie", "settings_input_svideo", "A HTTP session cookie", "", nil},
	{"HTTPParameter", "live_help", "", "A HTTP request parameter name", nil},
	{"HTTPParameterValue", "settings_ethernet", "A HTTP request parameter value", "", nil},
	{"HTTPStatusCode", "highlight", "A HTTP server response code", "", nil},
	{"ServerName", "scanner", "A network server software name", "", nil},
	{"SSHClient", "call_made", "A Secure Shell Protocol Client", "", nil},
	{"SSHServer", "call_received", "A Secure Shell Protocol Server", "", nil},
	{"TCPService", "device_hub", "A TCP network service", "", nil},
	{"UDPService", "developer_board", "A UDP network service", "", nil},
	{"UserAgent", "supervisor_account", "A HTTP User Agent", "", nil},
	{"Website", "web", "A HTTP Website", "maltego.Website", nil},
	{"DNSName", "dns", "A DNS Name", "maltego.DNSName", nil},
	{"File", "insert_drive_file", "A file", "maltego.File", nil},
	{"Domain", "domain", "A domain", "maltego.Domain", nil},
	{"Location", "location_on", "A location", "maltego.Location", nil},
	{"URL", "open_in_browser", "A Uniform Resource Identifier", "maltego.URL", nil},
	{"HTTPCookieValue", "info", "A HTTP cookie value", "", nil},
	{"ExifEntry", "info", "An Exif entry for an image file", "", nil},
	{"MD5Hash", "info", "An MD5 hash entry for an extracted file", "maltego.Hash", nil},
	{"PhoneNumber", "contact_phone", "A phone number", "maltego.PhoneNumber", nil},
}

// generate all entities and pack as archive
func TestGenerateAllEntities(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	genEntityArchive()

	// generate additional entities
	for _, e := range entities {
		genEntity("entities", e.Name, e.Icon, e.Description, e.Parent, e.Fields...)
	}

	// generate entities for audit records
	// *AuditRecords entity and an entity for the actual audit record instance
	decoder.ApplyActionToCustomDecoders(func(d decoder.CustomDecoderAPI) {
		genEntity("entities", d.GetName()+"AuditRecords", "insert_drive_file", "An archive of "+d.GetName()+" audit records", "", newStringField("path"))
		genEntity("entities", d.GetName(), d.GetName(), d.GetDescription(), "")
	})

	decoder.ApplyActionToGoPacketDecoders(func(e *decoder.GoPacketDecoder) {
		name := strings.ReplaceAll(e.Layer.String(), "/", "")
		genEntity("entities", name+"AuditRecords", "insert_drive_file", "An archive of "+e.Layer.String()+" audit records", "", newStringField("path"))
		genEntity("entities", name, name, e.Description, "")
	})

	packEntityArchive()

	copyFile("entities.mtz", filepath.Join(os.Getenv("HOME"), "entities.mtz"))
}

func TestGenerateAndPackVulnerabilityEntity(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	genEntityArchive()
	genEntity("entities", "Vulnerability", "Vulnerability", "A software vulnerability", "")
	packEntityArchive()
}

func TestGenerateAndPackCaptureProcessEntity(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	genEntityArchive()
	genEntity("entities", "CaptureProcess", "remove_red_eye", "A capture process", "")
	packEntityArchive()
}

func TestGenerateAndPackPCAPEntity(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	genEntityArchive()
	genEntity("entities", "PCAP", "sd_storage", "Packet capture file", "", newStringField("path"))
	packEntityArchive()
}

func TestGeneratePCAPXMLEntity(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	expected := `<MaltegoEntity id="netcap.PCAP" displayName="PCAP" displayNamePlural="PCAPs" description="Packet capture file" category="Netcap" smallIconResource="General/SharkAttack" largeIconResource="General/SharkAttack" allowedRoot="true" conversionOrder="2147483647" visible="true">
 <Properties value="properties.filename" displayValue="properties.filename">
  <Groups/>
  <Fields>
   <Field name="properties.filename" type="string" nullable="true" hidden="false" readonly="false" description="" displayName="Filename">
    <SampleValue>-</SampleValue>
   </Field>
   <Field name="path" type="string" nullable="true" hidden="false" readonly="false" description="" displayName="Path">
    <SampleValue></SampleValue>
   </Field>
  </Fields>
 </Properties>
</MaltegoEntity>`

	e := newEntity("PCAP", "General/SharkAttack", "Packet capture file", "", newStringField("path"))

	data, err := xml.MarshalIndent(e, "", " ")
	if err != nil {
		t.Fatal(err)
	}

	compareGeneratedXML(data, expected, t)
}

func compareGeneratedXML(data []byte, expected string, t *testing.T) {
	fmt.Println("-------------------RESULT--------------------------")
	fmt.Println(string(data))
	fmt.Println("------------------------------------------------")

	if string(data) != expected {

		fmt.Println("-------------------EXPECTED--------------------------")
		fmt.Println(expected)
		fmt.Println("------------------------------------------------")

		resultArr := strings.Split(string(data), "\n")
		expectedArr := strings.Split(string(expected), "\n")

		fmt.Println(ansi.Red, "len(resultArr)", len(resultArr), ansi.Blue, "len(expectedArr)", len(expectedArr), ansi.Reset)

		for i, line := range expectedArr {
			if len(resultArr) <= i {
				break
			}
			if line != resultArr[i] {
				fmt.Println(ansi.Red, resultArr[i], ansi.Reset)
				fmt.Println(ansi.Blue, expectedArr[i], ansi.Reset)
			} else {
				fmt.Println(resultArr[i])
			}
		}

		t.Fatal("unexpected output")
	}
}

func TestGenerateDHCPClientXMLEntity(t *testing.T) {

	if !generateMaltegoConfig {
		return
	}

	expected := `<MaltegoEntity id="netcap.DHCPClient" displayName="DHCPClient" displayNamePlural="DHCPClients" description="A DHCP client" category="Netcap" smallIconResource="Technology/WAN" largeIconResource="Technology/WAN" allowedRoot="true" conversionOrder="2147483647" visible="true">
 <Properties value="properties.dhcpclient" displayValue="properties.dhcpclient">
  <Fields>
   <Field name="properties.dhcpclient" type="string" nullable="true" hidden="false" readonly="false" description="" displayName="DHCPClient">
    <SampleValue>-</SampleValue>
   </Field>
  </Fields>
 </Properties>
</MaltegoEntity>`
	e := XMLEntity{
		ID:                "netcap.DHCPClient",
		DisplayName:       "DHCPClient",
		DisplayNamePlural: "DHCPClients",
		Description:       "A DHCP client",
		Category:          "Netcap",
		SmallIconResource: "Technology/WAN",
		LargeIconResource: "Technology/WAN",
		AllowedRoot:       true,
		ConversionOrder:   "2147483647",
		Visible:           true,
		Properties: EntityProperties{
			XMLName:      xml.Name{},
			Text:         "",
			Value:        "properties.dhcpclient",
			DisplayValue: "properties.dhcpclient",
			Fields: Fields{
				Text: "",
				Items: []PropertyField{
					{
						Name:        "properties.dhcpclient",
						Type:        "string",
						Nullable:    true,
						Hidden:      false,
						Readonly:    false,
						Description: "",
						DisplayName: "DHCPClient",
						SampleValue: "-",
					},
				},
			},
		},
	}

	data, err := xml.MarshalIndent(e, "", " ")
	if err != nil {
		t.Fatal(err)
	}

	compareGeneratedXML(data, expected, t)
}