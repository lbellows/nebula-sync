package pihole

import (
	"io"
	"net/http"
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

func TestReadHTTPBodyLimit(t *testing.T) {
	t.Parallel()

	response := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("abcd"))}
	body, err := readHTTPBodyLimit(response, 4)
	require.NoError(t, err)
	assert.Equal(t, []byte("abcd"), body)

	response = &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("abcde"))}
	_, err = readHTTPBodyLimit(response, 4)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds 4 bytes")
}
