package registry

import (
	"errors"
	"fmt"
	"net/http"

	rhttp "github.com/aviral26/acr-checkhealth/pkg/http"

	"github.com/rs/zerolog"
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
	rhttp.RoundTripper
	*Options
	zerolog.Logger
}

// NewProxy creates a new registry proxy.
func NewProxy(tripper http.RoundTripper, opts *Options, logger zerolog.Logger) (*Proxy, error) {
	if opts == nil {
		return nil, errors.New("opts required")
	}

	if opts.LoginServer == "" {
		return nil, errors.New("login server name required")
	}

	if tripper == nil {
		return nil, errors.New("round tripper required")
	}
	return &Proxy{
		RoundTripper: rhttp.RoundTripperWithContext{Logger: logger, Base: tripper},
		Options:      opts,
		Logger:       logger,
	}, nil
}

// Ping pings various registry endpoints with different auth modes.
func (p Proxy) Ping() (err error) {
	p.Logger.Info().Msg("pinging frontend")
	url := p.url(p.LoginServer, routeFrontendPing)
	regReq := registryRequest{
		method: http.MethodGet,
		url:    url,
	}

	if _, err = p.do(regReq, http.StatusUnauthorized, noAuth); err != nil {
		return err
	}

	if p.Username != "" {
		if _, err = p.do(regReq, http.StatusOK, basicAuth); err != nil {
			return err
		}

		if _, err = p.do(regReq, http.StatusOK, bearerAuth); err != nil {
			return err
		}
	}

	if p.DataEndpoint != "" {
		p.Logger.Info().Msg("pinging data proxy")
		regReq := registryRequest{
			method: http.MethodGet,
			url:    p.url(p.DataEndpoint, routeDataEndpointPing),
		}

		if _, err := p.do(regReq, http.StatusForbidden, noAuth); err != nil {
			return err
		}
	}

	return nil
}

// CheckHealth the health of core registry APIs.
func (p Proxy) CheckHealth() error {
	return p.oci()
}

// scheme determines the HTTP scheme of the request url.
func (p Proxy) url(hostname, route string) string {
	scheme := "https"
	if p.Insecure {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, hostname, route)
}

func (p Proxy) do(regReq registryRequest, expected int, at authType) (tripInfo rhttp.RoundTripInfo, err error) {
	var t transport
	switch at {
	case noAuth:
		t, err = newNoAuthTransport(p.RoundTripper, p.Logger)
		if err != nil {
			return tripInfo, err
		}

	case basicAuth:
		t, err = newBasicAuthTransport(p.RoundTripper, p.Username, p.Password, p.Logger)
		if err != nil {
			return tripInfo, err
		}
	case bearerAuth:
		t, err = newBearerAuthTransport(p.RoundTripper, p.Username, p.Password, p.Logger)
		if err != nil {
			return tripInfo, err
		}
	default:
		return tripInfo, fmt.Errorf("unknown auth type: %v", at)
	}

	result, err := t.RoundTrip(regReq)
	if err != nil {
		return result, err
	}
	if result.Response.Code != expected {
		return result, fmt.Errorf("invalid response code, expected: %v, got: %v", expected, result.Response.Code)
	}

	return result, nil
}
