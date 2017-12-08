// Freeze file containing urls along with the content of all urls. Frozen file
// will be a zip file, containing (by default):
//
//     /blob
//     /mapping.json
//     /files/<sha1 of url>
//     /files/<sha1 of url>
//     /files/...
//
package main

import (
	"archive/zip"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/miku/span"
	"mvdan.cc/xurls"
)

const (
	NameBlob    = "blob"
	NameMapping = "mapping.json"
	NameDir     = "files"
)

var (
	output      = flag.String("o", "", "output file")
	bestEffort  = flag.Bool("b", false, "report errors but do not stop")
	showVersion = flag.Bool("v", false, "prints current program version")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(span.AppVersion)
		os.Exit(0)
	}

	if *output == "" {
		log.Fatal("output file required")
	}

	file, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}

	w := zip.NewWriter(file)

	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	f, err := w.Create(NameBlob)
	if err != nil {
		log.Fatal(err)
	}
	f.Write(b)

	urls := xurls.Strict().FindAllString(string(b), -1)

	seen := make(map[string]bool)
	var unique []string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if _, ok := seen[u]; !ok {
			unique = append(unique, u)
			seen[u] = true
		}
	}

	// Not necessary, but keep an additional mapping to simplify reading later.
	mapping := make(map[string]string)

	for i, u := range unique {
		h := sha1.New()
		h.Write([]byte(u))
		name := fmt.Sprintf("%s/%x", NameDir, h.Sum(nil))

		mapping[u] = name

		resp, err := http.Get(u)
		if err != nil {
			if *bestEffort {
				log.Printf("[%04d %s] %v", i, u, err)
				continue
			} else {
				log.Fatal(err)
			}
		}
		defer resp.Body.Close()

		f, err := w.Create(name)
		if err != nil {
			log.Fatal(err)
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			log.Fatal(err)
		}
		log.Printf("[%04d %s] %s", i, name, u)
	}

	f, err = w.Create(NameMapping)
	if err != nil {
		log.Fatal(err)
	}
	if err := json.NewEncoder(f).Encode(mapping); err != nil {
		log.Fatal(err)
	}
	if err := w.Close(); err != nil {
		log.Fatal(err)
	}
}