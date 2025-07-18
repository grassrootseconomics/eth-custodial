package ensclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	registerAPIPath = "/api/v1/internal/register"

	SuffixENSName = ".sarafu.eth"
)

type (
	EnsClient struct {
		apiKey     string
		endpoint   string
		httpClient *http.Client
	}

	RegisterInput struct {
		Address string `json:"address"`
		Hint    string `json:"hint"`
	}

	RegisterResult struct {
		Address    string `json:"address"`
		AutoChoose bool   `json:"autoChoose"`
		Name       string `json:"name"`
	}

	RegisterResponse struct {
		Ok          bool           `json:"ok"`
		Description string         `json:"description"`
		Result      RegisterResult `json:"result"`
	}
)

func New(apiKey string, endpoint string) *EnsClient {
	return &EnsClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
		endpoint: endpoint,
	}
}

func (ec *EnsClient) SetHTTPClient(httpClient *http.Client) *EnsClient {
	ec.httpClient = httpClient
	return ec
}

func (ec *EnsClient) setDefaultHeaders(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer "+ec.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return req
}

func (ec *EnsClient) postRequestWithCtx(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	return ec.do(req)
}

func (ec *EnsClient) do(req *http.Request) (*http.Response, error) {
	return ec.httpClient.Do(ec.setDefaultHeaders(req))
}

func parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("ENS server error: code=%s: response_body=%s", resp.Status, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

func (ec *EnsClient) Register(ctx context.Context, input RegisterInput) (RegisterResponse, error) {
	var (
		buf              bytes.Buffer
		registerResponse RegisterResponse
	)

	if err := json.NewEncoder(&buf).Encode(input); err != nil {
		return registerResponse, err
	}

	resp, err := ec.postRequestWithCtx(ctx, ec.endpoint+registerAPIPath, &buf)
	if err != nil {
		return registerResponse, err
	}

	if err := parseResponse(resp, &registerResponse); err != nil {
		return registerResponse, err
	}

	return registerResponse, nil
}
