package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aviral26/acr-checkhealth/pkg/io"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// v2PushManifest pushes the data to repo with the given tag and media type, returning the digest and size
// of pushed content.
func (p Proxy) v2PushManifest(repo, tag, mediaType string, manifest v1.Manifest) (v1.Descriptor, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return v1.Descriptor{}, err
	}

	regReq := registryRequest{
		method:      http.MethodPut,
		url:         manifestURL,
		body:        io.NewReader(strings.NewReader(string(manifestBytes))),
		contentType: mediaType,
	}

	_, err = p.do(regReq, http.StatusCreated, bearerAuth)
	if err != nil {
		return v1.Descriptor{}, err
	}

	return v1.Descriptor{
		MediaType: mediaType,
		Digest:    digest.NewDigest(digest.SHA256, regReq.body.SHA256Hash()),
		Size:      regReq.body.N(),
	}, nil
}

// v2PullManifest pulls manifest from repo specified by tag or digest and verifies the download size.
func (p Proxy) v2PullManifest(repo, tag string, desc v1.Descriptor) (*v1.Manifest, error) {
	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	regReq := registryRequest{
		method: http.MethodGet,
		url:    manifestURL,
		accept: desc.MediaType,
	}

	manifestPullTripInfo, err := p.do(regReq, http.StatusOK, bearerAuth)
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

	pulledManifest := &v1.Manifest{}
	if err = json.Unmarshal(manifestPullTripInfo.Body, pulledManifest); err != nil {
		return nil, err
	}

	return pulledManifest, nil
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

		resp, err := p.do(regReq, http.StatusTemporaryRedirect, bearerAuth)
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

		tripInfo, err := p.do(regReq, http.StatusOK, noAuth)
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

		tripInfo, err := p.do(regReq, http.StatusAccepted, bearerAuth)
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
		tripInfo, err := p.do(regReq, http.StatusAccepted, bearerAuth)
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

		_, err = p.do(regReq, http.StatusCreated, bearerAuth)
		if err != nil {
			return d, err
		}
	}

	return d, nil
}
