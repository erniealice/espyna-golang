//go:build mysql

package event

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"

	mysqlCore "github.com/erniealice/espyna-golang/contrib/mysql/internal/adapter/core"
	"github.com/erniealice/espyna-golang/registry"
	entityid "github.com/erniealice/espyna-golang/registry/entityid"
	interfaces "github.com/erniealice/espyna-golang/shared/database/interfaces"
	eventoccurrencepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/event/event_occurrence"
	"google.golang.org/protobuf/encoding/protojson"
)

func init() {
	registry.RegisterRepositoryFactory("mysql", entityid.EventOccurrence, func(conn any, tableName string) (any, error) {
		db, ok := conn.(*sql.DB)
		if !ok {
			return nil, fmt.Errorf("mysql event_occurrence repository requires *sql.DB, got %T", conn)
		}
		return NewMySQLEventOccurrenceRepository(mysqlCore.NewWorkspaceAwareOperations(db), tableName), nil
	})
}

type MySQLEventOccurrenceRepository struct {
	eventoccurrencepb.UnimplementedEventOccurrenceDomainServiceServer
	dbOps     interfaces.DatabaseOperation
	tableName string
}

func NewMySQLEventOccurrenceRepository(dbOps interfaces.DatabaseOperation, tableName string) eventoccurrencepb.EventOccurrenceDomainServiceServer {
	if tableName == "" {
		tableName = "event_occurrence"
	}
	return &MySQLEventOccurrenceRepository{dbOps: dbOps, tableName: tableName}
}

func (r *MySQLEventOccurrenceRepository) ListEventOccurrences(ctx context.Context, req *eventoccurrencepb.ListEventOccurrencesRequest) (*eventoccurrencepb.ListEventOccurrencesResponse, error) {
	var params *interfaces.ListParams
	if req != nil && req.Filters != nil {
		params = &interfaces.ListParams{Filters: req.Filters}
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, fmt.Errorf("list event_occurrence: %w", err)
	}
	items := make([]*eventoccurrencepb.EventOccurrence, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, fmt.Errorf("marshal list row: %w", err)
		}
		obj := &eventoccurrencepb.EventOccurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, fmt.Errorf("decode list row: %w", err)
		}
		items = append(items, obj)
	}
	return &eventoccurrencepb.ListEventOccurrencesResponse{Data: items}, nil
}

func (r *MySQLEventOccurrenceRepository) GetEventOccurrenceListPageData(ctx context.Context, req *eventoccurrencepb.GetEventOccurrenceListPageDataRequest) (*eventoccurrencepb.GetEventOccurrenceListPageDataResponse, error) {
	params := &interfaces.ListParams{}
	if req != nil {
		params.Filters = req.Filters
		params.Sort = req.Sort
		params.Search = req.Search
		params.Pagination = req.Pagination
	}
	rows, err := r.dbOps.List(ctx, r.tableName, params)
	if err != nil {
		return nil, err
	}
	items := make([]*eventoccurrencepb.EventOccurrence, 0, len(rows.Data))
	for _, row := range rows.Data {
		b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
		if err != nil {
			return nil, err
		}
		obj := &eventoccurrencepb.EventOccurrence{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
			return nil, err
		}
		items = append(items, obj)
	}
	return &eventoccurrencepb.GetEventOccurrenceListPageDataResponse{EventOccurrenceList: items, Pagination: rows.Pagination}, nil
}

func (r *MySQLEventOccurrenceRepository) GetEventOccurrenceItemPageData(ctx context.Context, req *eventoccurrencepb.GetEventOccurrenceItemPageDataRequest) (*eventoccurrencepb.GetEventOccurrenceItemPageDataResponse, error) {
	if req == nil || req.EventOccurrenceId == "" {
		return nil, fmt.Errorf("event_occurrence ID is required")
	}
	row, err := r.dbOps.Read(ctx, r.tableName, req.EventOccurrenceId)
	if err != nil {
		return nil, fmt.Errorf("read event_occurrence: %w", err)
	}
	b, err := json.Marshal(mysqlCore.DenormalizeKeys(row))
	if err != nil {
		return nil, err
	}
	obj := &eventoccurrencepb.EventOccurrence{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(b, obj); err != nil {
		return nil, err
	}
	return &eventoccurrencepb.GetEventOccurrenceItemPageDataResponse{EventOccurrence: obj}, nil
}
