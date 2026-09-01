package pihole

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/lovelaze/nebula-sync/internal/pihole/model"
	"github.com/lovelaze/nebula-sync/version"
)

var userAgent = fmt.Sprintf("nebula-sync/%s", version.Version)

type Client interface {
	PostAuth() error
	DeleteSession() error
	GetTeleporter() ([]byte, error)
	PostTeleporter(payload []byte, teleporterRequest *model.PostTeleporterRequest) error
	GetConfig() (configResponse *model.ConfigResponse, err error)
	PatchConfig(patchRequest *model.PatchConfigRequest) error
	PostRunGravity() error
	String() string
	APIPath(target string) string
}

// NewClient takes two http clients: httpClient for ordinary API calls and
// longHTTPClient, with a larger timeout, for the calls that block until Pi-hole
// has finished working (gravity and teleporter).
func NewClient(piHole model.PiHole, httpClient, longHTTPClient *http.Client) Client {
	logger := log.With().Str("client", piHole.URL.String()).Logger()
	return &client{
		piHole:         piHole,
		logger:         &logger,
		httpClient:     httpClient,
		longHTTPClient: longHTTPClient,
	}
}

type client struct {
	piHole         model.PiHole
	auth           auth
	logger         *zerolog.Logger
	httpClient     *http.Client
	longHTTPClient *http.Client
}

type auth struct {
	sid      string
	csrf     string
	validity int
	valid    bool
}

func (a *auth) verify() error {
	if !a.valid {
		return errors.New("invalid sid found")
	}

	return nil
}

func (client *client) PostAuth() error {
	client.logger.Debug().Msg("PostAuth")
	authResponse := model.AuthResponse{}

	reqBytes, err := json.Marshal(model.AuthRequest{Password: client.piHole.Password}) //nolint:gosec // mirror API model
	if err != nil {
		return client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		client.APIPath("/auth"),
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return client.wrapError(err, req)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	response, err := client.httpClient.Do(req)
	if err != nil {
		return client.wrapError(err, req)
	}
	defer response.Body.Close()

	body, err := readHTTPBody(response)
	if err != nil {
		return client.wrapError(err, req)
	}

	if err = json.Unmarshal(body, &authResponse); err != nil {
		return client.wrapError(err, req)
	}

	client.auth = auth{
		sid:      authResponse.Session.Sid,
		csrf:     authResponse.Session.Csrf,
		validity: authResponse.Session.Validity,
		valid:    authResponse.Session.Valid,
	}

	return client.auth.verify()
}

func (client *client) DeleteSession() error {
	client.logger.Debug().Msg("Delete session")

	// Check for a session before validating it. A client that never
	// authenticated has nothing to tear down, and returning an error here would
	// burn the whole retry budget on a call that can never succeed.
	if client.auth.sid == "" {
		log.Debug().Msg("Trying to delete empty session")
		return nil
	}

	if err := client.auth.verify(); err != nil {
		return client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, client.APIPath("auth"), nil)
	if err != nil {
		return client.wrapError(err, req)
	}

	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("User-Agent", userAgent)

	response, err := client.httpClient.Do(req)
	if err != nil {
		return client.wrapError(err, req)
	}
	defer response.Body.Close()

	_, err = readHTTPBody(response)
	return client.wrapError(err, req)
}

func (client *client) GetTeleporter() ([]byte, error) {
	client.logger.Debug().Msg("Get teleporter")
	if err := client.auth.verify(); err != nil {
		return nil, client.wrapError(err, nil)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.APIPath("teleporter"), nil)
	if err != nil {
		return nil, client.wrapError(err, req)
	}
	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("User-Agent", userAgent)

	response, err := client.longHTTPClient.Do(req)
	if err != nil {
		return nil, client.wrapError(err, req)
	}
	defer response.Body.Close()

	body, err := readHTTPBody(response)
	return body, client.wrapError(err, req)
}

func (client *client) PostTeleporter(payload []byte, teleporterRequest *model.PostTeleporterRequest) error {
	client.logger.Debug().Any("payload", teleporterRequest).Msg("Post teleporter")

	if err := client.auth.verify(); err != nil {
		return client.wrapError(err, nil)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fileWriter, _ := writer.CreateFormFile("file", "config.zip")
	if _, err := io.Copy(fileWriter, bytes.NewReader(payload)); err != nil {
		return client.wrapError(err, nil)
	}

	if teleporterRequest != nil {
		jsonData, err := json.Marshal(teleporterRequest)
		if err != nil {
			return client.wrapError(err, nil)
		}
		if err = writer.WriteField("import", string(jsonData)); err != nil {
			return client.wrapError(err, nil)
		}
	}

	if err := writer.Close(); err != nil {
		return client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, client.APIPath("teleporter"), &requestBody)
	if err != nil {
		return client.wrapError(err, req)
	}
	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", userAgent)

	response, err := client.longHTTPClient.Do(req)
	if err != nil {
		return client.wrapError(err, req)
	}
	defer response.Body.Close()

	_, err = readHTTPBody(response)
	return client.wrapError(err, req)
}

func (client *client) GetConfig() (*model.ConfigResponse, error) {
	var configResponse model.ConfigResponse
	client.logger.Debug().Msg("Get config")
	if err := client.auth.verify(); err != nil {
		return &configResponse, client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.APIPath("config"), nil)
	if err != nil {
		return &configResponse, client.wrapError(err, req)
	}
	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("User-Agent", userAgent)

	response, err := client.httpClient.Do(req)
	if err != nil {
		return &configResponse, client.wrapError(err, req)
	}
	defer response.Body.Close()

	body, err := readHTTPBody(response)
	if err != nil {
		return &configResponse, client.wrapError(err, req)
	}

	if err := json.Unmarshal(body, &configResponse); err != nil {
		return &configResponse, client.wrapError(err, req)
	}

	return &configResponse, client.wrapError(err, req)
}

func (client *client) PatchConfig(patchRequest *model.PatchConfigRequest) error {
	client.logger.Debug().Any("payload", patchRequest).Msgf("Patch config")
	if err := client.auth.verify(); err != nil {
		return client.wrapError(err, nil)
	}

	reqBytes, err := json.Marshal(patchRequest)
	if err != nil {
		return client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPatch,
		client.APIPath("config"),
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return client.wrapError(err, req)
	}
	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("User-Agent", userAgent)

	response, err := client.httpClient.Do(req)
	if err != nil {
		return client.wrapError(err, req)
	}
	defer response.Body.Close()

	_, err = readHTTPBody(response)
	return client.wrapError(err, req)
}

func (client *client) PostRunGravity() error {
	client.logger.Debug().Msg("Post run gravity")
	if err := client.auth.verify(); err != nil {
		return client.wrapError(err, nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, client.APIPath("action/gravity"), nil)
	if err != nil {
		return client.wrapError(err, req)
	}
	req.Header.Set("Sid", client.auth.sid)
	req.Header.Set("User-Agent", userAgent)

	response, err := client.longHTTPClient.Do(req)
	if err != nil {
		return client.wrapError(err, req)
	}
	defer response.Body.Close()

	_, err = readHTTPBody(response)
	return client.wrapError(err, req)
}

func (client *client) String() string {
	return client.piHole.URL.String()
}

func (client *client) APIPath(target string) string {
	return client.piHole.URL.JoinPath("api", target).String()
}

func (client *client) wrapError(err error, req *http.Request) error {
	if err != nil {
		if req != nil {
			return fmt.Errorf("%s: %w", req.URL.String(), err)
		}
		return fmt.Errorf("%s: %w", client.String(), err)
	}
	return nil
}

func readHTTPBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if err := successfulHTTPStatus(response.StatusCode, body); err != nil {
		return nil, err
	}
	return body, nil
}

func successfulHTTPStatus(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode <= 299 {
		return nil
	}

	const maxBody = 1024
	msg := string(body)
	if len(msg) > maxBody {
		msg = msg[:maxBody] + "..."
	}

	return fmt.Errorf("unexpected status code: %d, response body: %s", statusCode, msg)
}
