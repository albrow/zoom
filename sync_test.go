// Copyright 2014 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

package zoom

import (
	"testing"
	"time"
)

func TestAsyncUpdate(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// save a model
	m := &modelWithSync{
		Attr1: "A",
		Attr2: "B",
	}
	if err := Save(m); err != nil {
		t.Error(err)
	}

	// get two references to the model and update
	// each one asynchronously. We expect both changes to
	// stick.
	wait := make(chan bool)
	mCopy1, mCopy2 := &modelWithSync{}, &modelWithSync{}
	go func() {
		if err := ScanById(m.Id, mCopy1); err != nil {
			t.Error(err)
		}
		time.Sleep(50 * time.Millisecond)
		mCopy1.Attr1 = "B"
		if err := Save(mCopy1); err != nil {
			t.Error(err)
		}
		defer mCopy1.Unlock()
		wait <- true
	}()
	go func() {
		if err := ScanById(m.Id, mCopy2); err != nil {
			t.Error(err)
		}
		time.Sleep(50 * time.Millisecond)
		mCopy2.Attr2 = "C"
		if err := Save(mCopy2); err != nil {
			t.Error(err)
		}
		defer mCopy1.Unlock()
		wait <- true
	}()

	<-wait
	<-wait

	// get a third reference to the model
	mCopy3 := &modelWithSync{}
	if err := ScanById(m.Id, mCopy3); err != nil {
		t.Error(err)
	}

	// check that both attributes were updated
	if mCopy3.Attr1 != "B" {
		t.Errorf("Attr1 was not updated! Expected B but got %s", mCopy3.Attr1)
	}
	if mCopy3.Attr2 != "C" {
		t.Errorf("Attr2 was not updated! Expected C but got %s", mCopy3.Attr2)
	}
}
