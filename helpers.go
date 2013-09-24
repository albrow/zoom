// File helpers.go contains various public-facing helper methods.
// They are typically used to convert the responses of a query to their
// underlying type.

package zoom

import (
	"fmt"
	"github.com/stephenalexbrowne/zoom/util"
	"reflect"
)

// Models converts an interface to a slice of Model. It is typically
// used to convert a return value of a FindAllQuery into the underlying
// type.
func Models(in interface{}) []Model {
	typ := reflect.TypeOf(in)
	if !util.TypeIsSliceOrArray(typ) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nArgument must be slice or array.", in)
		panic(msg)
	}
	elemTyp := typ.Elem()
	if !util.TypeIsPointerToStruct(elemTyp) {
		msg := fmt.Sprintf("zoom: panic in Models() - attempt to convert invalid type %T to []Model.\nSlice or array must have elements of type pointer to struct.", in)
		panic(msg)
	}
	_, found := typeToName[elemTyp]
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
