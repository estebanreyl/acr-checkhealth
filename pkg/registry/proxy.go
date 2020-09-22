package registry

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
)

var authHeaderRegex = regexp.MustCompile(`(realm|service|scope)="([^"]*)`)

const (
	// Registry REST routes
	routeFrontendPing     = "/v2/"
	routeDataEndpointPing = "/"
	routeTokenServerPing  = "/oauth2/token"
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
	transport transport
	*Options
}

// NewProxy creates a new registry proxy.
func NewProxy(rt http.RoundTripper, opts *Options) (*Proxy, error) {
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
		transport: newTransport(rt, opts.Username, opts.Password),
		Options:   opts,
	}, nil
}

// Logs returns the logs of all operations done on the registry.
func (p Proxy) Logs() (string, error) {
	return p.transport.obs.Marshal()
}

// Ping pings various registry endpoints.
func (p Proxy) Ping() error {
	// Ping the frontend.
	url := fmt.Sprintf("%s://%s%s", p.scheme(), p.LoginServer, routeFrontendPing)
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return err
	}
	_, err = p.transport.RoundTrip(req)

	return err
}

func (p Proxy) scheme() string {
	scheme := "https"
	if p.Insecure {
		scheme = "http"
	}
	return scheme
}
