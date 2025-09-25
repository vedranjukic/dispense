module cli

go 1.24.0

toolchain go1.24.7

require (
	apiclient v0.0.0
	github.com/asdine/storm/v3 v3.2.1
	github.com/go-playground/validator/v10 v10.16.0
	github.com/modelcontextprotocol/go-sdk v0.1.0
	github.com/spf13/cobra v1.8.0
	golang.org/x/crypto v0.42.0
	google.golang.org/grpc v1.67.0
	google.golang.org/protobuf v1.34.2
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
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
)
