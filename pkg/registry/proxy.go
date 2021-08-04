package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	rhttp "github.com/aviral26/acr-checkhealth/pkg/http"
	"github.com/aviral26/acr-checkhealth/pkg/io"
	v2specs "github.com/opencontainers/artifacts/specs-go"
	v2 "github.com/opencontainers/artifacts/specs-go/v2"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/rs/zerolog"
)

// Registry REST routes
const (
	// Ping routes
	routeFrontendPing     = "/v2/"
	routeDataEndpointPing = "/"

	// Blob routes
	routeInitiateBlobUpload = "/v2/%s/blobs/uploads/" // add repo name
	routeBlobPull           = "/v2/%s/blobs/%s"       // add repo name and digest

	// Manifest routes
	routeManifest = "/v2/%s/manifests/%s" // add repo name and digest/tag
)

// Constants for generated data.
const (
	checkHealthAuthor       = "ACR Check Health"
	checkHealthMediaType    = "application/acr.checkhealth.test"
	checkHealthArtifactType = "application/acr.checkhealth.artifact.test"
	checkHealthLayerFmt     = "Test layer authored by " + checkHealthAuthor + " at %s" // add time
	checkHealthRepoPrefix   = "acrcheckhealth"
)

// Other data.
var (
	ociConfig = v1.Image{
		Author: checkHealthAuthor,
	}
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

	// BasicAuthMode indicates that only basic auth should be used
	BasicAuthMode bool
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

	if _, err = p.roundTrip(regReq, http.StatusUnauthorized, noAuth); err != nil {
		return err
	}

	if p.Username != "" {
		if _, err = p.roundTrip(regReq, http.StatusOK, basicAuth); err != nil {
			return err
		}

		if !p.BasicAuthMode {
			if _, err = p.roundTrip(regReq, http.StatusOK, bearerAuth); err != nil {
				return err
			}
		}
	}

	if p.DataEndpoint != "" {
		p.Logger.Info().Msg("pinging data proxy")
		regReq := registryRequest{
			method: http.MethodGet,
			url:    p.url(p.DataEndpoint, routeDataEndpointPing),
		}

		if _, err := p.roundTrip(regReq, http.StatusForbidden, noAuth); err != nil {
			return err
		}
	}

	p.Logger.Info().Msg("ping was successful")

	return nil
}

// CheckHealth checks the health of core registry APIs.
func (p Proxy) CheckHealth() error {
	var (
		repo = fmt.Sprintf("%v%v", checkHealthRepoPrefix, time.Now().Unix())
		tag  = fmt.Sprintf("%v", time.Now().Unix())
	)

	// Push simple image
	desc, err := p.pushOCIImage(repo, tag)
	if err != nil {
		return err
	}

	// Pull image
	err = p.pullOCIImage(repo, tag, desc)
	if err != nil {
		return err
	}

	p.Logger.Info().Msg("check-health was successful")

	return nil
}

// CheckReferrers checks the registry's referrer APIs.
func (p Proxy) CheckReferrers() error {
	var (
		repo        = fmt.Sprintf("%v%v", checkHealthRepoPrefix, time.Now().Unix())
		imageTag    = fmt.Sprintf("%v", time.Now().Unix())
		artifactTag = fmt.Sprintf("%v-art-%v", imageTag, time.Now().Unix())
	)

	// Push simple image
	imageDesc, err := p.pushOCIImage(repo, imageTag)
	if err != nil {
		return err
	}

	// Push artifact layer
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(fmt.Sprintf(checkHealthLayerFmt, time.Now()))))
	if err != nil {
		return err
	}

	artifact := v2.Artifact{
		Versioned: v2specs.Versioned{SchemaVersion: 3},
		Blobs: []v2.Descriptor{
			{
				MediaType: layerDesc.MediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
		MediaType:    v2.MediaTypeArtifactManifest,
		ArtifactType: checkHealthArtifactType,
		SubjectManifest: v2.Descriptor{
			MediaType: imageDesc.MediaType,
			Digest:    imageDesc.Digest,
			Size:      imageDesc.Size,
		},
	}

	artifactBytes, err := json.Marshal(artifact)
	if err != nil {
		return err
	}

	p.Logger.Info().Msg(fmt.Sprintf("push OCI artifact %v:%v", repo, artifactTag))

	// Push artifact
	artifactDesc, err := p.v2PushManifest(repo, artifactTag, artifact.MediaType, artifactBytes)
	if err != nil {
		return err
	}

	p.Logger.Info().Msg(fmt.Sprintf("pull OCI artifact %v:%v", repo, artifactTag))

	// Pull artifact manifest
	pulledArtifactBytes, err := p.v2PullManifest(repo, artifactTag, artifactDesc)
	if err != nil {
		return err
	}

	pulledArtifact := &v2.Artifact{}
	if err = json.Unmarshal(pulledArtifactBytes, pulledArtifact); err != nil {
		return err
	}

	// Pull artifact layer
	if err = p.v2PullBlob(repo, v1.Descriptor{
		MediaType: pulledArtifact.Blobs[0].MediaType,
		Digest:    pulledArtifact.Blobs[0].Digest,
		Size:      pulledArtifact.Blobs[0].Size,
	}); err != nil {
		return err
	}

	p.Logger.Info().Msg(fmt.Sprintf("subject for artifact %v:%v was pushed as %v:%v", repo, artifactTag, repo, imageTag))

	// Pull subject image
	err = p.pullOCIImage(repo, imageTag, v1.Descriptor{
		MediaType: pulledArtifact.SubjectManifest.MediaType,
		Digest:    pulledArtifact.SubjectManifest.Digest,
		Size:      pulledArtifact.SubjectManifest.Size,
	})
	if err != nil {
		return err
	}

	p.Logger.Info().Msg("check-referrers was successful")

	return nil
}

// scheme determines the HTTP scheme of the request url.
func (p Proxy) url(hostname, route string) string {
	scheme := "https"
	if p.Insecure {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, hostname, route)
}

// auth returns the configured auth type.
func (p Proxy) auth() authType {
	switch p.BasicAuthMode {
	case true:
		return basicAuth
	default:
		return bearerAuth
	}
}

// pullOCIImage pulls the image from repo by tag and validates against the given descriptor.
func (p Proxy) pullOCIImage(repo, tag string, desc v1.Descriptor) error {
	p.Logger.Info().Msg(fmt.Sprintf("pull OCI image %v:%v", repo, tag))

	pulledManifestBytes, err := p.v2PullManifest(repo, tag, desc)
	if err != nil {
		return err
	}

	pulledManifest := &v1.Manifest{}
	if err = json.Unmarshal(pulledManifestBytes, pulledManifest); err != nil {
		return err
	}

	// Pull config blob
	if err = p.v2PullBlob(repo, pulledManifest.Config); err != nil {
		return err
	}

	// Pull layer blob
	if err = p.v2PullBlob(repo, pulledManifest.Layers[0]); err != nil {
		return err
	}

	return nil
}

// pushOCIImage creates and pushes a simple OCI application/vnd.oci.image.manifest.v1+json image.
func (p Proxy) pushOCIImage(repo, tag string) (v1.Descriptor, error) {
	p.Logger.Info().Msg(fmt.Sprintf("push OCI image %v:%v", repo, tag))

	configBytes, err := json.Marshal(ociConfig)
	if err != nil {
		return v1.Descriptor{}, err
	}

	// Upload config blob
	configDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(string(configBytes))))
	if err != nil {
		return v1.Descriptor{}, err
	}

	// Upload a layer
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(fmt.Sprintf(checkHealthLayerFmt, time.Now()))))
	if err != nil {
		return v1.Descriptor{}, err
	}

	ociManifest := v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: v1.Descriptor{
			MediaType: checkHealthMediaType,
			Digest:    configDesc.Digest,
			Size:      configDesc.Size,
		},
		Layers: []v1.Descriptor{
			{
				MediaType: checkHealthMediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
	}

	manifestBytes, err := json.Marshal(ociManifest)
	if err != nil {
		return v1.Descriptor{}, err
	}

	// Push manifest
	return p.v2PushManifest(repo, tag, v1.MediaTypeImageManifest, manifestBytes)
}

// v2PushManifest pushes the data to repo with the given tag and media type, returning the digest and size
// of pushed content.
func (p Proxy) v2PushManifest(repo, tag, mediaType string, manifestBytes []byte) (v1.Descriptor, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	regReq := registryRequest{
		method:      http.MethodPut,
		url:         manifestURL,
		body:        io.NewReader(strings.NewReader(string(manifestBytes))),
		contentType: mediaType,
	}

	_, err := p.roundTrip(regReq, http.StatusCreated, p.auth())
	if err != nil {
		return v1.Descriptor{}, err
	}

	dgst := digest.NewDigest(digest.SHA256, regReq.body.SHA256Hash())
	p.Logger.Info().Msg(dgst.String())
	return v1.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      regReq.body.N(),
	}, nil
}

// v2PullManifest pulls manifest from repo specified by tag or digest and verifies the download size.
func (p Proxy) v2PullManifest(repo, tag string, desc v1.Descriptor) ([]byte, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	regReq := registryRequest{
		method: http.MethodGet,
		url:    manifestURL,
		accept: desc.MediaType,
	}

	manifestPullTripInfo, err := p.roundTrip(regReq, http.StatusOK, p.auth())
	if err != nil {
		return nil, err
	}

	// Validate we got what we sent
	if manifestPullTripInfo.Response.Size != desc.Size {
		return nil, fmt.Errorf("manifest size mismatch; expected: %v, got: %v", desc.Size, manifestPullTripInfo.Response.Size)
	}
	if manifestPullTripInfo.Response.SHA256Sum != desc.Digest {
		return nil, fmt.Errorf("manifest digest mismatch; expected: %v, got: %v", desc.Digest, manifestPullTripInfo.Response.SHA256Sum)
	}

	return manifestPullTripInfo.Body, nil
}

// v2PullBlob pulls a blob from the registry and verifies the digest
// TODO: add size validation
func (p Proxy) v2PullBlob(repo string, desc v1.Descriptor) error {
	var nextURL *url.URL

	// Obtain SAS
	{
		regReq := registryRequest{
			url:    p.url(p.LoginServer, fmt.Sprintf(routeBlobPull, repo, desc.Digest)),
			method: http.MethodGet,
		}

		resp, err := p.roundTrip(regReq, http.StatusTemporaryRedirect, p.auth())
		if err != nil {
			return err
		}

		nextURL = resp.HeaderLocation
	}

	// Download content
	{
		regReq := registryRequest{
			url:    nextURL.String(),
			method: http.MethodGet,
		}

		tripInfo, err := p.roundTrip(regReq, http.StatusOK, noAuth)
		if err != nil {
			return err
		}

		// Validate data integrity
		if tripInfo.Response.SHA256Sum != desc.Digest {
			return fmt.Errorf("blob digest mismatch; expected: %v, got: %v", desc.Digest, tripInfo.Response.SHA256Sum)
		}
	}

	return nil
}

// v2PushBlob uploads a blob to a repository
func (p Proxy) v2PushBlob(repo string, data io.Reader) (d v1.Descriptor, err error) {
	var nextURL *url.URL

	// Initiate blob upload
	{
		regReq := registryRequest{
			url:    p.url(p.LoginServer, fmt.Sprintf(routeInitiateBlobUpload, repo)),
			method: http.MethodPost,
		}

		tripInfo, err := p.roundTrip(regReq, http.StatusAccepted, p.auth())
		if err != nil {
			return d, err
		}

		nextURL = tripInfo.HeaderLocation
	}

	// Upload blob
	{
		regReq := registryRequest{
			url:    nextURL.String(),
			method: http.MethodPatch,
			body:   data,
		}
		tripInfo, err := p.roundTrip(regReq, http.StatusAccepted, p.auth())
		if err != nil {
			return d, err
		}
		nextURL = tripInfo.HeaderLocation
		d.Size = data.N()
		d.Digest = digest.NewDigest(digest.SHA256, data.SHA256Hash())
	}

	// Complete upload
	{
		q := nextURL.Query()
		q.Set("digest", d.Digest.String())
		nextURL.RawQuery = q.Encode()
		regReq := registryRequest{
			url:    nextURL.String(),
			method: http.MethodPut,
		}

		_, err = p.roundTrip(regReq, http.StatusCreated, p.auth())
		if err != nil {
			return d, err
		}
	}

	return d, nil
}

// roundTrip makes an HTTP request using the specified auth mode and returns the response body.
// It validates the returned response code.
func (p Proxy) roundTrip(regReq registryRequest, expected int, at authType) (tripInfo rhttp.RoundTripInfo, err error) {
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

	result, err := t.roundTrip(regReq)
	if err != nil {
		return result, err
	}
	if result.Response.Code != expected {
		return result, fmt.Errorf("invalid response code, expected: %v, got: %v", expected, result.Response.Code)
	}

	return result, nil
}
