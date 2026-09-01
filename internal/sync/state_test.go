package sync

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_Add(t *testing.T) {
	s := NewState()
	require.Empty(t, s.Outcomes())

	outcome := *NewOutcome(true)
	s.Add(outcome)

	assert.Len(t, s.Outcomes(), 1)
	assert.Contains(t, s.Outcomes(), outcome)
}

func TestState_OnSuccess(t *testing.T) {
	s := NewState()
	require.Empty(t, s.Outcomes())

	s.OnSuccess()

	assert.Len(t, s.Outcomes(), 1)
	assert.True(t, s.Outcomes()[0].Success)
	now := time.Now()
	assert.WithinRange(t, s.Outcomes()[0].Timestamp, now.Add(-1*time.Second), now.Add(1*time.Second))
}

func TestState_OnFailure(t *testing.T) {
	s := NewState()
	require.Empty(t, s.Outcomes())

	s.OnFailure(errors.New("test error"))

	assert.Len(t, s.Outcomes(), 1)
	assert.False(t, s.Outcomes()[0].Success)
	now := time.Now()
	assert.WithinRange(t, s.Outcomes()[0].Timestamp, now.Add(-1*time.Second), now.Add(1*time.Second))
}
