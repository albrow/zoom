package zoom

// Query represents a query which will retrieve some models from
// the database. A Query may consist of one or more query modifiers
// (e.g. Filter or Order) and may be executed with a query finisher
// (e.g. Run or Ids).
type Query struct {
	*query
}

// NewQuery is used to construct a query. The query returned can be chained
// together with one or more query modifiers (e.g. Filter or Order), and then
// executed using the Run, RunOne, Count, or Ids methods. If no query modifiers
// are used, running the query will return all models of the given type in
// unspecified order. Queries use delayed execution, so nothing touches the
// database until you execute them.
func (collection *Collection) NewQuery() *Query {
	return &Query{
		query: newQuery(collection),
	}
}

// Order specifies a field by which to sort the models. fieldName should be a
// field in the struct type corresponding to the Collection used in the query
// constructor. By default, the records are sorted by ascending order by the
// given field. To sort by descending order, put a negative sign before the
// field name. Zoom can only sort by fields which have been indexed, i.e. those
// which have the `zoom:"index"` struct tag. Only one order may be specified per
// Order will set an error on the query if the fieldName is invalid, if another
// order has already been applied to the query, or if the fieldName specified
// does not correspond to an indexed field. The error, same as any other error
// that occurs during the lifetime of the query, is not returned until the query
// is executed.
func (q *Query) Order(fieldName string) *Query {
	q.query.Order(fieldName)
	return q
}

// Limit specifies an upper limit on the number of models to return. If amount
// is 0, no limit will be applied and any number of models may be returned. The
// default value is 0.
func (q *Query) Limit(amount uint) *Query {
	q.query.Limit(amount)
	return q
}

// Offset specifies a starting index (inclusive) from which to start counting
// models that will be returned. For example, if offset is 10, the first 10
// models that the query would otherwise return will be skipped. The default
// value is 0.
func (q *Query) Offset(amount uint) *Query {
	q.query.Offset(amount)
	return q
}

// Include specifies one or more field names which will be read from the
// database and scanned into the resulting models when the query is run. Field
// names which are not specified in Include will not be read or scanned. You can
// only use one of Include or Exclude, not both on the same query. Include will
// set an error if you try to use it with Exclude on the same query. The error,
// same as any other error that occurs during the lifetime of the query, is not
// returned until the query is executed.
func (q *Query) Include(fields ...string) *Query {
	q.query.Include(fields...)
	return q
}

// Exclude specifies one or more field names which will *not* be read from the
// database and scanned. Any other fields *will* be read and scanned into the
// resulting models when the query is run. You can only use one of Include or
// Exclude, not both on the same query. Exclude will set an error if you try to
// use it with Include on the same query. The error, same as any other error
// that occurs during the lifetime of the query, is not returned until the query
// is executed.
func (q *Query) Exclude(fields ...string) *Query {
	q.query.Exclude(fields...)
	return q
}

// Filter applies a filter to the query, which will cause the query to only
// return models with field values matching the expression. filterString should
// be an expression which includes a fieldName, a space, and an operator in that
// order. For example: Filter("Age >=", 30) would only return models which have
// an Age value greater than or equal to 30. Operators must be one of "=", "!=",
// ">", "<", ">=", or "<=". You can only use Filter on fields which are indexed,
// i.e. those which have the `zoom:"index"` struct tag. If multiple filters are
// applied to the same query, the query will only return models which have
// matches for *all* of the filters. Filter will set an error on the query if
// the arguments are improperly formated, if the field you are attempting to
// filter is not indexed, or if the type of value does not match the type of the
// field. The error, same as any other error that occurs during the lifetime of
// the query, is not returned until the query is executed.
func (q *Query) Filter(filterString string, value interface{}) *Query {
	q.query.Filter(filterString, value)
	return q
}

// Run executes the query and scans the results into models. The type of models
// should be a pointer to a slice of Models. If no models fit the criteria, Run
// will set the length of models to 0 but will *not* return an error. Run will
// return the first error that occurred during the lifetime of the query (if
// any), or if models is the wrong type.
func (q *Query) Run(models interface{}) error {
	tx := q.pool.NewTransaction()
	newTransactionalQuery(q.query, tx).Run(models)
	return tx.Exec()
}

// RunOne is exactly like Run but finds only the first model that fits the query
// criteria and scans the values into model. If no model fits the criteria,
// RunOne *will* return a ModelNotFoundError.
func (q *Query) RunOne(model Model) error {
	tx := q.pool.NewTransaction()
	newTransactionalQuery(q.query, tx).RunOne(model)
	return tx.Exec()
}

// Count counts the number of models that would be returned by the query without
// actually retrieving the models themselves. Count will also return the first
// error that occurred during the lifetime of the query (if any).
func (q *Query) Count() (int, error) {
	tx := q.pool.NewTransaction()
	var count int
	newTransactionalQuery(q.query, tx).Count(&count)
	if err := tx.Exec(); err != nil {
		return 0, err
	}
	return count, nil
}

// Ids returns only the ids of the models without actually retrieving the
// models themselves. Ids will return the first error that occurred during the
// lifetime of the query (if any).
func (q *Query) Ids() ([]string, error) {
	tx := q.pool.NewTransaction()
	ids := []string{}
	newTransactionalQuery(q.query, tx).Ids(&ids)
	if err := tx.Exec(); err != nil {
		return nil, err
	}
	return ids, nil
}

// StoreIds executes the query and stores the model ids matching the query
// criteria in a list identified by destKey. The list will be completely
// overwritten, and the model ids stored there will be in the correct order if
// the query includes an Order modifier. StoreIds will return the first error
// that occurred during the lifetime of the query (if any).
func (q *Query) StoreIds(destKey string) error {
	tx := q.pool.NewTransaction()
	newTransactionalQuery(q.query, tx).StoreIds(destKey)
	return tx.Exec()
}
