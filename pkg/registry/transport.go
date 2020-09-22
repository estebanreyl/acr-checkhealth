package registry

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aviral26/acr/runner/pkg/stats"
)

const (
	headerChallenge = "Www-Authenticate"
)

type roundTripStats struct {
	URL     string `json:"url"`
	Code    int    `json:"code"`
	Elapsed string `json:"elapsed"`
	Error   error  `json:"error"`
}

type transport struct {
	rt       http.RoundTripper
	username string
	password string
	obs      *stats.Observations
}

func newTransport(rt http.RoundTripper, username, password string) transport {
	return transport{
		rt:       rt,
		username: username,
		password: password,
		obs:      &stats.Observations{},
	}
}

func (t transport) RoundTrip(req *http.Request) (bodyString string, err error) {
	clonedReq := req.Clone(req.Context())
	resp, err := t.roundTrip(clonedReq)
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		resp.Body.Close()
		scheme, params := parseAuthHeader(resp.Header.Get(headerChallenge))
		if scheme == "bearer" {
			token, _, err := t.fetchToken(params)
			if err != nil {
				return "", err
			}

			clonedReq := req.Clone(req.Context())
			clonedReq.Header.Set("Authorization", "Bearer "+token)
			resp, err = t.roundTrip(clonedReq)
		}
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func (t transport) fetchToken(params map[string]string) (string, *http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, params["realm"], nil)
	if err != nil {
		return "", nil, err
	}
	if t.username != "" {
		req.SetBasicAuth(t.username, t.password)
	}

	query := url.Values{}
	if service, ok := params["service"]; ok {
		query.Set("service", service)
	}
	if scope, ok := params["scope"]; ok {
		query.Set("scope", scope)
	}
	req.URL.RawQuery = query.Encode()

	resp, err := t.roundTrip(req)
	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", resp, errors.New("get access token failed")
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}
	return result.AccessToken, nil, nil
}

func (t transport) roundTrip(req *http.Request) (*http.Response, error) {
	stats := roundTripStats{
		URL: req.URL.String(),
	}
	start := time.Now()
	defer func() {
		stats.Elapsed = time.Since(start).String()
		t.obs.Record(start.Local().String(), stats)
	}()

	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		stats.Error = err
		return resp, err
	}

	stats.Code = resp.StatusCode

	return resp, nil
}

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
