package local

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"qsc/tools"

	u "github.com/quollix/common/utils"
)

const (
	dockerHubBaseURL         = "https://registry.hub.docker.com"
	dockerHubRegistryBaseURL = "https://registry-1.docker.io"
	dockerHubLoginURL        = "https://hub.docker.com/v2/users/login/"
)

type dockerHubTagsResponse struct {
	Results []dockerHubTagResult `json:"results"`
	Next    string               `json:"next"`
}

type dockerHubTagResult struct {
	Name string `json:"name"`
}

type DockerHubRegistry interface {
	FetchTags(repoPath string) ([]string, error)
	FetchDigest(repoPath, tag string) (string, error)
}

type DockerHubAuthProvider interface {
	GetDockerHubAuth() (*tools.DockerHubAuth, error)
}

type DockerHubRegistryImpl struct {
	HTTPClient            *http.Client
	DockerHubAuthProvider DockerHubAuthProvider
	BaseURL               string
	RegistryBaseURL       string
	LoginURL              string
	apiToken              string
}

func (d *DockerHubRegistryImpl) FetchTags(repoPath string) ([]string, error) {
	return fetchAllPaginatedTags(d.buildTagsURL(repoPath), d.fetchTagPage)
}

func (d *DockerHubRegistryImpl) FetchDigest(repoPath, tag string) (string, error) {
	bearerToken := ""
	return fetchManifestDigest(
		d.buildManifestURL(repoPath, tag),
		&bearerToken,
		d.httpClient(),
		func(method, endpoint, bearerToken string) (*http.Response, error) {
			return newRegistryRequest(d.httpClient(), method, endpoint, bearerToken)
		},
	)
}

func (d *DockerHubRegistryImpl) buildTagsURL(repoPath string) string {
	baseURL := d.BaseURL
	if baseURL == "" {
		baseURL = dockerHubBaseURL
	}
	return fmt.Sprintf("%s/v2/repositories/%s/tags?page_size=100", strings.TrimRight(baseURL, "/"), repoPath)
}

func (d *DockerHubRegistryImpl) buildManifestURL(repoPath, tag string) string {
	baseURL := d.RegistryBaseURL
	if baseURL == "" {
		baseURL = dockerHubRegistryBaseURL
	}
	return fmt.Sprintf("%s/v2/%s/manifests/%s", strings.TrimRight(baseURL, "/"), repoPath, tag)
}

func (d *DockerHubRegistryImpl) fetchTagPage(pageURL string) ([]string, string, error) {
	response, err := d.getTagsResponse(pageURL)
	if err != nil {
		return nil, "", err
	}
	return extractDockerHubTags(response.Results), response.Next, nil
}

func (d *DockerHubRegistryImpl) getTagsResponse(pageURL string) (dockerHubTagsResponse, error) {
	resp, err := d.getTagPage(pageURL)
	if err != nil {
		return dockerHubTagsResponse{}, err
	}
	defer u.Close(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return dockerHubTagsResponse{}, readUnexpectedStatusError(resp)
	}

	var response dockerHubTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return dockerHubTagsResponse{}, u.Logger.NewError(err.Error(), tools.UrlField, pageURL)
	}
	return response, nil
}

func (d *DockerHubRegistryImpl) getTagPage(pageURL string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), tools.UrlField, pageURL)
	}
	authToken, err := d.getDockerHubAPIToken()
	if err != nil {
		return nil, err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "JWT "+authToken)
	}
	resp, err := d.httpClient().Do(req)
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), tools.UrlField, pageURL)
	}
	return resp, nil
}

func (d *DockerHubRegistryImpl) getDockerHubAPIToken() (string, error) {
	if d.apiToken != "" {
		return d.apiToken, nil
	}
	if d.DockerHubAuthProvider == nil {
		return "", nil
	}
	config, err := d.DockerHubAuthProvider.GetDockerHubAuth()
	if err != nil {
		return "", err
	}
	if config == nil {
		return "", nil
	}
	apiToken, err := d.loginToOfficialDockerHub(config)
	if err != nil {
		return "", err
	}
	d.apiToken = apiToken
	return d.apiToken, nil
}

func (d *DockerHubRegistryImpl) loginToOfficialDockerHub(config *tools.DockerHubAuth) (string, error) {
	request := map[string]string{
		"username": config.Username,
		"password": config.Token,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", u.Logger.NewError(err.Error())
	}
	resp, err := d.httpClient().Post(d.loginURL(), "application/json", bytes.NewReader(body))
	if err != nil {
		return "", u.Logger.NewError(err.Error(), tools.UrlField, d.loginURL())
	}
	defer u.Close(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", readUnexpectedStatusError(resp)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", u.Logger.NewError(err.Error(), tools.UrlField, d.loginURL())
	}
	if out.Token == "" {
		return "", u.Logger.NewError("Docker Hub login response missing token")
	}
	return out.Token, nil
}

func (d *DockerHubRegistryImpl) loginURL() string {
	if d.LoginURL != "" {
		return d.LoginURL
	}
	return dockerHubLoginURL
}

func extractDockerHubTags(results []dockerHubTagResult) []string {
	tags := make([]string, 0, len(results))
	for _, result := range results {
		tags = append(tags, result.Name)
	}
	return tags
}

func (d *DockerHubRegistryImpl) httpClient() *http.Client {
	if d.HTTPClient != nil {
		return d.HTTPClient
	}
	return defaultRegistryHTTPClient()
}
