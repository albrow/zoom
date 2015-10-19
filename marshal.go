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
	"encoding/json"
)

// MarshalerUnmarshaler defines a handler for marshaling
// arbitrary data structures into a byte format and unmarshaling
// bytes into arbitrary data structures. Any struct which correctly
// implements the interface should have the property that what you
// put in using Marshal is identical to what you get out using
// Unmarshal.
type MarshalerUnmarshaler interface {
	// Marshal returns a byte-encoded representation of v
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal parses byte-encoded data and store the result in the value
	// pointed to by v.
	Unmarshal(data []byte, v interface{}) error
}

// GobMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// uses the builtin gob encoding. Note that not all types are supported by
// the gob package. See https://golang.org/pkg/encoding/gob/
type GobMarshalerUnmarshaler struct{}

// Make the compiler enforce that GobMarshalerUnmarshaler implements
// MarshalerUnmarshaler
var _ MarshalerUnmarshaler = GobMarshalerUnmarshaler{}

// JSONMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// use the builtin json package. Note that not all types are supported by the
// json package. See https://golang.org/pkg/encoding/json/#Marshal
type JSONMarshalerUnmarshaler struct{}

// Make the compiler enforce that JSONMarshalerUnmarshaler implements
// MarshalerUnmarshaler
var _ MarshalerUnmarshaler = JSONMarshalerUnmarshaler{}

// defaultMarshalerUnmarshaler is used to marshal and unmarshal inconvertible
// fields whenever a custom MarshalerUnmarshaler is not provided.
var defaultMarshalerUnmarshaler MarshalerUnmarshaler = GobMarshalerUnmarshaler{}

// Marshal returns the gob encoding of v.
func (GobMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// Unmarshal parses the gob-encoded data and stores the result in the value
// pointed to by v.
func (GobMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// Marshal returns the json encoding of v.
func (JSONMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal parses the json-encoded data and stores the result in the value
// pointed to by v.
func (JSONMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
