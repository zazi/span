// span-tag takes an intermediate schema file and a configuration forest of
// filters for various tags and runs all filters on every record of the input
// to produce a stream of tagged records.
//
// $ span-tag -c '{"DE-15": {"any": {}}}' < input.ldj > output.ldj
//
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"

	log "github.com/sirupsen/logrus"

	"github.com/miku/span"
	"github.com/miku/span/filter"
	"github.com/miku/span/formats/finc"
	"github.com/miku/span/parallel"
)

func main() {
	config := flag.String("c", "", "JSON config file for filters")
	version := flag.Bool("v", false, "show version")
	size := flag.Int("b", 20000, "batch size")
	numWorkers := flag.Int("w", runtime.NumCPU(), "number of workers")
	cpuProfile := flag.String("cpuprofile", "", "write cpu profile to file")
	unfreeze := flag.String("unfreeze", "", "unfreeze filterconfig from a frozen file")

	flag.Parse()

	if *version {
		fmt.Println(span.AppVersion)
		os.Exit(0)
	}

	if *config == "" && *unfreeze == "" {
		log.Fatal("config file required")
	}

	if *cpuProfile != "" {
		file, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	// The configuration forest.
	var tagger filter.Tagger

	if *unfreeze != "" {
		dir, filterconfig, err := span.UnfreezeFilterConfig(*unfreeze)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("[span-tag] unfrooze filterconfig to: %s", filterconfig)
		defer os.RemoveAll(dir)
		*config = filterconfig
	}

	// Test, if we are given JSON directly.
	err := json.Unmarshal([]byte(*config), &tagger)
	if err != nil {
		// Fallback to parse config file.
		f, err := os.Open(*config)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&tagger); err != nil {
			log.Fatal(err)
		}
	}

	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	var reader io.Reader = os.Stdin

	if flag.NArg() > 0 {
		var files []io.Reader
		for _, filename := range flag.Args() {
			f, err := os.Open(filename)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			files = append(files, f)
		}
		reader = io.MultiReader(files...)
	}

	p := parallel.NewProcessor(bufio.NewReader(reader), w, func(_ int64, b []byte) ([]byte, error) {
		var is finc.IntermediateSchema
		if err := json.Unmarshal(b, &is); err != nil {
			return b, err
		}

		tagged := tagger.Tag(is)

		bb, err := json.Marshal(tagged)
		if err != nil {
			return bb, err
		}
		bb = append(bb, '\n')
		return bb, nil
	})

	p.NumWorkers = *numWorkers
	p.BatchSize = *size

	if err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
