//go:build postgresql

package operation

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// protoToMap marshals any proto message to a JSON-shaped map[string]any for the
// generic dbOps write path. It is a pure proto→map utility (protojson.Marshal +
// json.Unmarshal) with no domain logic.
func protoToMap(msg proto.Message) (map[string]any, error) {
	jsonData, err := protojson.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf to JSON: %w", err)
	}
	var data map[string]any
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map: %w", err)
	}
	return data, nil
}
