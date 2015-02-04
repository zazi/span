package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/miku/span/holdings"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("input XML (ovid) required")
	}

	ff, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(ff)

	// XML decoder
	decoder := xml.NewDecoder(reader)
	var inElement string

	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			inElement = se.Name.Local
			if inElement == "holding" {
				var item holdings.Holding
				decoder.DecodeElement(&item, &se)
				// fmt.Printf("%+v\n", item)
				for _, ent := range item.Entitlements {
					obj := make(map[string]interface{})
					obj["pissn"] = item.PISSN
					obj["eissn"] = item.EISSN
					obj["status"] = ent.Status
					b, _ := json.Marshal(obj)
					fmt.Println(string(b))
				}
			}
		default:
		}
	}
}
