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

package decoder

import (
	"log"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/dreadl0ck/gopacket"
	"github.com/dreadl0ck/netcap/types"
	"github.com/dreadl0ck/netcap/utils"
	"github.com/gogo/protobuf/proto"
)

// ConnectionID is a bidirectional connection
// between two devices over the network
// that includes the Link, Network and TransportLayer
type ConnectionID struct {
	LinkFlowID      uint64
	NetworkFlowID   uint64
	TransportFlowID uint64
}

func (c ConnectionID) String() string {
	return strconv.FormatUint(c.LinkFlowID, 10) + strconv.FormatUint(c.NetworkFlowID, 10) + strconv.FormatUint(c.TransportFlowID, 10)
}

type Conn struct {
	*types.Connection
	sync.Mutex
}

// AtomicConnMap contains all connections and provides synchronized access
type AtomicConnMap struct {
	Items map[string]*Conn
	sync.Mutex
}

// Size returns the number of elements in the Items map
func (a *AtomicConnMap) Size() int {
	a.Lock()
	defer a.Unlock()

	return len(a.Items)
}

type ConnectionDecoder struct {
	*CustomDecoder
	Conns *AtomicConnMap
}

var connectionDecoder = &ConnectionDecoder{
	CustomDecoder: &CustomDecoder{
		Type:        types.Type_NC_Connection,
		Name:        "Connection",
		Description: "A connection represents bi-directional network communication between two hosts based on the combined link-, network- and transport layer identifiers",
	},
	Conns: &AtomicConnMap{
		Items: make(map[string]*Conn),
	},
}

func (cd *ConnectionDecoder) PostInit() error {

	// simply overwrite the handler with our custom one
	// this way the CustomEncoders default Decode() implementation will be used
	// (it takes care of applying config options and tracking stats)
	// but with our custom logic to handle the actual packet
	cd.Handler = cd.handlePacket

	return nil
}

// Destroy closes and flushes all writers and calls deinit if set
func (cd *ConnectionDecoder) Destroy() (name string, size int64) {

	// call Deinit on FlowDecoder, instead of CustomDecoder
	err := cd.DeInit()
	if err != nil {
		panic(err)
	}

	return cd.writer.Close()
}

func (cd *ConnectionDecoder) handlePacket(p gopacket.Packet) proto.Message {
	// assemble connectionID
	connID := ConnectionID{}
	if ll := p.LinkLayer(); ll != nil {
		connID.LinkFlowID = ll.LinkFlow().FastHash()
	}
	if nl := p.NetworkLayer(); nl != nil {
		connID.NetworkFlowID = nl.NetworkFlow().FastHash()
	}
	if tl := p.TransportLayer(); tl != nil {
		connID.TransportFlowID = tl.TransportFlow().FastHash()
	}

	// lookup flow
	cd.Conns.Lock()
	if conn, ok := cd.Conns.Items[connID.String()]; ok {

		// connID exists. update fields
		calcDuration := false

		conn.Lock()

		// check if received packet from the same flow
		// was captured BEFORE the flows first seen timestamp
		if !utils.StringToTime(conn.TimestampFirst).Before(p.Metadata().Timestamp) {

			calcDuration = true

			// rewrite timestamp
			conn.TimestampFirst = utils.TimeToString(p.Metadata().Timestamp)

			// rewrite source and destination parameters
			// since the first packet decides about the flow direction
			if ll := p.LinkLayer(); ll != nil {
				conn.SrcMAC = ll.LinkFlow().Src().String()
				conn.DstMAC = ll.LinkFlow().Dst().String()
			}
			if nl := p.NetworkLayer(); nl != nil {
				conn.SrcIP = nl.NetworkFlow().Src().String()
				conn.DstIP = nl.NetworkFlow().Dst().String()
			}
			if tl := p.TransportLayer(); tl != nil {
				conn.SrcPort = tl.TransportFlow().Src().String()
				conn.DstPort = tl.TransportFlow().Dst().String()
			}
		}

		// check if last timestamp was before the current packet
		if utils.StringToTime(conn.TimestampLast).Before(p.Metadata().Timestamp) {
			// current packet is newer
			// update last seen timestamp
			conn.TimestampLast = utils.TimeToString(p.Metadata().Timestamp)
			calcDuration = true
		} // else: do nothing, timestamp is still the oldest one

		conn.NumPackets++
		conn.TotalSize += int32(len(p.Data()))

		// only calculate duration when timetamps have changed
		if calcDuration {
			conn.Duration = utils.StringToTime(conn.TimestampLast).Sub(utils.StringToTime(conn.TimestampFirst)).Nanoseconds()
		}

		conn.Unlock()
	} else {

		// create a new Connection
		conn := &types.Connection{}
		conn.UID = calcMd5(conn.String())
		conn.TimestampFirst = utils.TimeToString(p.Metadata().Timestamp)

		if ll := p.LinkLayer(); ll != nil {
			conn.LinkProto = ll.LayerType().String()
			conn.SrcMAC = ll.LinkFlow().Src().String()
			conn.DstMAC = ll.LinkFlow().Dst().String()
		}
		if nl := p.NetworkLayer(); nl != nil {
			conn.NetworkProto = nl.LayerType().String()
			conn.SrcIP = nl.NetworkFlow().Src().String()
			conn.DstIP = nl.NetworkFlow().Dst().String()
		}
		if tl := p.TransportLayer(); tl != nil {
			conn.TransportProto = tl.LayerType().String()
			conn.SrcPort = tl.TransportFlow().Src().String()
			conn.DstPort = tl.TransportFlow().Dst().String()
		}
		if al := p.ApplicationLayer(); al != nil {
			conn.ApplicationProto = al.LayerType().String()
			conn.AppPayloadSize = int32(len(al.Payload()))
		}
		cd.Conns.Items[connID.String()] = &Conn{
			Connection: conn,
		}

		conns := atomic.AddInt64(&stats.numConns, 1)

		// flush
		if c.ConnFlushInterval != 0 && conns%int64(c.ConnFlushInterval) == 0 {

			var selectConns []*types.Connection
			for id, entry := range cd.Conns.Items {
				// flush entries whose last timestamp is connTimeOut older than current packet
				if p.Metadata().Timestamp.Sub(utils.StringToTime(entry.TimestampLast)) > c.ConnTimeOut {
					selectConns = append(selectConns, entry.Connection)
					// cleanup
					delete(cd.Conns.Items, id)
				}
			}

			// flush selection in background
			go func() {
				for _, c := range selectConns {
					cd.writeConn(c)
				}
			}()
		}
	}
	cd.Conns.Unlock()
	return nil
}

func (cd *ConnectionDecoder) DeInit() error {
	if !cd.writer.IsChanWriter {
		cd.Conns.Lock()
		for _, f := range cd.Conns.Items {
			f.Lock()
			cd.writeConn(f.Connection)
			f.Unlock()
		}
		cd.Conns.Unlock()
	}
	return nil
}

// writeConn writes the connection
func (cd *ConnectionDecoder) writeConn(conn *types.Connection) {

	if c.Export {
		conn.Inc()
	}

	atomic.AddInt64(&cd.numRecords, 1)
	err := cd.writer.Write(conn)
	if err != nil {
		log.Fatal("failed to write proto: ", err)
	}
}