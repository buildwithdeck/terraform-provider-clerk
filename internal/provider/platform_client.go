package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

func (c *PlatformClient) doMultipartUpload(ctx context.Context, method, path, fieldName string, fileContent []byte, result interface{}) error {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile(fieldName, "upload")
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}
	if _, err := part.Write(fileContent); err != nil {
		return fmt.Errorf("writing file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

// --- Application Logo/Favicon ---

func (c *PlatformClient) UploadLogo(ctx context.Context, appID string, fileContent []byte) error {
	return c.doMultipartUpload(ctx, http.MethodPost,
		"/platform/applications/"+appID+"/logo", "file", fileContent, nil)
}

func (c *PlatformClient) DeleteLogo(ctx context.Context, appID string) error {
	return c.doRequest(ctx, http.MethodDelete,
		"/platform/applications/"+appID+"/logo", nil, nil)
}

func (c *PlatformClient) UploadFavicon(ctx context.Context, appID string, fileContent []byte) error {
	return c.doMultipartUpload(ctx, http.MethodPost,
		"/platform/applications/"+appID+"/favicon", "file", fileContent, nil)
}

func (c *PlatformClient) DeleteFavicon(ctx context.Context, appID string) error {
	return c.doRequest(ctx, http.MethodDelete,
		"/platform/applications/"+appID+"/favicon", nil, nil)
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

// --- Instance Config ---

// GetInstanceConfig retrieves the instance configuration as a dynamic map.
// Returns the config map (without config_version), the config_version string, and any error.
func (c *PlatformClient) GetInstanceConfig(ctx context.Context, appID, instanceID string) (map[string]interface{}, string, error) {
	var raw map[string]interface{}
	path := fmt.Sprintf("/platform/applications/%s/instances/%s/config", appID, instanceID)
	err := c.doRequest(ctx, http.MethodGet, path, nil, &raw)
	if err != nil {
		return nil, "", err
	}

	configVersion := ""
	if v, ok := raw["config_version"]; ok {
		if s, ok := v.(string); ok {
			configVersion = s
		}
		delete(raw, "config_version")
	}

	return raw, configVersion, nil
}

// UpdateInstanceConfig patches the instance configuration with the given key-value pairs.
// Uses destructive=true and optionally sets If-Match header for optimistic concurrency.
// Returns the new config_version and any error.
func (c *PlatformClient) UpdateInstanceConfig(ctx context.Context, appID, instanceID string, config map[string]interface{}) (string, error) {
	path := fmt.Sprintf("/platform/applications/%s/instances/%s/config?destructive=true", appID, instanceID)

	var raw map[string]interface{}
	err := c.doRequest(ctx, http.MethodPatch, path, config, &raw)
	if err != nil {
		return "", err
	}

	newVersion := ""
	if v, ok := raw["config_version"]; ok {
		if s, ok := v.(string); ok {
			newVersion = s
		}
	}

	return newVersion, nil
}
