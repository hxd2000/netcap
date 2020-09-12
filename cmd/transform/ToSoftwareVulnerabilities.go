package transform

import (
	"strings"

	"github.com/dreadl0ck/netcap/maltego"
	"github.com/dreadl0ck/netcap/types"
	"github.com/dreadl0ck/netcap/utils"
)

func toSoftwareVulnerabilities() {
	maltego.VulnerabilityTransform(
		nil,
		func(lt maltego.LocalTransform, trx *maltego.Transform, vuln *types.Vulnerability, min, max uint64, path string, mac string, ipaddr string) {
			val := vuln.ID
			product := vuln.Software.Product + " / " + vuln.Software.Version
			if len(product) > 0 {
				// for splitting descriptions from exploitdb
				//parts := strings.Split(vuln.Description, "-")
				//if len(parts) > 1 {
				//	val = parts[0] + "\n" + strings.Join(parts[1:], "-")
				//}
				val += "\n" + product
			}

			ent := trx.AddEntityWithPath("netcap.Vulnerability", val, path)
			ent.AddProperty("timestamp", "Timestamp", maltego.Strict, utils.UnixTimeToUTC(vuln.Timestamp))
			ent.AddProperty("id", "ID", maltego.Strict, vuln.ID)
			ent.AddProperty("notes", "Notes", maltego.Strict, vuln.Notes)
			ent.AddProperty("flows", "flows", maltego.Strict, strings.Join(vuln.Software.Flows, ","))
			ent.AddProperty("software", "Software", maltego.Strict, vuln.Software.Product+" "+vuln.Software.Version)

			ent.AddDisplayInformation(vuln.Description, "Description")
			ent.AddDisplayInformation(strings.Join(vuln.Software.Flows, "<br>"), "Flows")
		},
	)
}
