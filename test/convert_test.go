package test

import (
	"github.com/stephenalexbrowne/zoom"
	"github.com/stephenalexbrowne/zoom/test_support"
	"github.com/stephenalexbrowne/zoom/util"
	"testing"
)

func TestPrimativeTypes(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	pts, err := test_support.NewPrimativeTypes(1)
	if err != nil {
		t.Error(err)
	}
	pt := pts[0]
	zoom.Save(pt)

	ptCopy := &test_support.PrimativeTypes{}
	if _, err := zoom.ScanById(pt.Id, ptCopy).Run(); err != nil {
		t.Error(err)
	}

	equal, err := util.Equals(pt, ptCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", pt, ptCopy)
	}
}
