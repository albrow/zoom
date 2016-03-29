package zoom

import (
	"testing"
)

func TestTransactionQueries(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create some test models
	models, err := createAndSaveIndexedTestModels(10)
	if err != nil {
		t.Fatal(err)
	}

	// Create a transaction and add some queries to it
	tx := testPool.NewTransaction()
	queries := []*TransactionQuery{
		tx.Query(indexedTestModels),
		tx.Query(indexedTestModels).Filter("Int >", 3).Order("-String").Limit(3),
		// Note: Offset(11) means no models should be returned.
		tx.Query(indexedTestModels).Offset(11),
	}

	// Calculate the expected models and got models for each query
	gotModels := make([][]*indexedTestModel, len(queries))
	expectedModels := make([][]*indexedTestModel, len(queries))
	for i, query := range queries {
		expectedModels[i] = expectedResultsForQuery(query.query, models)
		modelsHolder := []*indexedTestModel{}
		query.Run(&modelsHolder)
		expectedModels[i] = modelsHolder
	}

	// Execute the transaction and check the results
	if err := tx.Exec(); err != nil {
		t.Fatal(err)
	}
	for i, query := range queries {
		if err := expectModelsToBeEqual(expectedModels[i], gotModels[i], true); err != nil {
			t.Errorf("Query %d failed: %s\n%v", i, query, err)
		}
		checkForLeakedTmpKeys(t, query.query)
	}
}

func TestTransactionQueriesError(t *testing.T) {
	testingSetUp()
	defer testingTearDown()

	// Create some test models
	if _, err := createAndSaveIndexedTestModels(10); err != nil {
		t.Fatal(err)
	}

	// Create a transaction and add a RunOne query to it. We expect this query to
	// fail because we used Offset(11) and there are only 10 models.
	tx := testPool.NewTransaction()
	gotModel := indexedTestModel{}
	query := tx.Query(indexedTestModels).Offset(11)
	query.RunOne(&gotModel)

	// Execute the transaction and check the results
	if err := tx.Exec(); err == nil {
		t.Error("Expected an error but got none")
	} else if _, ok := err.(ModelNotFoundError); !ok {
		t.Errorf("Expected a ModelNotFoundError but got: %T: %v", err, err)
	}
	checkForLeakedTmpKeys(t, query.query)
}
