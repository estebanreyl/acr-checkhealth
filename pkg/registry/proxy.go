package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	rhttp "github.com/aviral26/acr-checkhealth/pkg/http"
	"github.com/aviral26/acr-checkhealth/pkg/io"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ociimagespec "github.com/opencontainers/image-spec/specs-go/v1"
	orasartifact "github.com/oras-project/artifacts-spec/specs-go/v1"

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

	// Referrer routes
	routeReferrers = "/oras/artifacts/v1/%s/manifests/%s/referrers" // add repo name and digest
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
	ociConfig = ociimagespec.Image{
		Author: checkHealthAuthor,
	}
)

// referrer describes a single object in a /referrers API response.
// See: https://gist.github.com/aviral26/ca4b0c1989fd978e74be75cbf3f3ea92
// type referrer struct {
// 	// MediaType is the media type of the targeted content.
// 	MediaType string `json:"mediaType"`

// 	// Digest is the digest of the targeted content.
// 	Digest string `json:"digest"`

// 	// Size is the size of the targeted content.
// 	Size int64 `json:"size"`

// 	// ArtifactType is the artifact type of the targeted content.
// 	ArtifactType string `json:"artifactType"`

// 	// Data is the base64 encoded bytes of the targeted content.
// 	Data string `json:"data,omitempty"`
// }

// referrersResponse describes the referrers API response.
// See: https://gist.github.com/aviral26/ca4b0c1989fd978e74be75cbf3f3ea92
type referrersResponse struct {
	// Referrers is a collection of referrers.
	Referrers []orasartifact.Descriptor `json:"references"`
}

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
func (p Proxy) CheckReferrers(count int) error {
	var (
		repo     = fmt.Sprintf("%v%v", checkHealthRepoPrefix, time.Now().Unix())
		imageTag = fmt.Sprintf("%v", time.Now().Unix())
	)

	// Push simple image
	imageDesc, err := p.pushOCIImage(repo, imageTag)
	if err != nil {
		return err
	}

	pushedReferrers, err := p.pushReferrers(repo, imageDesc, count)
	if err != nil {
		return err
	}

	// Discover and verify referrers
	err = p.verifyReferrers(repo, imageDesc, pushedReferrers)
	if err != nil {
		return err
	}

	p.Logger.Info().Msg(fmt.Sprintf("subject is %v:%v", repo, imageTag))

	// Pull subject image
	err = p.pullOCIImage(repo, imageTag, imageDesc)
	if err != nil {
		return err
	}

	p.Logger.Info().Msg("check-referrers was successful")

	return nil
}

// CheckReferrers checks the registry's referrer APIs.
func (p Proxy) CheckReferrersOutOfOrder(count int) error {
	var (
		repo     = fmt.Sprintf("%v%v", checkHealthRepoPrefix, time.Now().Unix())
		imageTag = fmt.Sprintf("%v", time.Now().Unix())
	)
	p.Logger.Info().Msg(fmt.Sprint("Push OCI subject layers"))
	digest, _, _, mediaType, data, err := p.createOCIImage(repo, imageTag)
	if err != nil {
		return err
	}

	// Push simple image
	imageDesc := ociimagespec.Descriptor{
		Digest:    digest,
		MediaType: mediaType,
		Size:      int64(len(data)),
	}

	pushedReferrers, err := p.pushReferrers(repo, imageDesc, count)
	if err != nil {
		return err
	}

	// Push subject after the referrers
	p.Logger.Info().Msg(fmt.Sprintf("Push OCI subject: %v:%v  Digest %v", repo, imageTag, digest.String()))
	p.v2PushManifest(repo, imageTag, ociimagespec.MediaTypeImageManifest, data)

	// Discover and verify referrers
	err = p.verifyReferrers(repo, imageDesc, pushedReferrers)
	if err != nil {
		return err
	}
	// Pull subject image
	err = p.pullOCIImage(repo, imageTag, imageDesc)
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

func (p Proxy) pushReferrers(repo string, subject ociimagespec.Descriptor, count int) ([]ociimagespec.Descriptor, error) {
	if count < 1 {
		p.Logger.Warn().Msg("setting referrers count to 1")
		count = 1
	} else if count > 100 {
		p.Logger.Warn().Msg("max referrers limited to 100")
		count = 100
	}

	var referrers []ociimagespec.Descriptor

	for i := 0; i < count; i++ {
		time.Sleep(time.Second * 4)
		// Push artifact layer
		layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(fmt.Sprintf(checkHealthLayerFmt+"  ~ %v", time.Now(), i))))
		if err != nil {
			return nil, err
		}

		artifact := ociimagespec.Artifact{
			Blobs: []ociimagespec.Descriptor{
				{
					MediaType: layerDesc.MediaType,
					Digest:    layerDesc.Digest,
					Size:      layerDesc.Size,
				},
			},
			ArtifactType: checkHealthArtifactType,
			Subject: &ociimagespec.Descriptor{
				MediaType: subject.MediaType,
				Digest:    subject.Digest,
				Size:      subject.Size,
			},
			MediaType: ociimagespec.MediaTypeArtifactManifest,
		}

		if i%2 == 0 {
			artifact.Annotations = map[string]string{ociimagespec.AnnotationArtifactCreated: time.Now().Format(time.RFC3339)}
		}

		artifactBytes, err := json.Marshal(artifact)
		if err != nil {
			return nil, err
		}

		artifactTag := fmt.Sprintf("art-%v-%v", i+1, time.Now().Unix())
		p.Logger.Info().Msg(fmt.Sprintf("push OCI artifact %v:%v, createdTime %t", repo, artifactTag, i%2 == 0))

		// Push artifact
		artifactDesc, err := p.v2PushManifest(repo, artifactTag, ociimagespec.MediaTypeArtifactManifest, artifactBytes)
		if err != nil {
			return nil, err
		}

		referrers = append(referrers, ociimagespec.Descriptor{
			MediaType:    artifactDesc.MediaType,
			Digest:       artifactDesc.Digest,
			Size:         artifactDesc.Size,
			ArtifactType: artifact.ArtifactType}) // Data: base64.StdEncoding.EncodeToString(artifactBytes)})
	}

	return referrers, nil
}

func (p Proxy) createOCIImage(repo, tag string) (digest.Digest, string, string, string, []byte, error) {
	configBytes, err := json.Marshal(ociConfig)
	if err != nil {
		return "", "", "", "", nil, err
	}

	// Upload config blob
	configDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(string(configBytes))))
	if err != nil {
		return "", "", "", "", nil, err
	}

	// Upload a layer
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(fmt.Sprintf(checkHealthLayerFmt, time.Now()))))
	if err != nil {
		return "", "", "", "", nil, err
	}

	ociManifest := ociimagespec.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: ociimagespec.Descriptor{
			MediaType: checkHealthMediaType,
			Digest:    configDesc.Digest,
			Size:      configDesc.Size,
		},
		Layers: []ociimagespec.Descriptor{
			{
				MediaType: checkHealthMediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
	}

	manifestBytes, err := json.Marshal(ociManifest)
	if err != nil {
		return "", "", "", "", nil, err
	}

	body := io.NewReader(strings.NewReader(string(manifestBytes)))
	ioutil.ReadAll(body)
	dgst := digest.NewDigest(digest.SHA256, body.SHA256Hash())
	return dgst, repo, tag, ociimagespec.MediaTypeImageManifest, manifestBytes, nil
}

// verifyReferrers verifies that the given subject has the expectedReferrers in the registry.
func (p Proxy) verifyReferrers(repo string, subject ociimagespec.Descriptor, expectedReferrers []ociimagespec.Descriptor) error {
	p.Logger.Info().Msg(fmt.Sprintf("discover referrers for %v@%v", repo, subject.Digest))

	// Discover all referrers
	discoveredReferrers, err := p.getReferrers(repo, subject.Digest)
	if err != nil {
		return err
	}

	if len(discoveredReferrers) != len(expectedReferrers) {
		return fmt.Errorf("unexpected referrers count, expected: %v, got: %v", len(expectedReferrers), len(discoveredReferrers))
	}

	matchedReferrers := make(map[string]string)

	for _, discoveredReferrer := range discoveredReferrers {
		for _, expectedReferrer := range expectedReferrers {
			if discoveredReferrer.Digest == expectedReferrer.Digest &&
				discoveredReferrer.Size == expectedReferrer.Size &&
				discoveredReferrer.MediaType == expectedReferrer.MediaType &&
				// discoveredReferrer.Data == expectedReferrer.Data {
				discoveredReferrer.ArtifactType == expectedReferrer.ArtifactType {

				// Verify this is a unique digest
				if _, ok := matchedReferrers[discoveredReferrer.Digest.String()]; ok {
					return errors.New("duplicate referrer result detected")
				}

				// Successfully discovered
				p.Logger.Info().Msg(discoveredReferrer.Digest.String())
				matchedReferrers[discoveredReferrer.Digest.String()] = ""
				break
			}
		}
	}

	if len(matchedReferrers) != len(expectedReferrers) {
		return errors.New("not all referrers matched")
	}

	for _, gotReferrer := range discoveredReferrers {
		p.Logger.Info().Msg(fmt.Sprintf("pull referrer %v@%v", repo, gotReferrer.Digest))

		// Pull artifact manifest
		pulledArtifactBytes, err := p.v2PullManifest(repo, gotReferrer.Digest.String(),
			ociimagespec.Descriptor{MediaType: gotReferrer.MediaType, Digest: digest.Digest(gotReferrer.Digest), Size: gotReferrer.Size})
		if err != nil {
			return err
		}

		pulledArtifact := &orasartifact.Manifest{}
		if err = json.Unmarshal(pulledArtifactBytes, pulledArtifact); err != nil {
			return err
		}

		// Pull artifact layer
		if err = p.v2PullBlob(repo, ociimagespec.Descriptor{
			MediaType: pulledArtifact.Blobs[0].MediaType,
			Digest:    pulledArtifact.Blobs[0].Digest,
			Size:      pulledArtifact.Blobs[0].Size,
		}); err != nil {
			return err
		}
	}

	return nil
}

// pullOCIImage pulls the image from repo by tag and validates against the given descriptor.
func (p Proxy) pullOCIImage(repo, tag string, desc ociimagespec.Descriptor) error {
	p.Logger.Info().Msg(fmt.Sprintf("pull OCI image %v:%v", repo, tag))

	pulledManifestBytes, err := p.v2PullManifest(repo, tag, desc)
	if err != nil {
		return err
	}

	pulledManifest := &ociimagespec.Manifest{}
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
func (p Proxy) pushOCIImage(repo, tag string) (ociimagespec.Descriptor, error) {
	p.Logger.Info().Msg(fmt.Sprintf("push OCI image %v:%v", repo, tag))

	configBytes, err := json.Marshal(ociConfig)
	if err != nil {
		return ociimagespec.Descriptor{}, err
	}

	// Upload config blob
	configDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(string(configBytes))))
	if err != nil {
		return ociimagespec.Descriptor{}, err
	}

	// Upload a layer
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(fmt.Sprintf(checkHealthLayerFmt, time.Now()))))
	if err != nil {
		return ociimagespec.Descriptor{}, err
	}

	ociManifest := ociimagespec.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: ociimagespec.Descriptor{
			MediaType: checkHealthMediaType,
			Digest:    configDesc.Digest,
			Size:      configDesc.Size,
		},
		Layers: []ociimagespec.Descriptor{
			{
				MediaType: checkHealthMediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
	}

	manifestBytes, err := json.Marshal(ociManifest)
	if err != nil {
		return ociimagespec.Descriptor{}, err
	}

	// Push manifest
	return p.v2PushManifest(repo, tag, ociimagespec.MediaTypeImageManifest, manifestBytes)
}

// getReferrers discovers referrers of the given subject using the referrers API.
// See: https://gist.github.com/aviral26/ca4b0c1989fd978e74be75cbf3f3ea92
func (p Proxy) getReferrers(repo string, subject digest.Digest) ([]orasartifact.Descriptor, error) {
	referrersUrl := p.url(p.LoginServer, fmt.Sprintf(routeReferrers, repo, string(subject)))

	var (
		referrers []orasartifact.Descriptor
		page      int
	)

	for {
		regReq := registryRequest{
			method: http.MethodGet,
			url:    referrersUrl,
		}

		page += 1

		p.Logger.Debug().Msg(fmt.Sprintf("enumerating referrers page %v, %v", page, regReq.url))

		tripInfo, err := p.roundTrip(regReq, http.StatusOK, p.auth())
		if err != nil {
			return nil, err
		}

		var resp referrersResponse
		err = json.Unmarshal(tripInfo.Body, &resp)
		if err != nil {
			return nil, err
		}

		referrers = append(referrers, resp.Referrers...)

		if tripInfo.HeaderLink == "" {
			break
		}

		link := tripInfo.HeaderLink
		referrersUrl = link[1:strings.Index(link, ">")]
	}

	p.Logger.Info().Msg(fmt.Sprintf("found %v referrers", len(referrers)))

	return referrers, nil
}

// v2PushManifest pushes the data to repo with the given tag and media type, returning the digest and size
// of pushed content.
func (p Proxy) v2PushManifest(repo, tag, mediaType string, manifestBytes []byte) (ociimagespec.Descriptor, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	regReq := registryRequest{
		method:      http.MethodPut,
		url:         manifestURL,
		body:        io.NewReader(strings.NewReader(string(manifestBytes))),
		contentType: mediaType,
	}

	_, err := p.roundTrip(regReq, http.StatusCreated, p.auth())
	if err != nil {
		return ociimagespec.Descriptor{}, err
	}

	dgst := digest.NewDigest(digest.SHA256, regReq.body.SHA256Hash())
	p.Logger.Info().Msg(dgst.String())
	return ociimagespec.Descriptor{
		MediaType: mediaType,
		Digest:    dgst,
		Size:      regReq.body.N(),
	}, nil
}

// v2PullManifest pulls manifest from repo specified by tag or digest and verifies the download size.
func (p Proxy) v2PullManifest(repo, tagOrDigest string, desc ociimagespec.Descriptor) ([]byte, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tagOrDigest))

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
func (p Proxy) v2PullBlob(repo string, desc ociimagespec.Descriptor) error {
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
		if tripInfo.Response.Size != desc.Size {
			return fmt.Errorf("blob size mismatch; expected: %v, got: %v", desc.Size, tripInfo.Response.Size)
		}
	}

	return nil
}

// v2PushBlob uploads a blob to a repository
func (p Proxy) v2PushBlob(repo string, data io.Reader) (d ociimagespec.Descriptor, err error) {
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
		return result, fmt.Errorf("invalid response code, expected: %v, got: %v, %s", expected, result.Response.Code, result.Response.Body)
	}

	return result, nil
}
