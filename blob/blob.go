// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// Package blob deals with encoding arbitrary data structures
// into byte format and decoding bytes into arbitrary data
// structures.
package blob

import (
	"bytes"
	"encoding/gob"
)

// Interface MarshalerUnmarshaler defines a handler for marshaling
// arbitrary data structures into a byte format and unmarshaling
// bytes into arbitrary data structures. Any struct which correctly
// implements the interface should have the property that what you
// put in using Marshal is identical to what you get out using
// Unmarshal.
type MarshalerUnmarshaler interface {
	Marshal(v interface{}) ([]byte, error)      // Return a byte-encoded representation of v
	Unmarshal(data []byte, v interface{}) error // Parse byte-encoded data and store the result in the value pointed to by v.
}

// DefaultMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// uses the builtin gob encoding.
type DefaultMarshalerUnmarshaler struct{}

// Marshal returns the gob encoding of v.
func (DefaultMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// Unmarshal parses the gob-encoded data and stores the result in the value pointed to by v.
func (DefaultMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	var buff bytes.Buffer
	dec := gob.NewDecoder(&buff)
	buff.Write(data)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}
