package document_series

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/shared/usecaseerr"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/revenue/document_series"
	"google.golang.org/protobuf/proto"
)

type fakeRepo struct {
	pb.UnimplementedDocumentSeriesDomainServiceServer
	rows map[string]*pb.DocumentSeries
}

func (f *fakeRepo) CreateDocumentSeries(_ context.Context, r *pb.CreateDocumentSeriesRequest) (*pb.CreateDocumentSeriesResponse, error) {
	f.rows[r.Data.Id] = proto.Clone(r.Data).(*pb.DocumentSeries)
	return &pb.CreateDocumentSeriesResponse{Data: []*pb.DocumentSeries{r.Data}, Success: true}, nil
}
func (f *fakeRepo) ReadDocumentSeries(_ context.Context, r *pb.ReadDocumentSeriesRequest) (*pb.ReadDocumentSeriesResponse, error) {
	row, ok := f.rows[r.Data.Id]
	if !ok {
		return nil, errors.New("record not found")
	}
	return &pb.ReadDocumentSeriesResponse{Data: []*pb.DocumentSeries{proto.Clone(row).(*pb.DocumentSeries)}}, nil
}
func (f *fakeRepo) UpdateDocumentSeries(_ context.Context, r *pb.UpdateDocumentSeriesRequest) (*pb.UpdateDocumentSeriesResponse, error) {
	proto.Merge(f.rows[r.Data.Id], r.Data)
	return &pb.UpdateDocumentSeriesResponse{Data: []*pb.DocumentSeries{f.rows[r.Data.Id]}, Success: true}, nil
}

// LockDocumentSeriesForUpdate satisfies DocumentSeriesLocker (the update use case fails closed without it).
func (f *fakeRepo) LockDocumentSeriesForUpdate(_ context.Context, id string) (*pb.DocumentSeries, error) {
	row, ok := f.rows[id]
	if !ok {
		return nil, errors.New("document_series lock: not found")
	}
	return proto.Clone(row).(*pb.DocumentSeries), nil
}
func (f *fakeRepo) ListDocumentSeries(context.Context, *pb.ListDocumentSeriesRequest) (*pb.ListDocumentSeriesResponse, error) {
	var out []*pb.DocumentSeries
	for _, r := range f.rows {
		out = append(out, r)
	}
	return &pb.ListDocumentSeriesResponse{Data: out, Success: true}, nil
}

type authz struct{ deny bool }

func (a *authz) IsEnabled() bool { return true }
func (a *authz) HasPermission(context.Context, string, string) (bool, error) {
	return !a.deny, nil
}

type tx struct{}

func (tx) SupportsTransactions() bool               { return true }
func (tx) IsTransactionActive(context.Context) bool { return false }
func (tx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type ids struct{ n atomic.Int64 }

func (s *ids) GenerateID() string                        { return fmt.Sprintf("ds-%03d", s.n.Add(1)) }
func (s *ids) IsEnabled() bool                           { return true }
func (s *ids) GetProviderInfo() string                   { return "seq" }
func (s *ids) GenerateIDWithPrefix(prefix string) string { return prefix + s.GenerateID() }

func harness(deny bool) (*UseCases, *fakeRepo) {
	repo := &fakeRepo{rows: map[string]*pb.DocumentSeries{}}
	uc := NewUseCases(Repositories{DocumentSeries: repo}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{deny: deny}, ports.NewNoOpTranslator()),
	})
	return uc, repo
}

func ctx() context.Context { return contextutil.WithUserID(context.Background(), "u1") }

func valid(code string) *pb.DocumentSeries {
	return &pb.DocumentSeries{Code: code, IssuerName: "Acme Leasing", DocumentKind: enumspb.ChargeDocumentKind_CHARGE_DOCUMENT_KIND_RECOVERY_DOCUMENT, Prefix: strp("SOA-"), NumberPadding: 6}
}

func TestCreateDefaultsAndUniqueCode(t *testing.T) {
	uc, repo := harness(false)
	resp, err := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")})
	if err != nil {
		t.Fatal(err)
	}
	row := repo.rows[resp.Data[0].Id]
	if row.NextNumber != 1 || row.NumberPadding != 6 || row.Status != pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE || row.FiscalReset != pb.DocumentSeriesFiscalReset_DOCUMENT_SERIES_FISCAL_RESET_NONE {
		t.Fatalf("defaults wrong: %+v", row)
	}
	if _, err := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")}); !usecaseerr.IsCode(err, "code_taken") {
		t.Fatalf("duplicate code: want code_taken, got %v", err)
	}
	for name, mut := range map[string]func(*pb.DocumentSeries){
		"lowercase code":  func(s *pb.DocumentSeries) { s.Code = "soa" },
		"no issuer":       func(s *pb.DocumentSeries) { s.IssuerName = " " },
		"no kind":         func(s *pb.DocumentSeries) { s.DocumentKind = 0 },
		"negative next":   func(s *pb.DocumentSeries) { s.NextNumber = -1 },
		"zero padding":    func(s *pb.DocumentSeries) { s.NumberPadding = 0 },
		"padding over 18": func(s *pb.DocumentSeries) { s.NumberPadding = 19 },
	} {
		s := valid("OTHER")
		mut(s)
		if _, err := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: s}); !usecaseerr.IsCode(err, "validation") {
			t.Errorf("%s: want validation, got %v", name, err)
		}
	}
}

func TestUpdateIsRetireOnly(t *testing.T) {
	uc, repo := harness(false)
	resp, _ := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")})
	id := resp.Data[0].Id
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Name: strp("Statements"), NextNumber: 99, Code: "HACK", NumberPadding: 6}}); err != nil {
		t.Fatal(err)
	}
	if r := repo.rows[id]; r.GetName() != "Statements" || r.NextNumber != 1 || r.Code != "SOA" {
		t.Fatalf("code/next_number must be immutable through update: %+v", r)
	}
	for _, pad := range []int32{-1, 19} {
		if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Name: strp("x"), NumberPadding: pad}}); !usecaseerr.IsCode(err, "validation") {
			t.Fatalf("number_padding %d must be refused on update, got %v", pad, err)
		}
	}
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Status: pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_ACTIVE}}); !usecaseerr.IsCode(err, "validation") {
		t.Fatalf("setting ACTIVE must be refused, got %v", err)
	}
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Status: pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED, NumberPadding: 6}}); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Name: strp("again"), NumberPadding: 6}}); !usecaseerr.IsCode(err, "retired") {
		t.Fatalf("retired series must be immutable, got %v", err)
	}
}

func TestPermissionDeniedFailsClosed(t *testing.T) {
	uc, _ := harness(true)
	if _, err := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")}); err == nil {
		t.Error("create must be denied")
	}
	if _, err := uc.ListDocumentSeries.Execute(ctx(), &pb.ListDocumentSeriesRequest{}); err == nil {
		t.Error("list must be denied")
	}
}

// hidden wraps the repository so its Locker capability is not visible.
type noLockRepo struct {
	pb.DocumentSeriesDomainServiceServer
}

// C4: the update use case fails closed when the repository cannot lock the series row.
func TestUpdateFailsClosedWithoutLocker(t *testing.T) {
	uc, repo := harness(false)
	resp, _ := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")})
	id := resp.Data[0].Id
	nolock := NewUseCases(Repositories{DocumentSeries: noLockRepo{repo}}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{}, ports.NewNoOpTranslator()),
	})
	_, err := nolock.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Name: strp("x"), NumberPadding: 6}})
	if err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("unlockable repository must fail closed with a non-refusal error, got %v", err)
	}
	if repo.rows[id].GetName() == "x" {
		t.Fatal("nothing may be written without the lock")
	}
}

type failingReadRepo struct {
	pb.DocumentSeriesDomainServiceServer
}

func (failingReadRepo) ReadDocumentSeries(context.Context, *pb.ReadDocumentSeriesRequest) (*pb.ReadDocumentSeriesResponse, error) {
	return nil, errors.New("connection reset by peer")
}

// C9: a repository failure is never reported as not_found; a genuinely missing row is.
func TestReadDoesNotMaskRepositoryErrors(t *testing.T) {
	uc, _ := harness(false)
	if _, err := uc.ReadDocumentSeries.Execute(ctx(), &pb.ReadDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: "ghost"}}); !usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("missing row: want not_found, got %v", err)
	}
	bad := NewUseCases(Repositories{DocumentSeries: failingReadRepo{}}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(&authz{}, ports.NewNoOpTranslator()),
	})
	if _, err := bad.ReadDocumentSeries.Execute(ctx(), &pb.ReadDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: "x"}}); err == nil || usecaseerr.IsCode(err, "not_found") {
		t.Fatalf("infrastructure failure must not surface as not_found: %v", err)
	}
}

// Partial update: a status-only retire sends no number_padding; prefix/padding are locked once numbers are issued.
func TestUpdateStatusOnlyAndNumberingLocked(t *testing.T) {
	uc, repo := harness(false)
	resp, _ := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")})
	id := resp.Data[0].Id
	// next_number == 1: prefix and padding may still change.
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Prefix: strp("STM-"), NumberPadding: 5}}); err != nil {
		t.Fatalf("pre-issuance edit: %v", err)
	}
	repo.rows[id].NextNumber = 4
	for name, d := range map[string]*pb.DocumentSeries{
		"prefix":  {Id: id, Prefix: strp("X-")},
		"padding": {Id: id, NumberPadding: 8},
	} {
		if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: d}); !usecaseerr.IsCode(err, "numbering_locked") {
			t.Fatalf("%s change after issuance must be numbering_locked, got %v", name, err)
		}
	}
	// same values and descriptive edits stay allowed.
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Name: strp("N"), Prefix: strp("STM-"), NumberPadding: 5}}); err != nil {
		t.Fatalf("unchanged prefix/padding: %v", err)
	}
	// status-only retire, no padding resent.
	if _, err := uc.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Status: pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED}}); err != nil {
		t.Fatalf("status-only retire: %v", err)
	}
	if r := repo.rows[id]; r.NumberPadding != 5 || r.GetPrefix() != "STM-" {
		t.Fatalf("retire must not touch padding/prefix: %+v", r)
	}
}

// shadowAuthz allows through the shadow-capable path and answers the strict path with strictAllow.
type shadowAuthz struct{ strictAllow bool }

func (shadowAuthz) IsEnabled() bool                                             { return true }
func (shadowAuthz) HasPermission(context.Context, string, string) (bool, error) { return true, nil }
func (a shadowAuthz) HasPermissionStrict(context.Context, string, string) (bool, error) {
	return a.strictAllow, nil
}

// C26: create and update (which carries retire) use the strict gate: a shadow-mode allow never lets
// a denied principal start or retire a numbering series.
func TestCreateAndRetireUseTheStrictGate(t *testing.T) {
	uc, repo := harness(false)
	resp, err := uc.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("SOA")})
	if err != nil {
		t.Fatal(err)
	}
	id := resp.Data[0].Id
	strictDeny := NewUseCases(Repositories{DocumentSeries: repo}, Services{
		Authorizer: ports.NewNoOpAuthorizer(), Transactor: tx{}, Translator: ports.NewNoOpTranslator(), IDGenerator: &ids{},
		ActionGatekeeper: actiongate.NewActionGatekeeper(shadowAuthz{strictAllow: false}, ports.NewNoOpTranslator()),
	})
	if _, err := strictDeny.CreateDocumentSeries.Execute(ctx(), &pb.CreateDocumentSeriesRequest{Data: valid("OTHER")}); !usecaseerr.IsCode(err, "permission_denied") {
		t.Fatalf("create: want permission_denied under a strict deny, got %v", err)
	}
	if _, err := strictDeny.UpdateDocumentSeries.Execute(ctx(), &pb.UpdateDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id, Status: pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED}}); !usecaseerr.IsCode(err, "permission_denied") {
		t.Fatalf("retire: want permission_denied under a strict deny, got %v", err)
	}
	if repo.rows[id].GetStatus() == pb.DocumentSeriesStatus_DOCUMENT_SERIES_STATUS_RETIRED || len(repo.rows) != 1 {
		t.Fatalf("a strict deny wrote: %+v", repo.rows)
	}
	// The shadow-capable read path stays on the regular gate.
	if _, err := strictDeny.ReadDocumentSeries.Execute(ctx(), &pb.ReadDocumentSeriesRequest{Data: &pb.DocumentSeries{Id: id}}); err != nil {
		t.Fatalf("read: %v", err)
	}
}
