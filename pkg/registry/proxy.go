package registry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

const (
	// Registry REST routes
	routeFrontendPing     = "/v2/"
	routeDataEndpointPing = "/"
)

// Options configures the proxy.
type Options struct {
	// LoginServer is the registry login server name, such as myregistry.azurecr.io
	LoginServer string

	// DataEndpoint is the registry data endpoint, such as myregistry.southindia.azurecr.io
	DataEndpoint string

	// Username is the registry login username
	Username string

	// Password is the registry login password
	Password string

	// Insecure indicates if registry should be accessed over HTTP
	Insecure bool
}

// Proxy acts as a proxy to a remote registry.
type Proxy struct {
	http.RoundTripper
	*Options
	zerolog.Logger
}

// NewProxy creates a new registry proxy.
func NewProxy(rt http.RoundTripper, opts *Options, logger zerolog.Logger) (*Proxy, error) {
	if opts == nil {
		return nil, errors.New("opts required")
	}

	if opts.LoginServer == "" {
		return nil, errors.New("login server name required")
	}

	if rt == nil {
		rt = http.DefaultTransport
	}
	return &Proxy{
		RoundTripper: rt,
		Options:      opts,
		Logger:       logger,
	}, nil
}

// Ping pings various registry endpoints.
func (p Proxy) Ping() (err error) {
	url := fmt.Sprintf("%s://%s%s", p.scheme(), p.LoginServer, routeFrontendPing)

	if err = p.doNoAuth(url, http.StatusUnauthorized); err != nil {
		return err
	}

	if err = p.doBasicAuth(url, http.StatusOK); err != nil {
		return err
	}

	if err = p.doBearerAuth(url, http.StatusOK); err != nil {
		return err
	}

	if p.DataEndpoint != "" {
		url := fmt.Sprintf("%s://%s%s", p.scheme(), p.DataEndpoint, routeDataEndpointPing)
		return p.doNoAuth(url, http.StatusForbidden)
	}

	return nil
}

func (p Proxy) scheme() string {
	scheme := "https"
	if p.Insecure {
		scheme = "http"
	}
	return scheme
}

func (p Proxy) doNoAuth(url string, expected int) error {
	return do(url, newNoAuthTransport(p.RoundTripper, p.Logger), expected)
}

func (p Proxy) doBasicAuth(url string, expected int) error {
	return do(url, newBasicAuthTransport(p.RoundTripper, p.Username, p.Password, p.Logger), expected)
}

func (p Proxy) doBearerAuth(url string, expected int) error {
	return do(url, newBearerAuthTransport(p.RoundTripper, p.Username, p.Password, p.Logger), expected)
}

func do(url string, t transport, expected int) error {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return err
	}

	result, err := t.RoundTrip(req)
	if result.Response.Code != expected {
		return fmt.Errorf("invalid response code, expected: %v, got: %v", expected, result.Response.Code)
	}

	return nil
}
