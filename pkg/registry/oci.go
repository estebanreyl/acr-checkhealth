package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aviral26/acr-checkhealth/pkg/io"
	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// oci pushes a small OCI image to the registry and pulls it.
func (p Proxy) oci() error {
	p.Logger.Info().Msg("checking OCI push")

	var (
		repo = fmt.Sprintf("%v%v", repoPrefix, time.Now().Unix())
		tag  = fmt.Sprintf("%v", time.Now().Unix())
	)

	configBytes, err := json.Marshal(ociConfig)
	if err != nil {
		return err
	}

	// Upload config blob
	configDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(string(configBytes))))
	if err != nil {
		return err
	}

	// Upload a layer
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(layer)))
	if err != nil {
		return err
	}

	ociManifest := v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: v1.Descriptor{
			MediaType: mediaType,
			Digest:    configDesc.Digest,
			Size:      configDesc.Size,
		},
		Layers: []v1.Descriptor{
			v1.Descriptor{
				MediaType: mediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
	}

	manifestBytes, err := json.Marshal(ociManifest)
	if err != nil {
		return err
	}

	manifestURL := p.url(p.LoginServer, fmt.Sprintf(routeManifest, repo, tag))

	// Push manifest
	regReq := registryRequest{
		method:      http.MethodPut,
		url:         manifestURL,
		body:        io.NewReader(strings.NewReader(string(manifestBytes))),
		contentType: v1.MediaTypeImageManifest,
	}

	_, err = p.do(regReq, http.StatusCreated, bearerAuth)
	if err != nil {
		return err
	}

	pushedManifestSize := regReq.body.N()
	pushedManifestDigest := digest.NewDigest(digest.SHA256, regReq.body.SHA256Hash())

	p.Logger.Info().Msg("checking OCI pull")
	regReq = registryRequest{
		method: http.MethodGet,
		url:    manifestURL,
		accept: v1.MediaTypeImageManifest,
	}

	manifestPullTripInfo, err := p.do(regReq, http.StatusOK, bearerAuth)
	if err != nil {
		return err
	}

	// Validate we got what we sent
	if manifestPullTripInfo.Response.Size != pushedManifestSize {
		return fmt.Errorf("manifest size mismatch; pushed: %v, pulled: %v", pushedManifestSize, manifestPullTripInfo.Response.Size)
	}
	if manifestPullTripInfo.Response.SHA256Sum != pushedManifestDigest {
		return fmt.Errorf("manifest digest mismatch; pushed: %v, pulled: %v", pushedManifestDigest, manifestPullTripInfo.Response.SHA256Sum)
	}

	pulledManifest := &v1.Manifest{}
	if err = json.Unmarshal(manifestPullTripInfo.Body, pulledManifest); err != nil {
		return err
	}

	// Pull config blob
	if err = p.v2PullBlob(repo, pulledManifest.Config.Digest); err != nil {
		return err
	}

	// Pull layer blob
	if err = p.v2PullBlob(repo, pulledManifest.Layers[0].Digest); err != nil {
		return err
	}

	return nil
}

// v2PullBlob pulls a blob from the registry and verifies the digest
func (p Proxy) v2PullBlob(repo string, dgst digest.Digest) error {
	var nextURL *url.URL

	// Obtain SAS
	{
		regReq := registryRequest{
			url:    p.url(p.LoginServer, fmt.Sprintf(routeBlobPull, repo, dgst)),
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
		if tripInfo.Response.SHA256Sum != dgst {
			return fmt.Errorf("blob digest mismatch; expected: %v, got: %v", dgst, tripInfo.Response.SHA256Sum)
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
