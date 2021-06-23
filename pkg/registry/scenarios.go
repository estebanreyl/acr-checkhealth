package registry

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aviral26/acr-checkhealth/pkg/io"
	"github.com/opencontainers/image-spec/specs-go"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// verifyReferrers pushes an OCI image and an associated referrer artifact.
// It then verifies that both objects are pullable.
func (p Proxy) verifyReferrers() error {
	return fmt.Errorf("verifyReferrers: not implemented")
}

// verifySimple pushes a small OCI image to the registry and pulls it.
func (p Proxy) verifySimple() error {
	var (
		repo = fmt.Sprintf("%v%v", repoPrefix, time.Now().Unix())
		tag  = fmt.Sprintf("%v", time.Now().Unix())
	)

	// Push simple image
	desc, err := p.pushImage(repo, tag)
	if err != nil {
		return err
	}

	// Pull image
	err = p.pullImage(repo, tag, desc)
	if err != nil {
		return err
	}

	return nil
}

// pullImage pulls the image from repo by tag and validates against the given descriptor.
func (p Proxy) pullImage(repo, tag string, desc v1.Descriptor) error {
	p.Logger.Info().Msg(fmt.Sprintf("pull OCI image %v:%v", repo, tag))

	pulledManifest, err := p.v2PullManifest(repo, tag, desc)
	if err != nil {
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

// pushImage creates and pushes a simple OCI application/vnd.oci.image.manifest.v1+json image.
func (p Proxy) pushImage(repo, tag string) (v1.Descriptor, error) {
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
	layerDesc, err := p.v2PushBlob(repo, io.NewReader(strings.NewReader(layer)))
	if err != nil {
		return v1.Descriptor{}, err
	}

	ociManifest := v1.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config: v1.Descriptor{
			MediaType: testMediaType,
			Digest:    configDesc.Digest,
			Size:      configDesc.Size,
		},
		Layers: []v1.Descriptor{
			{
				MediaType: testMediaType,
				Digest:    layerDesc.Digest,
				Size:      layerDesc.Size,
			},
		},
	}

	// Push manifest
	return p.v2PushManifest(repo, tag, v1.MediaTypeImageManifest, ociManifest)
}
