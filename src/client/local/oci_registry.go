package local

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	u "github.com/quollix/common/utils"
)

const defaultOCITagPageSize = 100

var ociTagPageSizesByHost = map[string]int{
	"codeberg.org": 1000,
	"ghcr.io":      1000,
	"quay.io":      100,
}

type ociTagsResponse struct {
	Tags []string `json:"tags"`
}

type OCIRegistry interface {
	FetchTags(host, repoPath string) ([]string, error)
	FetchDigest(host, repoPath, tag string) (string, error)
}

type OCIRegistryImpl struct {
	HTTPClient       *http.Client
	BaseURLOverrides map[string]string
}

type authChallenge struct {
	realm   string
	service string
	scope   string
}

func (o *OCIRegistryImpl) FetchTags(host, repoPath string) ([]string, error) {
	registryURL := o.buildTagsURL(host, repoPath)
	bearerToken := ""
	return fetchAllPaginatedTags(registryURL, func(pageURL string) ([]string, string, error) {
		return o.fetchTagPage(pageURL, &bearerToken)
	})
}

func (o *OCIRegistryImpl) FetchDigest(host, repoPath, tag string) (string, error) {
	bearerToken := ""
	return fetchManifestDigest(o.buildManifestURL(host, repoPath, tag), &bearerToken, o.httpClient(), o.newRequest)
}

func (o *OCIRegistryImpl) buildTagsURL(host, repoPath string) string {
	return o.registryBaseURL(host) + "/v2/" + repoPath + "/tags/list?n=" + strconv.Itoa(o.tagPageSize(host))
}

func (o *OCIRegistryImpl) buildManifestURL(host, repoPath, tag string) string {
	return o.registryBaseURL(host) + "/v2/" + repoPath + "/manifests/" + tag
}

func (o *OCIRegistryImpl) fetchTagPage(pageURL string, bearerToken *string) ([]string, string, error) {
	response, nextURL, retryCurrentPage, err := o.getTagsResponse(pageURL, bearerToken)
	if err != nil {
		return nil, "", err
	}
	if retryCurrentPage {
		return nil, pageURL, nil
	}
	return response.Tags, nextURL, nil
}

func (o *OCIRegistryImpl) getTagsResponse(pageURL string, bearerToken *string) (ociTagsResponse, string, bool, error) {
	resp, err := o.newRequest(http.MethodGet, pageURL, *bearerToken)
	if err != nil {
		return ociTagsResponse{}, "", false, err
	}
	defer u.Close(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized && *bearerToken == "" {
		retryCurrentPage, err := handleRegistryAnonymousAuth(resp, bearerToken, o.httpClient())
		return ociTagsResponse{}, "", retryCurrentPage, err
	}
	if resp.StatusCode != http.StatusOK {
		return ociTagsResponse{}, "", false, readUnexpectedStatusError(resp)
	}

	var response ociTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return ociTagsResponse{}, "", false, u.Logger.NewError(err.Error(), "url", pageURL)
	}
	nextURL, err := parseNextLink(resp.Header.Get("Link"), pageURL)
	if err != nil {
		return ociTagsResponse{}, "", false, err
	}
	return response, nextURL, false, nil
}

func parseBearerChallenge(header string) (authChallenge, error) {
	if !strings.HasPrefix(header, "Bearer ") {
		return authChallenge{}, u.Logger.NewError("unsupported registry auth challenge", "challenge", header)
	}
	challenge := authChallenge{}
	for part := range strings.SplitSeq(strings.TrimPrefix(header, "Bearer "), ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"`)
		switch key {
		case "realm":
			challenge.realm = value
		case "service":
			challenge.service = value
		case "scope":
			challenge.scope = value
		}
	}
	return challenge, nil
}

func parseNextLink(header, baseURL string) (string, error) {
	if header == "" || !strings.Contains(header, `rel="next"`) {
		return "", nil
	}
	start := strings.Index(header, "<")
	end := strings.Index(header, ">")
	if start == -1 || end == -1 || end <= start+1 {
		return "", u.Logger.NewError("invalid registry pagination link", "link", header)
	}
	nextRef, err := url.Parse(header[start+1 : end])
	if err != nil {
		return "", u.Logger.NewError(err.Error(), "link", header)
	}
	baseRef, err := url.Parse(baseURL)
	if err != nil {
		return "", u.Logger.NewError(err.Error(), "url", baseURL)
	}
	return baseRef.ResolveReference(nextRef).String(), nil
}

func (o *OCIRegistryImpl) newRequest(method, endpoint, bearerToken string) (*http.Response, error) {
	return newRegistryRequest(o.httpClient(), method, endpoint, bearerToken)
}

func (o *OCIRegistryImpl) registryBaseURL(host string) string {
	if o.BaseURLOverrides != nil {
		if overridden := o.BaseURLOverrides[host]; overridden != "" {
			return strings.TrimRight(overridden, "/")
		}
	}
	return "https://" + host
}

func (o *OCIRegistryImpl) tagPageSize(host string) int {
	if pageSize, ok := ociTagPageSizesByHost[host]; ok {
		return pageSize
	}
	u.Logger.Warn("unknown OCI registry host for tag page size, using conservative default", "host", host, "page_size", defaultOCITagPageSize)
	return defaultOCITagPageSize
}

func (o *OCIRegistryImpl) httpClient() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return defaultRegistryHTTPClient()
}

func readUnexpectedStatusError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return u.Logger.NewError("registry request failed", "status_code", resp.StatusCode)
	}
	return u.Logger.NewError("registry request failed", "status_code", resp.StatusCode, "response_body", string(body))
}

func fetchManifestDigest(endpoint string, bearerToken *string, httpClient *http.Client, newRequest func(method, endpoint, bearerToken string) (*http.Response, error)) (string, error) {
	resp, err := newRequest(http.MethodHead, endpoint, *bearerToken)
	if err != nil {
		return "", err
	}
	defer u.Close(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized && *bearerToken == "" {
		retryCurrentPage, err := handleRegistryAnonymousAuth(resp, bearerToken, httpClient)
		if err != nil {
			return "", err
		}
		if retryCurrentPage {
			return fetchManifestDigest(endpoint, bearerToken, httpClient, newRequest)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return "", readUnexpectedStatusError(resp)
	}
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", u.Logger.NewError("registry manifest response missing digest")
	}
	return digest, nil
}

func handleRegistryAnonymousAuth(resp *http.Response, bearerToken *string, httpClient *http.Client) (bool, error) {
	challenge, err := parseBearerChallenge(resp.Header.Get("Www-Authenticate"))
	if err != nil {
		return false, err
	}
	*bearerToken, err = fetchRegistryBearerToken(httpClient, challenge)
	if err != nil {
		return false, err
	}
	return true, nil
}

func fetchRegistryBearerToken(client *http.Client, challenge authChallenge) (string, error) {
	if challenge.realm == "" {
		return "", u.Logger.NewError("registry auth challenge missing realm")
	}
	params := url.Values{}
	if challenge.service != "" {
		params.Set("service", challenge.service)
	}
	if challenge.scope != "" {
		params.Set("scope", challenge.scope)
	}
	tokenURL := challenge.realm
	if encoded := params.Encode(); encoded != "" {
		tokenURL += "?" + encoded
	}
	resp, err := client.Get(tokenURL)
	if err != nil {
		return "", u.Logger.NewError(err.Error(), "url", tokenURL)
	}
	defer u.Close(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", readUnexpectedStatusError(resp)
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", u.Logger.NewError(err.Error(), "url", tokenURL)
	}
	if out.Token == "" {
		return "", u.Logger.NewError("registry token response missing token")
	}
	return out.Token, nil
}

func newRegistryRequest(client *http.Client, method, endpoint, bearerToken string) (*http.Response, error) {
	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), "method", method, "url", endpoint)
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.docker.distribution.manifest.v2+json",
	}, ", "))
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, u.Logger.NewError(err.Error(), "method", method, "url", endpoint)
	}
	return resp, nil
}

func fetchAllPaginatedTags(initialPageURL string, fetchPage func(pageURL string) ([]string, string, error)) ([]string, error) {
	var tags []string
	pageURL := initialPageURL
	for pageURL != "" {
		pageTags, nextPageURL, err := fetchPage(pageURL)
		if err != nil {
			return nil, err
		}
		tags = append(tags, pageTags...)
		pageURL = nextPageURL
	}
	return tags, nil
}

func defaultRegistryHTTPClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}
