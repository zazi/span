// span-tag takes an intermediate schema file and a configuration forest of
// filters for various tags and runs all filters on every record of the input
// to produce a stream of tagged records.
//
// $ span-tag -c '{"DE-15": {"any": {}}}' < input.ldj > output.ldj
//
package main

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

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
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	freeze := flag.String("freeze", "", "freeze a filterconfig to a given filename")
	unfreeze := flag.String("unfreeze", "", "unfreeze a filterconfig from a file")

	flag.Parse()

	if *version {
		fmt.Println(span.AppVersion)
		os.Exit(0)
	}

	if *config == "" && *freeze == "" {
		log.Fatal("config file required, or unfreeze")
	}

	if *cpuprofile != "" {
		file, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(file)
		defer pprof.StopCPUProfile()
	}

	// The configuration tree.
	var tagger filter.Tagger

	// Register types to freeze.
	gob.Register(filter.AndFilter{})
	gob.Register(filter.AnyFilter{})
	gob.Register(filter.CollectionFilter{})
	gob.Register(filter.DOIFilter{})
	gob.Register(filter.HoldingsFilter{})
	gob.Register(filter.ISSNFilter{})
	gob.Register(filter.NotFilter{})
	gob.Register(filter.OrFilter{})
	gob.Register(filter.PackageFilter{})
	gob.Register(filter.SourceFilter{})
	gob.Register(filter.SourceFilter{})
	gob.Register(filter.SubjectFilter{})

	// Unfreezing preferred. XXX(miku): Unfreeze holdings and cache.
	if *unfreeze != "" {
		f, err := os.Open(*unfreeze)
		if err != nil {
			log.Fatal(err)
		}

		dec := gob.NewDecoder(f)
		if err := dec.Decode(tagger); err != nil {
			log.Fatal(err)
		}
		log.Printf("unfreeze from %s completed", *unfreeze)
	} else {
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
	}

	// At this points, we should have an in-memory representation of the tree.
	if *freeze != "" {
		f, err := os.Create(*freeze)
		if err != nil {
			log.Fatal(err)
		}
		enc := gob.NewEncoder(f)
		if err := enc.Encode(tagger); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
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
