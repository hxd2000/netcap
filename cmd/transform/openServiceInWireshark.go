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
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func openServiceInWireshark() {
	var (
		lt              = maltego.ParseLocalArguments(os.Args)
		trx             = &maltego.Transform{}
		in              = strings.TrimSuffix(filepath.Dir(lt.Values["path"]), ".net")
		bpf             = makeServiceBPF(lt)
		outFile, exists = makeOutFilePath(in, bpf, lt, false, "")
		args            = []string{"-r", in, "-w", outFile, bpf}
	)

	if !exists {
		log.Println(tcpdumpPath, args)

		out, err := exec.Command(tcpdumpPath, args...).CombinedOutput()
		if err != nil {
			die(err.Error(), "open file failed:\n"+string(out))
		}

		log.Println(string(out))
	}

	log.Println(wiresharkPath, outFile)

	out, err := exec.Command(wiresharkPath, outFile).CombinedOutput()
	if err != nil {
		die(err.Error(), "open file failed:\n"+string(out))
	}

	log.Println(string(out))

	trx.AddUIMessage("completed!", maltego.UIMessageInform)
	fmt.Println(trx.ReturnOutput())
}

// creates a bpf to filter for traffic from and towards a specific network service
// e.g: host 127.0.0.1 and port 80
func makeServiceBPF(lt maltego.LocalTransform) string {
	var b strings.Builder

	b.WriteString("host ")
	b.WriteString(lt.Values["ip"])
	b.WriteString(" and port ")
	b.WriteString(lt.Values["port"])

	return b.String()
}