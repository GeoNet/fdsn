package fs_test

import (
	"github.com/GeoNet/fdsn/internal/mseednrt/fs"
	"testing"
	"time"
)

func TestListChannels(t *testing.T) {
	c, err := fs.ListChannels("etc")
	if err != nil {
		t.Error(err)
	}

	if len(c) != 1073 {
		t.Errorf("expected 1073 channels got %d\n", len(c))
	}
}

func TestNSLC_ListRecords(t *testing.T) {
	n := fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"}

	r, err := n.ListRecords("etc")
	if err != nil {
		t.Error(err)
	}

	if len(r) != 1 {
		t.Errorf("expected 1 record got %d\n", len(r))
	}

	if r[0].End != 1549696978968393000 {
		t.Errorf("expected End 1549696978968393000 got %d\n", r[0].End)
	}

	if r[0].Start != 1549696978968393000 {
		t.Errorf("expected Start 1549696978968393000 got %d\n", r[0].Start)
	}

	if r[0].Path != "etc/NZ/ABAZ/10/EHE/6/07/1549696978968393000-1549696984368393000" {
		t.Errorf("expected path etc/NZ/ABAZ/10/EHE/6/07/1549696978968393000-1549696984368393000 got %s\n", r[0].Path)
	}
}

func TestPath(t *testing.T) {
	n := fs.NSLC{Network: "NZ", Station: "ABAZ", Location: "10", Channel: "EHE"}
	s := n.Path("etc", time.Unix(0, 1549696978968393000))
	if s != "etc/NZ/ABAZ/10/EHE/6/07" {
		t.Errorf("expected path etc/NZ/ABAZ/10/EHE/6/07 got %s\n", s)
	}

	s = n.Path("etc", time.Unix(0, 1549696984368393000))
	if s != "etc/NZ/ABAZ/10/EHE/6/07" {
		t.Errorf("expected path etc/NZ/ABAZ/10/EHE/6/07 got %s\n", s)
	}

}
