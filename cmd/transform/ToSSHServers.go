package transform

import (
	"github.com/dreadl0ck/netcap/maltego"
	"github.com/dreadl0ck/netcap/types"
	"github.com/dreadl0ck/netcap/utils"
)

func toSSHServers() {
	maltego.SSHTransform(
		nil,
		func(lt maltego.LocalTransform, trx *maltego.Transform, ssh *types.SSH, min, max uint64, path string, mac string, ipaddr string) {
			if !ssh.IsClient {
				val := ssh.HASSH + "\n" + ssh.Flow
				if len(ssh.Ident) > 0 {
					val += "\n" + ssh.Ident
				}

				ent := trx.AddEntityWithPath("netcap.SSHServer", val, path)
				ent.AddProperty("timestamp", "Timestamp", maltego.Strict, utils.UnixTimeToUTC(ssh.Timestamp))
				ent.AddProperty("ident", "Ident", maltego.Strict, ssh.Ident)
				ent.AddProperty("flow", "Flow", maltego.Strict, ssh.Flow)
				ent.AddProperty("algorithms", "Algorithms", maltego.Strict, ssh.Algorithms)
			}
		},
	)
}
