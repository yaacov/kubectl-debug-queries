package query

import (
	"fmt"
	"runtime"
	"sync"
)

// CalculateBatchSize determines an appropriate batch size based on the total items
// and available CPU cores. If providedBatchSize > 0, it uses that instead.
func CalculateBatchSize(totalItems int, providedBatchSize int) int {
	if providedBatchSize > 0 {
		return providedBatchSize
	}

	numCPU := runtime.NumCPU()
	targetBatches := numCPU * 4
	batchSize := totalItems / targetBatches

	if batchSize < 100 && totalItems > 100 {
		batchSize = 100
	} else if batchSize < 1 {
		batchSize = 1
	}

	return batchSize
}

// FilterItemsParallel filters items in parallel batches using the WHERE clause.
func FilterItemsParallel(items []map[string]interface{}, queryOpts *QueryOptions, batchSize int) ([]map[string]interface{}, error) {
	totalItems := len(items)
	if totalItems == 0 {
		return []map[string]interface{}{}, nil
	}

	tree, err := ParseWhereClause(queryOpts.Where)
	if err != nil {
		return nil, err
	}

	effectiveBatchSize := CalculateBatchSize(totalItems, batchSize)
	numBatches := (totalItems + effectiveBatchSize - 1) / effectiveBatchSize

	type batchResult struct {
		index int
		items []map[string]interface{}
		err   error
	}

	results := make([]batchResult, numBatches)
	var wg sync.WaitGroup

	for i := 0; i < numBatches; i++ {
		wg.Add(1)
		go func(batchIndex int) {
			defer wg.Done()

			start := batchIndex * effectiveBatchSize
			end := start + effectiveBatchSize
			if end > totalItems {
				end = totalItems
			}

			filteredBatch, err := ApplyFilter(items[start:end], tree, queryOpts.Select)
			results[batchIndex] = batchResult{
				index: batchIndex,
				items: filteredBatch,
				err:   err,
			}
		}(i)
	}

	wg.Wait()

	var allResults []map[string]interface{}
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("error filtering batch %d: %v", r.index, r.err)
		}
		allResults = append(allResults, r.items...)
	}

	return allResults, nil
}
