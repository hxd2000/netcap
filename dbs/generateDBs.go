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

package dbs

import (
	"fmt"
	"github.com/dreadl0ck/netcap/defaults"
	"github.com/dustin/go-humanize"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// A simple hook function that provides the option to modify the fetched data
type datasourceHook func(in string, d *datasource, base string) error

type datasource struct {
	url  string
	name string
	hook datasourceHook
}

func makeSource(url, name string, hook datasourceHook) *datasource {
	// if no name provided: use base
	if name == "" {
		name = filepath.Base(url)
	}
	return &datasource{
		url:  url,
		name: name,
		hook: hook,
	}
}

/*
 * Sources
 */

var sources = []*datasource{
	makeSource("http://s3.amazonaws.com/alexa-static/top-1m.csv.zip", "domain-whitelist.csv", moveToDbs),
	makeSource("https://raw.githubusercontent.com/tobie/ua-parser/master/regexes.yaml", "", moveToDbs),
	makeSource("https://svn.nmap.org/nmap/nmap-service-probes", "", moveToDbs),
	makeSource("https://macaddress.io/database-download", "macaddress.io-db.json", moveToDbs),
	makeSource("https://ja3er.com/getAllHashesJson", "ja3erDB.json", moveToDbs),
	makeSource("https://ja3er.com/getAllUasJson", "ja3UserAgents.json", moveToDbs),
	makeSource("https://github.com/dreadl0ck/netcap-dbs/blob/main/dbs/ja_3_3s.json", "", moveToDbs),
	makeSource("https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.csv", "", moveToDbs),
	makeSource("https://raw.githubusercontent.com/0x4D31/hassh-utils/master/hasshdb", "hasshdb.txt", moveToDbs), // hasshdb.json
	makeSource("https://github.com/trisulnsm/trisul-scripts/blob/master/lua/frontend_scripts/reassembly/ja3/prints/ja3fingerprint.json", "ja3fingerprint.json", moveToDbs),
	makeSource("https://raw.githubusercontent.com/karottc/fingerbank/master/upstream/startup/fingerprints.csv", "", moveToDbs), // dhcp-fingerprints.json
	makeSource("https://github.com/AliasIO/wappalyzer/blob/master/src/technologies.json", "", moveToDbs),                       // cmsdb.json
	makeSource("https://web.archive.org/web/20191227182527if_/https://geolite.maxmind.com/download/geoip/database/GeoLite2-ASN.tar.gz", "", untarAndMoveToDbs),
	makeSource("https://web.archive.org/web/20191227182209if_/https://geolite.maxmind.com/download/geoip/database/GeoLite2-City.tar.gz", "", untarAndMoveToDbs),
	makeSource("", "nvd.bleve", downloadAndIndexNVD),
	makeSource("https://raw.githubusercontent.com/offensive-security/exploitdb/master/files_exploits.csv", "exploit-db.bleve", downloadAndIndexExploitDB),
}

/*
 * Datasource Hooks
 */

func downloadAndIndexNVD(_ string, _ *datasource, base string) error {
	for _, year := range yearRange(2002, time.Now().Year()) {
		s := makeSource(fmt.Sprintf("https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-%s.json.gz", year), "", nil)
		fetchResource(s, filepath.Join(base, "build", s.name))
	}
	IndexData("nvd", filepath.Join(base, "dbs"))
	return nil
}

func downloadAndIndexExploitDB(_ string, _ *datasource, base string) error {
	IndexData("exploit-db", filepath.Join(base, "dbs"))
	return nil
}

func moveToDbs(in string, d *datasource, base string) error {
	return os.Rename(in, filepath.Join(base, "dbs", d.name))
}

// unpack compressed tarballs and move certain files to the dbs directory
// currently only used to extract *.mmdb files
func untarAndMoveToDbs(in string, d *datasource, base string) error {
	f, err := os.Open(in)
	if err != nil {
		return err
	}
	defer f.Close()

	name, err := unpackTarball(f, filepath.Join(base, "build"))
	fmt.Println("unpacked", name)

	// extract *.mmdb files
	files, err := filepath.Glob(filepath.Join(base, "build", name, "*.mmdb"))
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println("extracting file", file)
		err = os.Rename(
			filepath.Join(base, "build", name, filepath.Base(file)),
			filepath.Join(base, "dbs", filepath.Base(file)),
		)
		if err != nil {
			log.Fatal(err)
		}
	}

	return err
}

/*
 * Main
 */

var (
	numBytesFetched   uint64
	numBytesFetchedMu sync.Mutex
)

func GenerateDBs() {

	var (
		base  = "netcap-dbs-generated"
		_     = os.MkdirAll(filepath.Join(base, "build"), defaults.DirectoryPermission)
		_     = os.MkdirAll(filepath.Join(base, "dbs"), defaults.DirectoryPermission)
		wg    sync.WaitGroup
		start = time.Now()
		total int
	)

	for _, s := range sources {
		total++
		wg.Add(1)
		go processSource(s, base, &wg)
	}

	time.Sleep(1 * time.Second)
	fmt.Println("waiting for downloads to complete...")
	wg.Wait()

	out, err := exec.Command("tree", base).CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(out))

	fmt.Println("fetched", total, "sources ("+humanize.Bytes(numBytesFetched)+")", "in", time.Since(start))
}

func processSource(s *datasource, base string, wg *sync.WaitGroup) {

	outFilePath := filepath.Join(base, "build", s.name)

	// fetch via HTTP GET from single remote source if provided
	// if multiple sources need to be fetched, the logic can be implemented in the hook
	fetchResource(s, outFilePath)

	// run hook
	if s.hook != nil {
		err := s.hook(outFilePath, s, base)
		if err != nil {
			log.Println("hook for", s.name, "failed with error", err)
		}
	}

	wg.Done()
}

func fetchResource(s *datasource, outFilePath string) {
	if s.url != "" {

		fmt.Println("fetching", s.name, "from", s.url)

		// execute GET request
		resp, err := http.Get(s.url)
		if err != nil {
			log.Fatal("failed to retrieve data from", s)
		}

		// check status
		if resp.StatusCode != http.StatusOK {
			log.Fatal("failed to retrieve data from", s, resp.Status)
		}

		// read body data
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("failed to read body data from", s, " error: ", err)
		}

		numBytesFetchedMu.Lock()
		numBytesFetched += uint64(len(data))
		numBytesFetchedMu.Unlock()

		// create output file in build folder
		f, err := os.Create(outFilePath)
		if err != nil {
			log.Fatal("failed to create file in build folder", s, " error: ", err)
		}

		// write data into file
		_, err = f.Write(data)
		if err != nil {
			log.Fatal("failed to write data to file in build folder", s, " error: ", err)
		}

		// close the file
		err = f.Close()
		if err != nil {
			log.Fatal("failed to close file in build folder", s, " error: ", err)
		}
	}
}

// TODO: automate generation of cmsdb.json from the technologies.json file
type WebTechnologies struct {
	Schema     string `json:"$schema"`
	Categories struct {
		Num1 struct {
			Name     string `json:"name"`
			Priority int    `json:"priority"`
		} `json:"1"`
	} `json:"categories"`
	Technologies struct {
		OneCBitrix struct {
			Cats        []int  `json:"cats"`
			Description string `json:"description"`
			Headers     struct {
				SetCookie   string `json:"Set-Cookie"`
				XPoweredCMS string `json:"X-Powered-CMS"`
			} `json:"headers"`
			HTML    string `json:"html"`
			Icon    string `json:"icon"`
			Implies string `json:"implies"`
			Scripts string `json:"scripts"`
			Website string `json:"website"`
		} `json:"1C-Bitrix"`
	} `json:"technologies"`
}