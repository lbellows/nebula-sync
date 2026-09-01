package pihole

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessfulHTTPStatus(t *testing.T) {
	t.Parallel()

	require.NoError(t, successfulHTTPStatus(200, nil))
	require.NoError(t, successfulHTTPStatus(204, []byte("")))

	err := successfulHTTPStatus(403, []byte(`{"error":{"key":"forbidden"}}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 403")
	assert.Contains(t, err.Error(), `"key":"forbidden"`)
}

func TestSuccessfulHTTPStatusTruncatesBody(t *testing.T) {
	t.Parallel()

	err := successfulHTTPStatus(500, []byte(strings.Repeat("a", 1500)))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
	assert.True(t, strings.HasSuffix(err.Error(), "..."))
	assert.LessOrEqual(t, len(err.Error()), len("unexpected status code: 500, response body: ")+1024+3)
}
