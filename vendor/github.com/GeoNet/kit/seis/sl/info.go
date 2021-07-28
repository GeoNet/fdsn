package sl

import (
	"encoding/xml"
)

type Info struct {
	XMLName xml.Name `xml:"seedlink"`

	Software     string `xml:"software,attr"`
	Organization string `xml:"organization,attr"`
	Started      string `xml:"started,attr"`
	Capability   []struct {
		Name string `xml:"name,attr"`
	} `xml:"capability"`
	Station []struct {
		Name        string `xml:"name,attr"`
		Network     string `xml:"network,attr"`
		Description string `xml:"description,attr"`
		BeginSeq    string `xml:"begin_seq,attr"`
		EndSeq      string `xml:"end_seq,attr"`
		StreamCheck string `xml:"stream_check,attr"`
		Stream      []struct {
			Location  string `xml:"location,attr"`
			Seedname  string `xml:"seedname,attr"`
			Type      string `xml:"type,attr"`
			BeginTime string `xml:"begin_time,attr"`
			EndTime   string `xml:"end_time,attr"`
		} `xml:"stream"`
	} `xml:"station"`
}

func (s *Info) Unmarshal(data []byte) error {
	return xml.Unmarshal(data, s)
}
