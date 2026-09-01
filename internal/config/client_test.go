package config

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_LoadClient(t *testing.T) {
	conf := Config{}

	t.Setenv("CLIENT_SKIP_TLS_VERIFICATION", "true")
	t.Setenv("CLIENT_TIMEOUT_SECONDS", "45")
	t.Setenv("CLIENT_RETRY_DELAY_SECONDS", "5")

	err := conf.loadClient()
	require.NoError(t, err)

	assert.True(t, conf.Client.SkipTLSVerification)
	assert.Equal(t, int64(45), conf.Client.Timeout)
	assert.Equal(t, int64(5), conf.Client.RetryDelay)
}

func TestClient_NewHTTPClients_TLSMinVersion(t *testing.T) {
	client := &Client{SkipTLSVerification: false, Timeout: 20, LongTimeout: 300}

	clients := client.NewHTTPClients()
	transport, ok := clients.Standard.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
	assert.NotNil(t, transport.Proxy)

	// The long client only differs by timeout, and shares the connection pool.
	assert.Equal(t, 20*time.Second, clients.Standard.Timeout)
	assert.Equal(t, 300*time.Second, clients.Long.Timeout)
	assert.Same(t, clients.Standard.Transport, clients.Long.Transport)
}

func TestConfig_LoadClient_LongTimeoutDefault(t *testing.T) {
	conf := Config{}
	require.NoError(t, conf.loadClient())

	assert.Equal(t, int64(20), conf.Client.Timeout)
	assert.Equal(t, int64(300), conf.Client.LongTimeout)
}
