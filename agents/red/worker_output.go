package agent

import (
	"fmt"

	"github.com/nexus-ia/nexus/db"
)

// latestWorkerOutput keeps the final result (or most recent text if no result
// exists) across every page. A long tool trace must not hide the conclusion from
// the planner. The caller supplies the existing local/source-aware read boundary.
func latestWorkerOutput(page func(int64, int) ([]db.Activity, int64, error)) (*db.Activity, error) {
	const pageSize = 1000
	var result, fallback *db.Activity
	var cursor int64
	for {
		items, next, err := page(cursor, pageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			switch item.Kind {
			case "result":
				copy := item
				result = &copy
			case "text":
				copy := item
				fallback = &copy
			}
		}
		if len(items) < pageSize {
			if result != nil {
				return result, nil
			}
			return fallback, nil
		}
		if next <= cursor {
			return nil, fmt.Errorf("worker output cursor did not advance")
		}
		cursor = next
	}
}
