package clean

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryAddCountsEachStatus(t *testing.T) {
	summary := Summary{}
	summary.Add(Result{PlaintextPath: "a", Status: StatusRemoved})
	summary.Add(Result{PlaintextPath: "b", Status: StatusAlreadyAbsent})
	summary.Add(Result{PlaintextPath: "c", Status: StatusRetained})
	summary.Add(Result{PlaintextPath: "d", Status: StatusFailed, Error: errors.New("boom")})

	assert.Equal(t, 1, summary.RemovedCount)
	assert.Equal(t, 1, summary.AlreadyAbsentCount)
	assert.Equal(t, 1, summary.RetainedCount)
	assert.Equal(t, 1, summary.FailedCount)
	assert.Len(t, summary.Results, 4)
}

func TestSummaryCheckPassesWithoutFailures(t *testing.T) {
	summary := Summary{}
	summary.Add(Result{Status: StatusRemoved})
	require.NoError(t, summary.Check())
}

func TestSummaryCheckReportsFailureCounts(t *testing.T) {
	summary := Summary{}
	summary.Add(Result{Status: StatusRemoved})
	summary.Add(Result{Status: StatusFailed})
	summary.Add(Result{Status: StatusFailed})

	err := summary.Check()
	require.Error(t, err)
	assert.Equal(t, "2 of 3 plaintext files failed to clean", err.Error())
}
