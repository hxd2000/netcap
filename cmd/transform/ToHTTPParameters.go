package transform

import (
	"github.com/dreadl0ck/netcap/maltego"
	"github.com/dreadl0ck/netcap/types"
)

func toHTTPParameters() {
	maltego.HTTPTransform(
		nil,
		func(lt maltego.LocalTransform, trx *maltego.Transform, http *types.HTTP, min, max uint64, path string, ipaddr string) {
			if http.SrcIP != ipaddr {
				return
			}
			for key := range http.Parameters {
				ent := trx.AddEntityWithPath("netcap.HTTPParameter", key, path)
				ent.AddProperty("ipaddr", "IPAddress", maltego.Strict, ipaddr)

				ent.SetLinkLabel(http.Method)
			}
		},
		false,
	)
}
