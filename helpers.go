// Copyright 2013 Alex Browne.  All rights reserved.
// Use of this source code is governed by the MIT
// license, which can be found in the LICENSE file.

// File helpers.go contains various public-facing helper methods.
// They are typically used to convert the responses of a query to their
// underlying type.

package zoom

import (
	"fmt"
	"reflect"
)

// Models converts an interface to a slice of Model. It is typically
// used to convert a return value of a MultiModelQuery.
func Models(in interface{}) []Model {
	typ := reflect.TypeOf(in)
	if !typeIsSliceOrArray(typ) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nArgument must be slice or array.", in)
		panic(msg)
	}
	elemTyp := typ.Elem()
	if !typeIsPointerToStruct(elemTyp) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nSlice or array must have elements of type pointer to struct.", in)
		panic(msg)
	}
	_, found := modelTypeToName[elemTyp]
	if !found {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nType %T is not registered.", in, in)
		panic(msg)
	}
	val := reflect.ValueOf(in)
	length := val.Len()
	results := make([]Model, length)
	for i := 0; i < length; i++ {
		elemVal := val.Index(i)
		model, ok := elemVal.Interface().(Model)
		if !ok {
			msg := fmt.Sprintf("zoom: panic in Models() - cannot convert type %T to Model", elemVal.Interface())
			panic(msg)
		}
		results[i] = model
	}
	return results
}

// Convert interface{} to []interface{}
func Interfaces(in interface{}) []interface{} {
	val := reflect.ValueOf(in)
	length := val.Len()
	results := make([]interface{}, length)
	for i := 0; i < length; i++ {
		elemVal := val.Index(i)
		results[i] = elemVal.Interface()
	}
	return results
}
