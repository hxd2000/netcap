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

package encoder

import (
	"github.com/dreadl0ck/ja3"
	"strconv"

	"github.com/dreadl0ck/netcap/types"
	"github.com/dreadl0ck/netcap/utils"

	"github.com/dreadl0ck/gopacket"
	"github.com/dreadl0ck/tlsx"
	"github.com/golang/protobuf/proto"
)

var tlsServerHelloEncoder = CreateCustomEncoder(types.Type_NC_TLSServerHello, "TLSServerHello", nil, func(p gopacket.Packet) proto.Message {

	hello := tlsx.GetServerHello(p)
	if hello != nil {

		var (
			extensions = make([]int32, len(hello.Extensions))
		)
		for i, v := range hello.Extensions {
			extensions[i] = int32(v)
		}

		var (
			srcPort, _ = strconv.Atoi(p.TransportLayer().TransportFlow().Src().String())
			dstPort, _ = strconv.Atoi(p.TransportLayer().TransportFlow().Src().String())
		)

		return &types.TLSServerHello{
			Timestamp:                    utils.TimeToString(p.Metadata().Timestamp),
			Version:                      int32(hello.Vers),
			Random:                       hello.Random,
			SessionID:                    hello.SessionID,
			CipherSuite:                  int32(hello.CipherSuite),
			CompressionMethod:            int32(hello.CompressionMethod),
			NextProtoNeg:                 hello.NextProtoNeg,
			NextProtos:                   hello.NextProtos,
			OCSPStapling:                 hello.OCSPStapling,
			TicketSupported:              hello.TicketSupported,
			SecureRenegotiationSupported: hello.SecureRenegotiationSupported,
			SecureRenegotiation:          hello.SecureRenegotiation,
			AlpnProtocol:                 hello.AlpnProtocol,
			Ems:                          hello.Ems,
			Scts:                         hello.Scts,
			SupportedVersion:             int32(hello.SupportedVersion),
			SelectedIdentityPresent:      hello.SelectedIdentityPresent,
			SelectedIdentity:             int32(hello.SelectedIdentity),
			Cookie:                       hello.Cookie,
			SelectedGroup:                int32(hello.SelectedGroup),
			Ja3S:                         ja3.DigestHexJa3s(&hello.ServerHelloBasic),
			SrcIP:                        p.NetworkLayer().NetworkFlow().Src().String(),
			DstIP:                        p.NetworkLayer().NetworkFlow().Dst().String(),
			SrcMAC:                       p.LinkLayer().LinkFlow().Src().String(),
			DstMAC:                       p.LinkLayer().LinkFlow().Dst().String(),
			SrcPort:                      int32(srcPort),
			DstPort:                      int32(dstPort),
			Extensions:                   extensions,
		}
	}
	return nil
}, nil)