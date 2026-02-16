package repotools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeFingerprint(t *testing.T) {
	// this test runs in the ox repo which has git history
	fp, err := ComputeFingerprint()
	require.NoError(t, err, "ComputeFingerprint() failed")

	// verify first commit is populated
	assert.NotEmpty(t, fp.FirstCommit, "expected FirstCommit to be non-empty")

	// verify first commit is a 40-char hex string (SHA-1)
	assert.Len(t, fp.FirstCommit, 40, "expected FirstCommit to be 40 chars: %s", fp.FirstCommit)

	// verify ancestry samples are populated (repo has commits)
	assert.NotEmpty(t, fp.AncestrySamples, "expected AncestrySamples to be non-empty")

	// first ancestry sample should be the first commit
	assert.Equal(t, fp.FirstCommit, fp.AncestrySamples[0],
		"expected first ancestry sample to equal FirstCommit")

	// verify monthly checkpoints (repo has recent commits)
	assert.NotEmpty(t, fp.MonthlyCheckpoints, "expected MonthlyCheckpoints to be non-empty for active repo")

	// verify month key format is "YYYY-MM"
	for monthKey := range fp.MonthlyCheckpoints {
		assert.Len(t, monthKey, 7, "expected month key format YYYY-MM, got: %s", monthKey)
		assert.Equal(t, byte('-'), monthKey[4], "expected month key format YYYY-MM, got: %s", monthKey)
	}
}

func TestComputeFingerprint_AncestrySamplePositions(t *testing.T) {
	fp, err := ComputeFingerprint()
	require.NoError(t, err, "ComputeFingerprint() failed")

	// verify we get samples at power-of-2 positions
	// expected positions (if repo has enough commits): 1, 2, 4, 8, 16, 32, 64, 128, 256
	// max possible samples is 9 (for 256+ commits)
	assert.LessOrEqual(t, len(fp.AncestrySamples), 9, "expected at most 9 ancestry samples")

	// all samples should be valid commit hashes (40 hex chars)
	for i, sample := range fp.AncestrySamples {
		assert.Len(t, sample, 40, "ancestry sample %d is not a valid commit hash: %s", i, sample)
	}
}

func TestRepoFingerprint_WithRemoteHashes(t *testing.T) {
	fp, err := ComputeFingerprint()
	require.NoError(t, err, "ComputeFingerprint() failed")

	// before calling WithRemoteHashes, RemoteHashes should be nil or empty
	assert.Empty(t, fp.RemoteHashes, "expected RemoteHashes to be empty before WithRemoteHashes()")

	// call WithRemoteHashes - may fail if no remotes configured
	err = fp.WithRemoteHashes()
	// don't fail on error - some test environments may not have remotes

	// if it succeeded and repo has remotes, verify hash format
	if err == nil && len(fp.RemoteHashes) > 0 {
		for i, hash := range fp.RemoteHashes {
			assert.GreaterOrEqual(t, len(hash), 10, "remote hash %d seems too short: %s", i, hash)
		}
	}
}

func TestRepoFingerprint_WithRemoteHashes_NilFingerprint(t *testing.T) {
	var fp *RepoFingerprint
	err := fp.WithRemoteHashes()
	assert.Error(t, err, "expected error when calling WithRemoteHashes on nil fingerprint")
}

func TestRepoFingerprint_WithRemoteHashes_EmptyFirstCommit(t *testing.T) {
	fp := &RepoFingerprint{}
	err := fp.WithRemoteHashes()
	assert.Error(t, err, "expected error when calling WithRemoteHashes with empty FirstCommit")
}

func TestComputeFingerprint_Deterministic(t *testing.T) {
	// calling ComputeFingerprint multiple times should return same values
	fp1, err := ComputeFingerprint()
	require.NoError(t, err, "ComputeFingerprint() failed")

	fp2, err := ComputeFingerprint()
	require.NoError(t, err, "ComputeFingerprint() failed on second call")

	assert.Equal(t, fp1.FirstCommit, fp2.FirstCommit, "FirstCommit not deterministic")
	assert.Len(t, fp2.AncestrySamples, len(fp1.AncestrySamples), "AncestrySamples length not deterministic")

	for i := range fp1.AncestrySamples {
		assert.Equal(t, fp1.AncestrySamples[i], fp2.AncestrySamples[i],
			"AncestrySamples[%d] not deterministic", i)
	}
}

func BenchmarkComputeFingerprint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ComputeFingerprint()
	}
}
