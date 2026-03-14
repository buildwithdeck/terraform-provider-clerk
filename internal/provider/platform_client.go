package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const platformBaseURL = "https://api.clerk.com/v1"

// PlatformClient is a minimal HTTP client for Clerk's Platform API (beta).
type PlatformClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewPlatformClient creates a PlatformClient with sensible defaults.
func NewPlatformClient(apiKey string) *PlatformClient {
	return &PlatformClient{
		BaseURL:    platformBaseURL,
		APIKey:     apiKey,
		HTTPClient: http.DefaultClient,
	}
}

// --- Request / Response types ---

type CreateApplicationParams struct {
	Name             string   `json:"name"`
	Domain           string   `json:"domain,omitempty"`
	ProxyPath        string   `json:"proxy_path,omitempty"`
	EnvironmentTypes []string `json:"environment_types,omitempty"`
	Template         string   `json:"template,omitempty"`
}

type UpdateApplicationParams struct {
	Name string `json:"name"`
}

type PlatformApplicationInstance struct {
	InstanceID      string `json:"instance_id"`
	EnvironmentType string `json:"environment_type"`
	SecretKey       string `json:"secret_key,omitempty"`
	PublishableKey  string `json:"publishable_key,omitempty"`
}

type PlatformApplicationResponse struct {
	ApplicationID string                        `json:"application_id"`
	Instances     []PlatformApplicationInstance `json:"instances"`
}

type PlatformDeletedObjectResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type PlatformAPIError struct {
	StatusCode int
	Body       string
}

func (e *PlatformAPIError) Error() string {
	return fmt.Sprintf("Platform API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// --- HTTP helpers ---

func (c *PlatformClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &PlatformAPIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshalling response: %w", err)
		}
	}

	return nil
}

// --- Application CRUD ---

func (c *PlatformClient) CreateApplication(ctx context.Context, params *CreateApplicationParams) (*PlatformApplicationResponse, error) {
	var result PlatformApplicationResponse
	err := c.doRequest(ctx, http.MethodPost, "/platform/applications", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PlatformClient) GetApplication(ctx context.Context, id string) (*PlatformApplicationResponse, error) {
	var result PlatformApplicationResponse
	err := c.doRequest(ctx, http.MethodGet, "/platform/applications/"+id+"?include_secret_keys=true", nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PlatformClient) UpdateApplication(ctx context.Context, id string, params *UpdateApplicationParams) (*PlatformApplicationResponse, error) {
	var result PlatformApplicationResponse
	err := c.doRequest(ctx, http.MethodPatch, "/platform/applications/"+id, params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PlatformClient) DeleteApplication(ctx context.Context, id string) (*PlatformDeletedObjectResponse, error) {
	var result PlatformDeletedObjectResponse
	err := c.doRequest(ctx, http.MethodDelete, "/platform/applications/"+id, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// --- Domain types ---

type CreateDomainParams struct {
	Name      string `json:"name"`
	ProxyPath string `json:"proxy_path,omitempty"`
}

type PlatformDomainCNAMETarget struct {
	Host     string `json:"host"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
}

type PlatformDomainResponse struct {
	ID                string                      `json:"id"`
	Name              string                      `json:"name"`
	IsSatellite       bool                        `json:"is_satellite"`
	IsProviderDomain  bool                        `json:"is_provider_domain"`
	FrontendAPIURL    string                      `json:"frontend_api_url"`
	DevelopmentOrigin string                      `json:"development_origin"`
	AccountsPortalURL string                      `json:"accounts_portal_url"`
	ProxyURL          string                      `json:"proxy_url"`
	CNAMETargets      []PlatformDomainCNAMETarget `json:"cname_targets"`
}

// --- Domain CRUD ---

func (c *PlatformClient) CreateDomain(ctx context.Context, appID string, params *CreateDomainParams) (*PlatformDomainResponse, error) {
	var result PlatformDomainResponse
	err := c.doRequest(ctx, http.MethodPost, "/platform/applications/"+appID+"/domains", params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PlatformClient) GetDomain(ctx context.Context, appID, domainID string) (*PlatformDomainResponse, error) {
	var result PlatformDomainResponse
	err := c.doRequest(ctx, http.MethodGet, "/platform/applications/"+appID+"/domains/"+domainID, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *PlatformClient) DeleteDomain(ctx context.Context, appID, domainID string) (*PlatformDeletedObjectResponse, error) {
	var result PlatformDeletedObjectResponse
	err := c.doRequest(ctx, http.MethodDelete, "/platform/applications/"+appID+"/domains/"+domainID, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
