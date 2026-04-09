module github.com/erniealice/espyna-golang/contrib/postgres

go 1.25.1

require (
	github.com/erniealice/espyna-golang v0.0.0
	github.com/erniealice/esqyma v0.0.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	google.golang.org/protobuf v1.36.11
)

require (
	cel.dev/expr v0.24.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/google/cel-go v0.23.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/exp v0.0.0-20230515195305-f3d0a9c9a5cc // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/grpc v1.75.1 // indirect
)

replace github.com/erniealice/espyna-golang => ../..

replace github.com/erniealice/esqyma => ../../../esqyma
