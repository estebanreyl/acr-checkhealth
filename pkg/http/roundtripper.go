package http

import (
	"net/url"
	"time"
)

// Request represents a request made to the registry.
type Request struct {
	URL                 *url.URL  `json:"url"`
	HeaderAuthorization string    `json:"authorization"`
	Body                string    `json:"body"`
	StartedAt           time.Time `json:"startedAt"`
}

// Response respresents a response received from the registry.
type Response struct {
	Code            int    `json:"code"`
	Body            string `json:"body"`
	HeaderChallenge string `json:"Www-Authenticate"`
	HeaderLocation  string `json:"redirectLocation"`
}

// RoundTripInfo represents information about a round-trip made to the registry.
type RoundTripInfo struct {
	Request  `json:"request"`
	Response `json:"response"`
	Elapsed  time.Duration `json:"elapsed"`
	Error    error         `json:"error"`
}

// RoundTripper provides a means to do an HTTP round trip.
type RoundTripper interface {
	// RoundTrip makes an HTTP request and returns the response with some stats.
	RoundTrip(Request) (RoundTripInfo, error)
}
