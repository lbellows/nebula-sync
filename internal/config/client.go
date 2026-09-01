package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Client struct {
	SkipTLSVerification bool  `default:"false" envconfig:"CLIENT_SKIP_TLS_VERIFICATION"`
	RetryDelay          int64 `default:"1"     envconfig:"CLIENT_RETRY_DELAY_SECONDS"`
	Timeout             int64 `default:"20"    envconfig:"CLIENT_TIMEOUT_SECONDS"`
	// LongTimeout covers requests that block until Pi-hole finishes work:
	// running gravity and importing/exporting a teleporter archive. Timeout is a
	// whole-request budget, so reusing the 20s default for these guarantees a
	// timeout on any non-trivial blocklist, and the retry then re-runs gravity.
	LongTimeout int64 `default:"300" envconfig:"CLIENT_LONG_TIMEOUT_SECONDS"`
}

func (c *Config) loadClient() error {
	client := Client{}

	if err := envconfig.Process("", &client); err != nil {
		return fmt.Errorf("client env vars: %w", err)
	}

	c.Client = &client

	return nil
}

// HTTPClients are the clients used to talk to Pi-hole. They share a transport,
// and therefore one connection pool, and differ only in their timeout.
type HTTPClients struct {
	// Standard serves ordinary API calls, where a short timeout is what makes an
	// unreachable target fail fast.
	Standard *http.Client
	// Long serves calls that block until Pi-hole has finished working: running
	// gravity, and teleporter import/export.
	Long *http.Client
}

func (c *Client) NewHTTPClients() HTTPClients {
	transport := cloneDefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: c.SkipTLSVerification,
	}

	return HTTPClients{
		Standard: &http.Client{
			Timeout:   time.Duration(c.Timeout) * time.Second,
			Transport: transport,
		},
		Long: &http.Client{
			Timeout:   time.Duration(c.LongTimeout) * time.Second,
			Transport: transport,
		},
	}
}

func cloneDefaultTransport() *http.Transport {
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		return base.Clone()
	}
	return &http.Transport{Proxy: http.ProxyFromEnvironment}
}

func (c *Client) String() string {
	return fmt.Sprintf("%+v", *c)
}
