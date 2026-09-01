package pihole

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/lovelaze/nebula-sync/internal/pihole/model"
)

// A client that never authenticated has no session to tear down. It must not
// return an error, otherwise deleteSessions() spends the full retry budget on a
// call that can never succeed.
func TestClient_DeleteSession_neverAuthenticated(t *testing.T) {
	logger := log.With().Logger()
	u, err := url.Parse("http://127.0.0.1:1")
	require.NoError(t, err)

	c := &client{
		piHole:     model.PiHole{URL: u, Password: "test"},
		logger:     &logger,
		httpClient: &http.Client{},
	}

	require.NoError(t, c.DeleteSession())
}
