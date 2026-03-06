module github.com/erniealice/espyna-golang

go 1.25.1

require (
	github.com/erniealice/esqyma v0.0.0
	github.com/erniealice/lyngua v0.0.0-00010101000000-000000000000
	github.com/google/cel-go v0.23.0
	github.com/google/uuid v1.6.0
	golang.org/x/text v0.33.0
	google.golang.org/grpc v1.75.1
	google.golang.org/protobuf v1.36.10
	leapfor.xyz/copya v0.0.0
	leapfor.xyz/vya v0.0.0
)

require (
	cel.dev/expr v0.24.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/erniealice/entydad-golang => ../entydad-golang-ryta

replace leapfor.xyz/copya => ../../../master-monorepo-v2/packages/copya

replace github.com/erniealice/esqyma => ../esqyma-ryta

replace github.com/erniealice/lyngua => ../lyngua-ryta

replace leapfor.xyz/vya => ../../../master-monorepo-v2/packages/vya
