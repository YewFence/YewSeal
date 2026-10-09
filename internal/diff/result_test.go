package diff

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryAddCountsOutcomes(t *testing.T) {
	summary := Summary{}
	summary.Add("a", "a.enc", DiffResult{}, nil)
	summary.Add("b", "b.enc", DiffResult{Different: true, Diff: "patch"}, nil)
	summary.Add("c", "c.enc", DiffResult{Skipped: MissingPlaintext}, nil)
	summary.Add("d", "d.enc", DiffResult{Skipped: NoMatchingIdentity}, nil)
	summary.Add("e", "e.enc", DiffResult{}, errors.New("boom"))

	assert.Equal(t, 2, summary.ComparedCount)
	assert.Equal(t, 1, summary.MissingInputCount)
	assert.Equal(t, 1, summary.NoIdentityCount)
	assert.Equal(t, 1, summary.FailedCount)
	require.Len(t, summary.Results, 5)
	assert.Equal(t, Outcome{
		PlaintextFile: "b",
		EncryptedFile: "b.enc",
		Diff:          "patch",
		Different:     true,
	}, summary.Results[1])
	assert.Equal(t, "boom", summary.Results[4].Error.Error())
}

func TestSummaryAddCountsErrorOverSkipped(t *testing.T) {
	summary := Summary{}
	summary.Add("a", "a.enc", DiffResult{Skipped: MissingCiphertext}, errors.New("boom"))

	assert.Equal(t, 1, summary.FailedCount)
	assert.Zero(t, summary.MissingInputCount)
	assert.Zero(t, summary.NoIdentityCount)
	assert.Zero(t, summary.ComparedCount)
}

func TestSummaryCheckPassesWithoutFailures(t *testing.T) {
	summary := Summary{}
	summary.Add("a", "a.enc", DiffResult{}, nil)
	require.NoError(t, summary.Check())
}

func TestSummaryCheckReportsFailureCounts(t *testing.T) {
	summary := Summary{}
	summary.Add("a", "a.enc", DiffResult{}, nil)
	summary.Add("b", "b.enc", DiffResult{}, errors.New("boom"))

	err := summary.Check()
	require.Error(t, err)
	assert.Equal(t, "1 of 2 files failed to compare", err.Error())
}
