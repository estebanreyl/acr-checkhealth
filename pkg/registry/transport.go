package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	reghttp "github.com/aviral26/acr/conformance/pkg/http"
	"github.com/rs/zerolog"
)

// authType is an authorization type to use for an HTTP request.
type authType int

// The different kinds of auth mechanisms supported by transport.
const (
	noAuth authType = iota
	basicAuth
	bearerAuth
)

const (
	headerChallenge     = "Www-Authenticate"
	headerLocation      = "Location"
	headerAuthorization = "Authorization"

	schemeBearer = "bearer"

	claimRealm   = "realm"
	claimService = "service"
	claimScope   = "scope"
)

var authHeaderRegex = regexp.MustCompile(`(realm|service|scope)="([^"]*)`)

// transport can be used to make HTTP requests with authentication.
// Basic and bearer auth are supported.
type transport struct {
	rt http.RoundTripper
	authType
	username string
	password string
	logger   zerolog.Logger
}

// newNoAuthTransport returns a new transport that does not use auth.
func newNoAuthTransport(rt http.RoundTripper, logger zerolog.Logger) transport {
	return transport{
		rt:       rt,
		authType: noAuth,
		logger:   logger,
	}
}

// newBasicAuthTransport returns a new transport that uses basic auth.
func newBasicAuthTransport(rt http.RoundTripper, username, password string, logger zerolog.Logger) transport {
	return transport{
		rt:       rt,
		authType: basicAuth,
		username: username,
		password: password,
		logger:   logger,
	}
}

// newBearerAuthTransport returns a new transport that uses bearer auth.
func newBearerAuthTransport(rt http.RoundTripper, username, password string, logger zerolog.Logger) transport {
	return transport{
		rt:       rt,
		authType: bearerAuth,
		username: username,
		password: password,
		logger:   logger,
	}
}

// RoundTrip makes an HTTP request and returns the response body.
// It supports basic and bearer authorization.
func (t transport) RoundTrip(originalReq *http.Request) (rtInfo reghttp.RoundTripInfo, err error) {
	req := originalReq.Clone(originalReq.Context())

	switch t.authType {
	case noAuth, bearerAuth:
	case basicAuth:
		if t.username == "" {
			return rtInfo, errors.New("username not provided")
		}
		req.SetBasicAuth(t.username, t.password)
	}

	rtInfo = t.roundTrip(req)
	if rtInfo.Error != nil {
		return rtInfo, rtInfo.Error
	}

	if t.authType == bearerAuth && rtInfo.Response.Code == http.StatusUnauthorized {
		scheme, params := parseAuthHeader(rtInfo.Response.HeaderChallenge)
		if scheme == schemeBearer {
			token, err := t.getToken(params)
			if err != nil {
				return rtInfo, err
			}

			req := originalReq.Clone(originalReq.Context())
			req.Header.Set(headerAuthorization, "Bearer "+token)
			rtInfo = t.roundTrip(req)
		}
	}

	return rtInfo, nil
}

// getToken attempts to get an auth token based on the given params.
// The params specify:
// - realm: the HTTP endpoint of the token server
// - service: the service to obtain the token for, such as myregistry.azurecr.io
// - scope: the authorization scope the token grants
func (t transport) getToken(params map[string]string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, params[claimRealm], nil)
	if err != nil {
		return "", err
	}
	if t.username != "" {
		req.SetBasicAuth(t.username, t.password)
	}

	query := url.Values{}
	if service, ok := params[claimService]; ok {
		query.Set(claimService, service)
	}
	if scope, ok := params[claimScope]; ok {
		query.Set(claimScope, scope)
	}
	req.URL.RawQuery = query.Encode()

	rtInfo := t.roundTrip(req)
	if rtInfo.Error != nil {
		return "", rtInfo.Error
	}
	if rtInfo.Response.Code != http.StatusOK {
		return "", fmt.Errorf("get access token failed, expected: 200, got: %v", rtInfo.Response.Code)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(rtInfo.Response.Body), &result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

// roundTrip makes the given request and returns the server response, with some statistics.
func (t transport) roundTrip(req *http.Request) reghttp.RoundTripInfo {
	info := reghttp.RoundTripInfo{
		Request: reghttp.Request{
			URL:                 req.URL,
			StartedAt:           time.Now(),
			HeaderAuthorization: req.Header.Get(headerAuthorization),
		},
	}
	defer func() {
		info.Elapsed = time.Since(info.StartedAt)
		t.logger.Info().Msgf("%+v", info)
	}()

	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		info.Error = err
		return info
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		info.Error = err
		return info
	}

	info.Response = reghttp.Response{
		Code:            resp.StatusCode,
		Body:            string(bodyBytes),
		HeaderChallenge: resp.Header.Get(headerChallenge),
		HeaderLocation:  resp.Header.Get(headerLocation),
	}

	return info
}

// parseAuthHeader parses the Www-Authenticate header and retrieves auth metadata
// that can be used to obtain auth tokens.
func parseAuthHeader(header string) (string, map[string]string) {
	parts := strings.SplitN(header, " ", 2)
	scheme := strings.ToLower(parts[0])
	if len(parts) < 2 {
		return scheme, nil
	}

	params := make(map[string]string)
	result := authHeaderRegex.FindAllStringSubmatch(parts[1], -1)
	for _, match := range result {
		params[strings.ToLower(match[1])] = match[2]
	}

	return scheme, params
}
