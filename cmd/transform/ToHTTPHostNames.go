package transform

import (
	maltego "github.com/dreadl0ck/netcap/maltego"
	"github.com/dreadl0ck/netcap/types"
	"strconv"
)

func ToHTTPHostNames() {

	var hostStats = make(map[string]int)

	maltego.HTTPTransform(
		nil,
		func(lt maltego.LocalTransform, trx *maltego.MaltegoTransform, http *types.HTTP, min, max uint64, profilesFile string, ipaddr string) {
			if http.Host != "" {
				ent := trx.AddEntity("netcap.Website", http.Host)
				ent.SetType("netcap.Website")
				ent.SetValue(http.Host)

				// di := "<h3>Host</h3><p>Timestamp: " + http.Timestamp + "</p>"
				// ent.AddDisplayInformation(di, "Netcap Info")

				//ent.SetLinkLabel(strconv.FormatInt(dns..NumPackets, 10) + " pkts")
				ent.SetLinkColor("#000000")

				hostStats[http.Host]++
				ent.SetLinkLabel(strconv.Itoa(hostStats[http.Host]))
				//ent.SetLinkThickness(maltego.GetThickness(ip.NumPackets))

				ent.AddProperty("ipaddr", "IPAddress", "strict", ipaddr)
				ent.AddProperty("path", "Path", "strict", profilesFile)
			}
		},
		false,
	)
}
