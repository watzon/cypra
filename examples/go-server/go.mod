module github.com/watzon/cypra/examples/go-server

go 1.25.0

require github.com/watzon/cypra/sdk/go v0.0.0

require (
	github.com/coreos/go-oidc/v3 v3.18.0 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
)

replace github.com/watzon/cypra/sdk/go => ../../sdk/go
