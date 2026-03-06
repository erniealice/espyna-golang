module github.com/erniealice/espyna-golang/contrib/postgres

go 1.25.0

require (
	github.com/erniealice/espyna-golang v0.0.0
	github.com/erniealice/esqyma v0.0.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251002232023-7c0ddcbb5797 // indirect
	google.golang.org/grpc v1.75.1 // indirect
)

replace github.com/erniealice/espyna-golang => ../..

replace github.com/erniealice/esqyma => ../../../esqyma-ryta
