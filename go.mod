module github.com/Q42/gcp-sema

go 1.14

replace github.com/Q42/gcp-sema/pkg/secretmanager => ./pkg/secretmanager

require (
	cloud.google.com/go v0.58.0
	github.com/fatih/color v1.9.0
	github.com/flynn/json5 v0.0.0-20160717195620-7620272ed633
	github.com/go-errors/errors v1.1.1
	github.com/golang/protobuf v1.4.2
	github.com/jessevdk/go-flags v1.4.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.4.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	google.golang.org/api v0.26.0
	google.golang.org/genproto v0.0.0-20200608115520-7c474a2e3482
	google.golang.org/grpc v1.29.1
	gopkg.in/yaml.v2 v2.2.2
)
