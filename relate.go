package zoom

import (
	"fmt"
	"reflect"
)

type Relation struct {
	Name string
	Ids  []string
}

// Return an interface{} as a result of an association lookup
func (m *Model) Fetch(relationName string) (interface{}, error) {
	// Get the id for the corresponding relation
	relation, err := findRelationByName(m.Parent, relationName)
	if err != nil {
		return nil, err
	}

	// find the result
	return FindById(relation.Name, relation.Ids[0])
}

// TODO: return a slice of interface{} for has many relations
func (m *Model) FetchAll(relationName string) ([]interface{}, error) {
	fmt.Println("TODO: implement FetchAll")

	relation, err := findRelationByName(m.Parent, relationName)
	if err != nil {
		return nil, err
	}
	fmt.Println("relation: ", relation)

	results := make([]interface{}, 0, len(relation.Ids))

	for _, id := range relation.Ids {
		item, err := FindById(relation.Name, id)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, nil
}

func fieldIsRelational(field reflect.StructField) bool {
	n := field.Name
	if n[len(n)-2:] == "Id" || n[len(n)-3:] == "Ids" {
		tag := field.Tag
		if tag.Get("refersTo") != "" {
			return true
		}
	}
	return false
}

func relationalModelName(field reflect.StructField) string {
	return field.Tag.Get("refersTo")
}

func findRelationByName(in interface{}, relationName string) (*Relation, error) {
	elem := reflect.ValueOf(in).Elem().Interface() // Get the actual element from the pointer
	val := reflect.ValueOf(elem)                   // for getting the actual field value
	typ := reflect.TypeOf(elem)                    // for name/type/kind information
	numFields := val.NumField()

	// we wish to iterate through the fields and find the one with the proper tags
	for i := 0; i < numFields; i++ {
		field := typ.Field(i)
		// skip the embedded Model struct since we know that's not
		// what we're looking for
		if field.Name == "Model" {
			continue
		}
		// there's a special case for relational attributes
		// a.k.a. those which include Id in the name and are
		// tagged with `refersTo:*`
		if fieldIsRelational(field) {
			if field.Tag.Get("as") == relationName {

				fieldVal := val.Field(i)
				ids := make([]string, 0, 1)

				// Special case for one-to-many relations
				if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
					// iterate through each element in the slice and append it to ids
					for i := 0; i < fieldVal.Len(); i++ {
						elem := fieldVal.Index(i)
						ids = append(ids, elem.String())
					}

				} else {
					// otherwise we're dealing with a one-to-one relation
					ids = append(ids, fieldVal.String())
				}

				relation := &Relation{
					Name: field.Tag.Get("refersTo"),
					Ids:  ids,
				}
				return relation, nil
			}
		}
	}

	return nil, NewRelationNotFoundError(relationName)
}

// Verifies that the refersTo tag is a valid model name and has been registered,
// if that's not the case, returns an error
func validateRelationalField(field reflect.StructField) error {
	relateName := relationalModelName(field)
	if !alreadyRegisteredName(relateName) {
		return NewModelNameNotRegisteredError(relateName)
	}
	return nil
}

// attempts to add to a relational index for one-to-many relations.
// the index is stored in redis as a set with a key of the form modelName:id:relationName
func addToIndex(prefix string, field reflect.StructField, arrayVal reflect.Value) error {

	// if the array is empty, we don't have to do anything
	if arrayVal.Len() == 0 {
		return nil
	}

	// get the relation name which will be used as a key
	relationName := field.Tag.Get("as")
	key := prefix + ":" + relationName

	// set up the argument slice which will be passed to the redis driver
	args := make([]interface{}, 0, 3)
	args = append(args, key)

	// iterate through each element in the slice and append it to args
	for i := 0; i < arrayVal.Len(); i++ {
		elem := arrayVal.Index(i)
		args = append(args, elem.Interface())
	}

	// invoke redis driver to write the ids to the database
	result := db.Command("sadd", args...)
	if result.Error() != nil {
		return result.Error()
	}

	return nil
}
