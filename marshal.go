// Copyright 2015 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File marshal.go deals with encoding arbitrary data structures
// into byte format and decoding bytes into arbitrary data
// structures.

package zoom

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

// gobMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// uses the builtin gob encoding.
type gobMarshalerUnmarshaler struct{}

// defaultMarshalerUnmarshaler is used to marshal and unmarshal inconvertible
// fields whenever a custom MarshalerUnmarshaler is not provided.
var defaultMarshalerUnmarshaler MarshalerUnmarshaler = gobMarshalerUnmarshaler{}

// Marshal returns the gob encoding of v.
func (gobMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// Unmarshal parses the gob-encoded data and stores the result in the value pointed to by v.
func (gobMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}
