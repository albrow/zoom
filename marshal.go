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

var (
	// GobMarshalerUnmarshaler is an object that implements MarshalerUnmarshaler
	// and uses uses the builtin gob package. Note that not all types are
	// supported by the gob package. See https://golang.org/pkg/encoding/gob/
	GobMarshalerUnmarshaler MarshalerUnmarshaler = gobMarshalerUnmarshaler{}
	// JSONMarshalerUnmarshaler is an object that implements MarshalerUnmarshaler
	// and uses the builtin json package. Note that not all types are supported
	// by the json package. See https://golang.org/pkg/encoding/json/#Marshal
	JSONMarshalerUnmarshaler MarshalerUnmarshaler = jsonMarshalerUnmarshaler{}
)

// gobMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// uses the builtin gob encoding. Note that not all types are supported by
// the gob package. See https://golang.org/pkg/encoding/gob/
type gobMarshalerUnmarshaler struct{}

// jsonMarshalerUnmarshaler is an implementation of MarshalerUnmarshaler that
// use the builtin json package. Note that not all types are supported by the
// json package. See https://golang.org/pkg/encoding/json/#Marshal
type jsonMarshalerUnmarshaler struct{}

// Marshal returns the gob encoding of v.
func (gobMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	if err := enc.Encode(v); err != nil {
		return buff.Bytes(), err
	}
	return buff.Bytes(), nil
}

// Unmarshal parses the gob-encoded data and stores the result in the value
// pointed to by v.
func (gobMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	buff := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buff)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

// Marshal returns the json encoding of v.
func (jsonMarshalerUnmarshaler) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal parses the json-encoded data and stores the result in the value
// pointed to by v.
func (jsonMarshalerUnmarshaler) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
