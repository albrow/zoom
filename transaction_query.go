package zoom

import "github.com/garyburd/redigo/redis"

// TransactionQuery represents a query which will be run inside an existing
// transaction. A TransactionQuery may consist of one or more query modifiers
// (e.g. Filter or Order) and should always be finished with a query finisher
// (e.g. Run or IDs). Unlike Query, the finisher methods for TransactionQuery
// always expect pointers as arguments and will set the values when the
// corresponding Transaction is executed.
type TransactionQuery struct {
	*query
	tx *Transaction
}

// newTransactionQuery creates and returns a new TransactionQuery. It is an
// internal function that allows us to convert a Query to a TransactionQuery.
// That way, there is only one canonical implementation of the query finisher
// methods (e.g. Run, RunOne, IDs).
func newTransactionQuery(query *query, tx *Transaction) *TransactionQuery {
	return &TransactionQuery{
		query: query,
		tx:    tx,
	}
}

// Query is used to construct a query in the context of an existing Transaction
// It can be used to run a query atomically along with commands, scripts, or
// other queries in a single round trip. Note that this method returns a
// TransactionQuery, whereas Collection.NewQuery returns a Query. The two
// types are very similar, but there are differences in how they are eventually
// executed. Like a regular Query, a TransactionQuery can be chained together
// with one or more query modifiers (e.g. Filter or Order). You also need to
// finish the query with a method such as Run, RunOne, or Count. The major
// difference is that TransactionQueries are not actually run until you call
// Transaction.Exec(). As a consequence, the finisher methods (e.g. Run, RunOne,
// Count, etc) do not return anything. Instead they accept arguments which are
// then mutated after the transaction is executed.
func (tx *Transaction) Query(collection *Collection) *TransactionQuery {
	return &TransactionQuery{
		query: newQuery(collection),
		tx:    tx,
	}
}

// Order works exactly like Query.Order. See the documentation for Query.Order
// for a full description.
func (q *TransactionQuery) Order(fieldName string) *TransactionQuery {
	q.query.Order(fieldName)
	return q
}

// Limit works exactly like Query.Limit. See the documentation for Query.Limit
// for more information.
func (q *TransactionQuery) Limit(amount uint) *TransactionQuery {
	q.query.Limit(amount)
	return q
}

// Offset works exactly like Query.Offset. See the documentation for
// Query.Offset for more information.
func (q *TransactionQuery) Offset(amount uint) *TransactionQuery {
	q.query.Offset(amount)
	return q
}

// Include works exactly like Query.Include. See the documentation for
// Query.Include for more information.
func (q *TransactionQuery) Include(fields ...string) *TransactionQuery {
	q.query.Include(fields...)
	return q
}

// Exclude works exactly like Query.Exclude. See the documentation for
// Query.Exclude for more information.
func (q *TransactionQuery) Exclude(fields ...string) *TransactionQuery {
	q.query.Exclude(fields...)
	return q
}

// Filter works exactly like Query.Filter. See the documentation for
// Query.Filter for more information.
func (q *TransactionQuery) Filter(filterString string, value interface{}) *TransactionQuery {
	q.query.Filter(filterString, value)
	return q
}

// Run will run the query and scan the results into models when the Transaction
// is executed. It works very similarly to Query.Run, so you can check the
// documentation for Query.Run for more information. The first error encountered
// will be saved to the corresponding Transaction (if there is not already an
// error for the Transaction) and returned when you call Transaction.Exec.
func (q *TransactionQuery) Run(models interface{}) {
	if q.hasError() {
		q.tx.setError(q.err)
		return
	}
	if err := q.collection.spec.checkModelsType(models); err != nil {
		q.tx.setError(err)
		return
	}
	idsKey, tmpKeys, err := generateIDsSet(q.query, q.tx)
	if err != nil {
		q.tx.setError(err)
		return
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, q.redisFieldNames(), limit, q.offset, q.order.kind == descendingOrder)
	q.tx.Command("SORT", sortArgs, newScanModelsHandler(q.collection.spec, append(q.fieldNames(), "-"), models))
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
}

// RunOne will run the query and scan the first model which matches the query
// criteria into model. If no model matches the query criteria, it will set a
// ModelNotFoundError on the Transaction. It works very similarly to
// Query.RunOne, so you can check the documentation for Query.RunOne for more
// information. The first error encountered will be saved to the corresponding
// Transaction (if there is not already an error for the Transaction) and
// returned when you call Transaction.Exec.
func (q *TransactionQuery) RunOne(model Model) {
	if q.hasError() {
		q.tx.setError(q.err)
		return
	}
	if err := q.collection.spec.checkModelType(model); err != nil {
		q.tx.setError(err)
		return
	}
	idsKey, tmpKeys, err := generateIDsSet(q.query, q.tx)
	if err != nil {
		q.tx.setError(err)
		return
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, q.redisFieldNames(), 1, q.offset, q.order.kind == descendingOrder)
	q.tx.Command("SORT", sortArgs, newScanOneModelHandler(q.query, q.collection.spec, append(q.fieldNames(), "-"), model))
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
}

// Count will count the number of models that match the query criteria and set
// the value of count. It works very similarly to Query.Count, so you can check
// the documentation for Query.Count for more information. The first error
// encountered will be saved to the corresponding Transaction (if there is not
// already an error for the Transaction) and returned when you call
// Transaction.Exec.
func (q *TransactionQuery) Count(count *int) {
	if q.hasError() {
		q.tx.setError(q.err)
		return
	}
	if !q.hasFilters() {
		// Start by getting the number of models in the all index set
		q.tx.Command("SCARD", redis.Args{q.collection.spec.indexKey()}, func(reply interface{}) error {
			gotCount, err := redis.Int(reply, nil)
			if err != nil {
				return err
			}
			// Apply math to take into account limit and offset
			if q.hasOffset() {
				gotCount = gotCount - int(q.offset)
			}
			if q.hasLimit() && int(q.limit) < gotCount {
				gotCount = int(q.limit)
			}
			// Assign the value of count
			(*count) = gotCount
			return nil
		})
	} else {
		// If the query has filters, it is difficult to do any optimizations.
		// Instead we'll just count the number of ids that match the query
		// criteria. To do in a single transaction, we use the StoreIDs method and
		// then add a LLEN command.
		destKey := generateRandomKey("tmp:countDestKey")
		q.StoreIDs(destKey)
		q.tx.Command("LLEN", redis.Args{destKey}, NewScanIntHandler(count))
		// Delete the temporary destKey when we're done.
		q.tx.Command("DEL", redis.Args{destKey}, nil)
	}
}

// IDs will find the ids for models matching the query criteria and set the
// value of ids. It works very similarly to Query.IDs, so you can check the
// documentation for Query.IDs for more information. The first error encountered
// will be saved to the corresponding Transaction (if there is not already an
// error for the Transaction) and returned when you call Transaction.Exec.
func (q *TransactionQuery) IDs(ids *[]string) {
	if q.hasError() {
		q.tx.setError(q.err)
		return
	}
	idsKey, tmpKeys, err := generateIDsSet(q.query, q.tx)
	if err != nil {
		q.tx.setError(err)
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, nil, limit, q.offset, q.order.kind == descendingOrder)
	q.tx.Command("SORT", sortArgs, NewScanStringsHandler(ids))
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
}

// StoreIDs will store the ids for for models matching the criteria in a list
// identified by destKey. It works very similarly to Query.StoreIDs, so you can
// check the documentation for Query.StoreIDs for more information. The first
// error encountered will be saved to the corresponding Transaction (if there is
// not already an error for the Transaction) and returned when you call
// Transaction.Exec.
func (q *TransactionQuery) StoreIDs(destKey string) {
	if q.hasError() {
		q.tx.setError(q.err)
		return
	}
	idsKey, tmpKeys, err := generateIDsSet(q.query, q.tx)
	if err != nil {
		q.tx.setError(err)
	}
	limit := int(q.limit)
	if limit == 0 {
		// In our query syntax, a limit of 0 means unlimited
		// But in Redis, -1 means unlimited
		limit = -1
	}
	sortArgs := q.collection.spec.sortArgs(idsKey, nil, limit, q.offset, q.order.kind == descendingOrder)
	// Append the STORE argument to cause Redis to store the results in destKey.
	sortAndStoreArgs := append(sortArgs, "STORE", destKey)
	q.tx.Command("SORT", sortAndStoreArgs, nil)
	if len(tmpKeys) > 0 {
		q.tx.Command("DEL", (redis.Args{}).Add(tmpKeys...), nil)
	}
}
