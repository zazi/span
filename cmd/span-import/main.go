// span-reshape is a dumbed down span-import.
package main

import (
	"encoding"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"bufio"

	"github.com/miku/parallel"
	"github.com/miku/span"
	"github.com/miku/span/formats/ceeol"
	"github.com/miku/span/formats/crossref"
	"github.com/miku/span/formats/degruyter"
	"github.com/miku/span/formats/doaj"
	"github.com/miku/span/formats/dummy"
	"github.com/miku/span/formats/elsevier"
	"github.com/miku/span/formats/finc"
	"github.com/miku/span/formats/genios"
	"github.com/miku/span/formats/highwire"
	"github.com/miku/span/formats/ieee"
	"github.com/miku/span/formats/imslp"
	"github.com/miku/span/formats/jstor"
	"github.com/miku/span/formats/thieme"
	"github.com/miku/xmlstream"
)

var (
	name        = flag.String("i", "", "input format name")
	list        = flag.Bool("list", false, "list input formats")
	numWorkers  = flag.Int("w", runtime.NumCPU(), "number of workers")
	showVersion = flag.Bool("v", false, "prints current program version")
	cpuprofile  = flag.String("cpuprofile", "", "write cpu profile to file")
)

// FormatMap maps format name to pointer to format struct.
var FormatMap = map[string]interface{}{
	"highwire":     new(highwire.Record),
	"ceeol":        new(ceeol.Article),
	"doaj":         new(doaj.Response),
	"crossref":     new(crossref.Document),
	"ieee":         new(ieee.Publication),
	"genios":       new(genios.Document),
	"jstor":        new(jstor.Article),
	"degruyter":    new(degruyter.Article),
	"elsevier-tar": struct{}{}, // It's complicated.
	"thieme-tm":    new(thieme.Document),
	"imslp":        new(imslp.Data),
	"dummy":        new(dummy.Example),
}

// IntermediateSchemaer wrap a basic conversion method.
type IntermediateSchemaer interface {
	ToIntermediateSchema() (*finc.IntermediateSchema, error)
}

// processXML converts XML based formats, given a format name. It reads XML as
// stream, finds records by given xml.Name and converts them to an intermediate
// schema at the moment.
func processXML(r io.Reader, w io.Writer, name string) error {
	if _, ok := FormatMap[name]; !ok {
		return fmt.Errorf("unknown format name: %s", name)
	}
	scanner := xmlstream.NewScanner(bufio.NewReader(r), FormatMap[name])
	for scanner.Scan() {
		tag := scanner.Element()
		converter, ok := tag.(IntermediateSchemaer)
		if !ok {
			return fmt.Errorf("cannot convert to intermediate schema: %T", tag)
		}
		output, err := converter.ToIntermediateSchema()
		if err != nil {
			if _, ok := err.(span.Skip); ok {
				continue
			}
			return err
		}
		if err := json.NewEncoder(w).Encode(output); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// processJSON convert JSON based formats. Input is interpreted as newline delimited JSON.
func processJSON(r io.Reader, w io.Writer, name string) error {
	if _, ok := FormatMap[name]; !ok {
		return fmt.Errorf("unknown format name: %s", name)
	}
	v := FormatMap[name]
	p := parallel.NewProcessor(r, w, func(b []byte) ([]byte, error) {
		if err := json.Unmarshal(b, v); err != nil {
			return nil, err
		}
		converter, ok := v.(IntermediateSchemaer)
		if !ok {
			return nil, fmt.Errorf("cannot convert to intermediate schema: %T", v)
		}
		output, err := converter.ToIntermediateSchema()
		if _, ok := err.(span.Skip); ok {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return json.Marshal(output)
	})
	return p.RunWorkers(*numWorkers)
}

// processText processes a single record from raw bytes.
func processText(r io.Reader, w io.Writer, name string) error {
	if _, ok := FormatMap[name]; !ok {
		return fmt.Errorf("unknown format name: %s", name)
	}
	// Get the format.
	data := FormatMap[name]

	// We need an unmarshaller first.
	unmarshaler, ok := data.(encoding.TextUnmarshaler)
	if !ok {
		return fmt.Errorf("cannot unmarshal text: %T", data)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if err := unmarshaler.UnmarshalText(b); err != nil {
		return err
	}

	// Now that data is populated we can convert.
	converter, ok := data.(IntermediateSchemaer)
	if !ok {
		return fmt.Errorf("cannot convert to intermediate schema: %T", data)
	}
	output, err := converter.ToIntermediateSchema()
	if _, ok := err.(span.Skip); ok {
		return nil
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(output)
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(span.AppVersion)
		os.Exit(0)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if *list {
		for k := range FormatMap {
			fmt.Println(k)
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

	switch *name {
	case "highwire", "ceeol", "ieee", "genios", "jstor":
		if err := processXML(reader, w, *name); err != nil {
			log.Fatal(err)
		}
	case "doaj", "crossref", "dummy":
		if err := processJSON(reader, w, *name); err != nil {
			log.Fatal(err)
		}
	case "imslp":
		if err := processText(reader, w, *name); err != nil {
			log.Fatal(err)
		}
	case "elsevier-tar":
		shipment, err := elsevier.NewShipment(reader)
		if err != nil {
			log.Fatal(err)
		}
		docs, err := shipment.BatchConvert()
		if err != nil {
			log.Fatal(err)
		}
		encoder := json.NewEncoder(w)
		for _, doc := range docs {
			if encoder.Encode(doc); err != nil {
				log.Fatal(err)
			}
		}
	default:
		log.Fatalf("unknown format: %s", *name)
	}
}
