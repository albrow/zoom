// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File convert_test.go is intended to test the conversion
// to and from go data structures of a variety of types

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

func TestPointerToPrimativeTypes(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	pts, err := test_support.NewInconvertibleTypes(1)
	if err != nil {
		t.Error(err)
	}
	pt := pts[0]
	zoom.Save(pt)

	ptCopy := &test_support.InconvertibleTypes{}
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

func TestInconvertibleTypes(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	ms, err := test_support.NewInconvertibleTypes(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	zoom.Save(m)

	mCopy := &test_support.InconvertibleTypes{}
	if _, err := zoom.ScanById(m.Id, mCopy).Run(); err != nil {
		t.Error(err)
	}

	equal, err := util.Equals(m, mCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", m, mCopy)
	}
}

func TestEmbeddedStruct(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	ms, err := test_support.NewEmbeddedStructs(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	zoom.Save(m)

	mCopy := &test_support.EmbeddedStruct{}
	if _, err := zoom.ScanById(m.Id, mCopy).Run(); err != nil {
		t.Error(err)
	}

	equal, err := util.Equals(m, mCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", m, mCopy)
	}
}

func TestPointerEmbeddedStruct(t *testing.T) {
	test_support.SetUp()
	defer test_support.TearDown()

	ms, err := test_support.NewPointerEmbeddedStructs(1)
	if err != nil {
		t.Error(err)
	}
	m := ms[0]
	zoom.Save(m)

	mCopy := &test_support.PointerEmbeddedStruct{}
	if _, err := zoom.ScanById(m.Id, mCopy).Run(); err != nil {
		t.Error(err)
	}

	equal, err := util.Equals(m, mCopy)
	if err != nil {
		t.Error(err)
	}
	if !equal {
		t.Errorf("model was not saved/retrieved correctly.\nExpected: %+v\nGot %+v\n", m, mCopy)
	}
}
