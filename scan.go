// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File scan.go contains code that converts go data structures
// to and from a format that redis can understand

package zoom

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func scanModel(bulk []interface{}, mr modelRef) error {
	if len(bulk)%2 != 0 {
		return errors.New("zoom: scanModel expects a list of key value pairs. len(bulk) must be even.")
	}
	for i := 0; i < len(bulk); i += 2 {
		bytes, ok := bulk[i].([]byte)
		if !ok {
			msg := fmt.Sprintf("zoom: could not convert bulk value %v of type %T to []byte\n", bulk[i], bulk[i])
			return errors.New(msg)
		}
		fieldName := string(bytes)
		ms := mr.modelSpec
		if _, found := ms.primatives[fieldName]; found {
			if err := scanPrimativeVal(bulk[i+1], mr.value(fieldName)); err != nil {
				return err
			}
		}
		if _, found := ms.pointers[fieldName]; found {
			if err := scanPointerVal(bulk[i+1], mr.value(fieldName)); err != nil {
				return err
			}
		}
		if _, found := ms.inconvertibles[fieldName]; found {
			if err := scanInconvertibleVal(bulk[i+1], mr.value(fieldName)); err != nil {
				return err
			}
		}
	}
	return nil
}

func scanPrimativeVal(src interface{}, dest reflect.Value) error {
	typ := dest.Type()
	srcBytes, ok := src.([]byte)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert %v of type %T to []byte.\n", src, src)
		return errors.New(msg)
	}
	if len(srcBytes) == 0 {
		return nil // skip blanks
	}
	if typeIsString(typ) {
		switch typ.Kind() {
		case reflect.String:
			// straight up string types
			srcString := string(srcBytes)
			dest.SetString(srcString)
		case reflect.Slice, reflect.Array:
			// slice or array of bytes
			dest.SetBytes(srcBytes)
		default:
			msg := fmt.Sprintf("zoom: don't know how to scan primative type: %T.\n", src)
			return errors.New(msg)
		}
	} else if typeIsNumeric(typ) {
		srcString := string(srcBytes)
		switch typ.Kind() {
		case reflect.Float32, reflect.Float64:
			// float types
			srcFloat, err := strconv.ParseFloat(srcString, 64)
			if err != nil {
				msg := fmt.Sprintf("zoom: could not convert %s to float.\n", srcString)
				return errors.New(msg)
			}
			dest.SetFloat(srcFloat)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			// int types
			srcInt, err := strconv.ParseInt(srcString, 10, 0)
			if err != nil {
				msg := fmt.Sprintf("zoom: could not convert %s to int.\n", srcString)
				return errors.New(msg)
			}
			dest.SetInt(srcInt)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// uint types
			srcUint, err := strconv.ParseUint(srcString, 10, 0)
			if err != nil {
				msg := fmt.Sprintf("zoom: could not convert %s to uint.\n", srcString)
				return errors.New(msg)
			}
			dest.SetUint(srcUint)
		default:
			msg := fmt.Sprintf("zoom: don't know how to scan primative type: %T.\n", src)
			return errors.New(msg)
		}
	} else if typeIsBool(typ) {
		srcString := string(srcBytes)
		srcBool, err := strconv.ParseBool(srcString)
		if err != nil {
			msg := fmt.Sprintf("zoom: could not convert %s to bool.\n", srcString)
			return errors.New(msg)
		}
		dest.SetBool(srcBool)
	} else {
		msg := fmt.Sprintf("zoom: don't know how to scan primative type: %T.\n", src)
		return errors.New(msg)
	}
	return nil
}

func scanPointerVal(src interface{}, dest reflect.Value) error {
	dest.Set(reflect.New(dest.Type().Elem()))
	return scanPrimativeVal(src, dest.Elem())
}

func scanInconvertibleVal(src interface{}, dest reflect.Value) error {
	srcBytes, ok := src.([]byte)
	if !ok {
		msg := fmt.Sprintf("zoom: could not convert %v of type %T to []byte.\n", src, src)
		return errors.New(msg)
	}
	if len(srcBytes) == 0 {
		return nil // skip blanks
	}

	// TODO: account for json, msgpack or other custom fallbacks
	if err := defaultMarshalerUnmarshaler.Unmarshal(srcBytes, dest.Addr().Interface()); err != nil {
		return err
	}
	return nil
}
