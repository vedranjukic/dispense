module cli

go 1.24.0

toolchain go1.24.7

require (
	apiclient v0.0.0
	github.com/asdine/storm/v3 v3.2.1
	github.com/go-playground/validator/v10 v10.16.0
	github.com/golang/protobuf v1.5.4
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2
	github.com/modelcontextprotocol/go-sdk v0.1.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/crypto v0.42.0
	google.golang.org/genproto/googleapis/api v0.0.0-20250922171735-9219d122eba9
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.9
)

replace apiclient => ../libs/api-client-go

require (
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	go.etcd.io/bbolt v1.3.4 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250908214217-97024824d090 // indirect
)
