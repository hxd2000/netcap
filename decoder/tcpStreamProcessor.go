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
	"fmt"
	"github.com/dreadl0ck/netcap/reassembly"
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"time"
)

var (
	streamDecodeTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nc_stream_decode_time",
			Help: "Time taken to process a TCP stream",
		},
		[]string{"Decoder"},
	)
	streamFeedDataTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nc_stream_feed_data_time",
			Help: "Time taken to feed data to a TCP stream consumer",
		},
		[]string{"Direction"},
	)
	streamProcessingTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nc_stream_processing_time",
			Help: "Time taken to save the data to disk",
		},
		[]string{"Direction"},
	)
)

func init() {
	prometheus.MustRegister(
		streamProcessingTime,
		streamDecodeTime,
		streamFeedDataTime,
	)
}

// internal data structure to parallelize processing of tcp streams
// when the core engine is stopped and the remaining open connections are processed.
type tcpStreamProcessor struct {
	workers          []chan streamReader
	numWorkers       int
	next             int
	wg               sync.WaitGroup
	numDone          int
	numTotal         int
	streamBufferSize int
	sync.Mutex
}

// to process the streams in parallel
// they are passed to several worker goroutines in round robin style.
func (tsp *tcpStreamProcessor) handleStream(s streamReader) {
	tsp.wg.Add(1)

	// make it work for 1 worker only, can be used for debugging
	//if c.numWorkers == 1 {
	//	c.workers[0] <- s
	//	return
	//}

	// send the packetInfo to the encoder routine
	tsp.workers[tsp.next] <- s

	// increment or reset next
	if tsp.numWorkers == tsp.next+1 {
		// reset
		tsp.next = 0
	} else {
		tsp.next++
	}
}

// worker spawns a new worker goroutine
// and returns a channel for receiving input packets.
// the wait group has already been incremented for each non-nil packet,
// so wg.Done() must be called before returning for each item.
func (tsp *tcpStreamProcessor) streamWorker(wg *sync.WaitGroup) chan streamReader {
	// init channel to receive input packets
	chanInput := make(chan streamReader, tsp.streamBufferSize)

	// start worker
	go func() {
		for s := range chanInput {
			// nil packet is used to exit the loop,
			// the processing logic will never send a streamReader in here that is nil
			if s == nil {
				return
			}

			// do not process streams that have been saved already by their cleanup functions
			// because the corresponding connection has been closed
			if s.Saved() {
				wg.Done()

				continue
			}

			t := time.Now()
			if s.IsClient() {
				// save the entire conversation.
				// we only need to do this once, when the client part of the connection is closed
				err := saveConnection(s.ConversationRaw(), s.ConversationColored(), s.Ident(), s.FirstPacket(), s.Transport())
				if err != nil {
					fmt.Println("failed to save connection", err)
				}

				streamProcessingTime.WithLabelValues(reassembly.TCPDirClientToServer.String()).Set(float64(time.Since(t).Nanoseconds()))
			} else {
				s.SortAndMergeFragments()

				// save the service banner
				saveTCPServiceBanner(s)

				streamProcessingTime.WithLabelValues(reassembly.TCPDirServerToClient.String()).Set(float64(time.Since(t).Nanoseconds()))
			}

			tsp.Lock()
			tsp.numDone++

			if !conf.Quiet {
				clearLine()
				fmt.Print("processing remaining open TCP streams... ", "(", tsp.numDone, "/", tsp.numTotal, ")")
			}

			tsp.Unlock()
			wg.Done()
		}
	}()

	// return input channel
	return chanInput
}

// spawn the configured number of workers.
func (tsp *tcpStreamProcessor) initWorkers(streamBufferSize int) {
	tsp.streamBufferSize = streamBufferSize

	// TODO: make configurable
	tsp.workers = make([]chan streamReader, 1000)

	for i := range tsp.workers {
		tsp.workers[i] = tsp.streamWorker(&tsp.wg)
	}

	tsp.numWorkers = len(tsp.workers)
}
