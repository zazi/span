package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/miku/holdings"
	"github.com/miku/holdings/generic"
	"github.com/miku/span"
	"github.com/miku/span/finc"
)

func main() {

	// tags collects all -g X:Y for holding files
	var holdingtags span.TagSlice

	flag.Var(&holdingtags, "g", "label:holding-file")
	version := flag.Bool("v", false, "show version")

	flag.Parse()

	if *version {
		fmt.Println(span.AppVersion)
		os.Exit(0)
	}

	if len(holdingtags) == 0 {
		log.Fatal("at least one -tag is required")
	}

	// only use the first for now
	tag := holdingtags[0]

	// reader for intermediate schema
	var r *bufio.Reader

	if flag.NArg() == 0 {
		r = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		r = bufio.NewReader(file)
	}

	// generic holding file, autodetect format
	file, err := generic.New(tag.Value)
	if err != nil {
		log.Fatal(err)
	}

	// all license entries
	entries, err := file.ReadEntries()
	if err != nil {
		log.Fatal(err)
	}

	// iterate over records
	for {
		b, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		var is finc.IntermediateSchema
		if err := json.Unmarshal(b, &is); err != nil {
			log.Fatal(err)
		}

		signature := holdings.Signature{
			Date:   is.Date.Format("2006-01-02"),
			Volume: is.Volume,
			Issue:  is.Issue,
		}

		// validate record, if at least one license allows this item
		var valid bool

		// for each ISSN in record, check each license found in holding file
	LOOP:
		for _, issn := range append(is.ISSN, is.EISSN...) {
			for _, license := range entries.Licenses(issn) {
				if err := license.Covers(signature); err != nil {
					continue
				} else {
					if err := license.TimeRestricted(is.Date); err != nil {
						continue
					} else {
						valid = true
						break LOOP
					}
				}
			}
		}

		if valid {
			fmt.Printf("%s\t%v\n", is.RecordID, tag.Tag)
		} else {
			fmt.Printf("%s\t%v\n", is.RecordID, "X")
		}

	}
}