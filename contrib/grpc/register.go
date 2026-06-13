// Package grpc registers the gRPC server adapter with espyna's registry.
// Blank-import to enable it (registration fires under -tags grpc):
//
//	import _ "github.com/erniealice/espyna-golang/contrib/grpc"
package grpc

import _ "github.com/erniealice/espyna-golang/contrib/grpc/internal/adapter"
