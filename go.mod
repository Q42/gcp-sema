module github.com/Q42/gcp-sema

go 1.16

replace github.com/Q42/gcp-sema/pkg/secretmanager => ./pkg/secretmanager

replace github.com/Q42/gcp-sema/pkg/schema => ./pkg/schema

replace github.com/Q42/gcp-sema/pkg/handlers => ./pkg/handlers

require (
	cloud.google.com/go/secretmanager v1.4.0
	github.com/BTBurke/snapshot v1.7.1
	github.com/fatih/color v1.12.0
	github.com/flynn/json5 v0.0.0-20160717195620-7620272ed633
	github.com/go-errors/errors v1.4.0
	github.com/golang/protobuf v1.5.2
	github.com/jessevdk/go-flags v1.5.0
	github.com/joho/godotenv v1.3.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/robertkrimen/otto v0.0.0-20200922221731-ef014fd054ac // indirect
	github.com/segmentio/textio v1.2.0
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	google.golang.org/api v0.85.0
	google.golang.org/genproto v0.0.0-20220617124728-180714bec0ad
	google.golang.org/grpc v1.47.0
	gopkg.in/sourcemap.v1 v1.0.5 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/apimachinery v0.20.5
)
