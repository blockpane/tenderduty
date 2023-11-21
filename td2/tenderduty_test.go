package tenderduty

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestIsLatestBlockTimeCurrent(t *testing.T) {
	currentTime := time.Now().UTC()

	// Check current time
	blockTime := currentTime
	assert.True(t, isLatestBlockTimeCurrent(blockTime, 10))

	// Check time from two minutes ago
	blockTime = currentTime.Add(-time.Minute * 2)
	assert.True(t, isLatestBlockTimeCurrent(blockTime, 10))

	// Check time from two hours ago
	blockTime = currentTime.Add(-time.Hour * 2)
	assert.False(t, isLatestBlockTimeCurrent(blockTime, 10))

	// Check time two minutes in the future
	blockTime = currentTime.Add(time.Minute * 2)
	assert.True(t, isLatestBlockTimeCurrent(blockTime, 10))
}
