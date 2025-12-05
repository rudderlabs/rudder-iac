package tasker

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResults(t *testing.T) {
	t.Parallel()

	var (
		results    = NewResults[int]()
		iterations = 2
	)

	for i := range iterations {
		t.Run(fmt.Sprintf("worker-%d", i+1), func(t *testing.T) {
			t.Parallel()
			results.Store(fmt.Sprintf("w%d-key1", i+1), i+1)
			val, ok := results.Get(fmt.Sprintf("w%d-key1", i+1))
			assert.True(t, ok)
			assert.Equal(t, i+1, val)
		})
	}
}
