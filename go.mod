module github.com/aviral26/acr-checkhealth

go 1.15

replace github.com/opencontainers/artifacts => github.com/aviral26/artifacts v0.0.3

require (
	github.com/opencontainers/artifacts v0.0.0-00010101000000-000000000000
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/rs/zerolog v1.20.0
	github.com/urfave/cli/v2 v2.2.0
)
