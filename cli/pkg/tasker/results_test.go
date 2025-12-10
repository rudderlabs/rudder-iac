package tasker

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResults(t *testing.T) {
	t.Parallel()

	var (
		results    = NewResults[int]()
		iterations = 10
		wg         = sync.WaitGroup{}
	)

	for i := range iterations {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			results.Store(fmt.Sprintf("w%d-key1", i+1), i+1)
			val, ok := results.Get(fmt.Sprintf("w%d-key1", i+1))
			assert.True(t, ok)
			assert.Equal(t, i+1, val)
		}(i)
	}

	wg.Wait()
}
