package main

import (
	"bytes"
	"encoding/gob"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var fdsnStations FDSNStationXML

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: xml2gob XML_FILE GOB_FILE")
		return
	}

	var b []byte
	var err error
	if b, err = ioutil.ReadFile(os.Args[1]); err != nil {
		log.Fatal(err.Error())
		return
	}

	if err = xml.Unmarshal(b, &fdsnStations); err != nil {
		log.Println("Error unmarshaling fdsn station xml", err.Error())
		return
	} else {
		log.Println("Done loading stations:", len(fdsnStations.Network[0].Station))
	}

	var by bytes.Buffer
	enc := gob.NewEncoder(&by)
	if err = enc.Encode(fdsnStations); err != nil {
		log.Println(err)
		return
	}

	if err = ioutil.WriteFile(os.Args[2], by.Bytes(), 0644); err != nil {
		log.Println(err)
		return
	}

	return
}

func (v xsdDateTime) MarshalBinary() ([]byte, error) {
	return v.MarshalText()
}

// UnmarshalBinary modifies the receiver so it must take a pointer receiver.
func (v *xsdDateTime) UnmarshalBinary(data []byte) error {
	return v.UnmarshalText(data)
}
