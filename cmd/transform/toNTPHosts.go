/*
 * NETCAP - Traffic Analysis Framework
 * Copyright (c) 2017-2020 Philipp Mieden <dreadl0ck [at] protonmail [dot] ch>
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package transform

import (
	"fmt"

	"github.com/dreadl0ck/netcap/maltego"
	"github.com/dreadl0ck/netcap/types"
)

func toNTPHosts() {
	var (
		profiles = maltego.LoadIPProfiles()
		hosts    = make(map[string]struct{})
		pathName string
	)

	maltego.NTPTransform(
		nil,
		func(lt maltego.LocalTransform, trx *maltego.Transform, ntp *types.NTP, min, max uint64, path string, ipaddr string) {
			if pathName == "" {
				pathName = path
			}
			hosts[ntp.SrcIP] = struct{}{}
			hosts[ntp.DstIP] = struct{}{}
		},
		true,
	)

	trx := maltego.Transform{}
	for ip := range hosts {
		if p, ok := profiles[ip]; ok {
			addIPProfile(&trx, p, pathName, 0, 0)
		}
	}

	trx.AddUIMessage("completed!", maltego.UIMessageInform)
	fmt.Println(trx.ReturnOutput())
}
