package task_outcome

import (
	"context"
	"errors"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	portsdomain "github.com/erniealice/espyna-golang/internal/application/ports/domain"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/task_outcome"
)

// fakeTx is a transactional provider that runs fn inline.
type fakeTx struct{ supports bool }

func (f fakeTx) ExecuteInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (f fakeTx) SupportsTransactions() bool               { return f.supports }
func (f fakeTx) IsTransactionActive(context.Context) bool { return false }

// plainRepo is a repository WITHOUT the guard / conditional capabilities.
type plainRepo struct {
	pb.UnimplementedTaskOutcomeDomainServiceServer
	stored      *pb.TaskOutcome
	updates     int
	creates     int
	lastWritten *pb.TaskOutcome
}

func (p *plainRepo) ReadTaskOutcome(_ context.Context, req *pb.ReadTaskOutcomeRequest) (*pb.ReadTaskOutcomeResponse, error) {
	if p.stored == nil || p.stored.GetId() != req.GetData().GetId() {
		return nil, errors.New("not found")
	}
	return &pb.ReadTaskOutcomeResponse{Data: []*pb.TaskOutcome{p.stored}}, nil
}
func (p *plainRepo) UpdateTaskOutcome(_ context.Context, req *pb.UpdateTaskOutcomeRequest) (*pb.UpdateTaskOutcomeResponse, error) {
	p.updates++
	p.lastWritten = req.GetData()
	return &pb.UpdateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{req.GetData()}}, nil
}
func (p *plainRepo) CreateTaskOutcome(_ context.Context, req *pb.CreateTaskOutcomeRequest) (*pb.CreateTaskOutcomeResponse, error) {
	p.creates++
	p.lastWritten = req.GetData()
	return &pb.CreateTaskOutcomeResponse{Success: true, Data: []*pb.TaskOutcome{req.GetData()}}, nil
}

// condRepo adds the cell-write guard + Q26 conditional writer, emulating the
// postgres adapter's semantics in memory.
type condRepo struct {
	plainRepo
	guardErr      error
	guardedTasks  []string
	condUpdates   int
	condCreates   int
	activeByCell  map[string]bool
	lastExpected  *pb.TaskOutcome
	storedChanged bool // simulate a concurrent writer between snapshot and write
}

func (c *condRepo) GuardCellWrite(_ context.Context, jobTaskID string) error {
	c.guardedTasks = append(c.guardedTasks, jobTaskID)
	return c.guardErr
}
func (c *condRepo) UpdateTaskOutcomeIfUnchanged(_ context.Context, data, expected *pb.TaskOutcome) (*pb.TaskOutcome, error) {
	c.condUpdates++
	c.lastExpected = expected
	if c.storedChanged || !SnapshotMatches(c.stored, expected) {
		return nil, portsdomain.ErrTaskOutcomeConflict
	}
	c.lastWritten = data
	return data, nil
}
func (c *condRepo) CreateTaskOutcomeIfAbsent(_ context.Context, data *pb.TaskOutcome) (*pb.TaskOutcome, error) {
	c.condCreates++
	k := data.GetJobTaskId() + ":" + data.GetCriteriaVersionId()
	if c.activeByCell[k] {
		return nil, portsdomain.ErrTaskOutcomeConflict
	}
	if c.activeByCell == nil {
		c.activeByCell = map[string]bool{}
	}
	c.activeByCell[k] = true
	c.lastWritten = data
	return data, nil
}

func svcs(tx bool) (UpdateTaskOutcomeServices, CreateTaskOutcomeServices) {
	gk := actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
	var t ports.Transactor = fakeTx{supports: tx}
	return UpdateTaskOutcomeServices{ActionGatekeeper: gk, Transactor: t, Translator: ports.NewNoOpTranslator()},
		CreateTaskOutcomeServices{ActionGatekeeper: gk, Transactor: t, Translator: ports.NewNoOpTranslator(), IDGenerator: fixedID{}}
}

type fixedID struct{}

func (fixedID) GenerateID() string                   { return "new-id" }
func (fixedID) GenerateIDWithPrefix(p string) string { return p + "new-id" }
func (fixedID) IsEnabled() bool                      { return true }
func (fixedID) GetProviderInfo() string              { return "fixed" }

func fp(v float64) *float64 { return &v }
func sp(v string) *string   { return &v }
func ip(v int64) *int64     { return &v }

func storedOutcome() *pb.TaskOutcome {
	return &pb.TaskOutcome{Id: "o1", JobTaskId: "jt1", CriteriaVersionId: "cr1", NumericValue: fp(3), DeterminationNote: sp("level 3"), DateModified: ip(1000)}
}

func TestUpdateIfUnchanged_SuccessGuardsAndPassesSnapshot(t *testing.T) {
	repo := &condRepo{plainRepo: plainRepo{stored: storedOutcome()}}
	us, _ := svcs(true)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	resp, conflict, err := uc.Execute(context.Background(),
		&pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(5), DeterminationNote: sp("level 5")}},
		storedOutcome())
	if err != nil || conflict || resp == nil {
		t.Fatalf("want success, got resp=%v conflict=%v err=%v", resp, conflict, err)
	}
	if len(repo.guardedTasks) != 1 || repo.guardedTasks[0] != "jt1" {
		t.Fatalf("cell-write guard must run on the existing owner's task, got %v", repo.guardedTasks)
	}
	if repo.updates != 0 || repo.condUpdates != 1 {
		t.Fatalf("must use the conditional writer only (plain=%d cond=%d)", repo.updates, repo.condUpdates)
	}
}

func TestUpdateIfUnchanged_ConflictReportsConflictAndWritesNothing(t *testing.T) {
	repo := &condRepo{plainRepo: plainRepo{stored: storedOutcome()}, storedChanged: true}
	us, _ := svcs(true)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	resp, conflict, err := uc.Execute(context.Background(),
		&pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(0)}}, storedOutcome())
	if err != nil || !conflict || resp != nil {
		t.Fatalf("want conflict, got resp=%v conflict=%v err=%v", resp, conflict, err)
	}
	if repo.lastWritten != nil || repo.updates != 0 {
		t.Fatal("a conflict must not write")
	}
}

func TestUpdateIfUnchanged_GoneRowIsConflict(t *testing.T) {
	repo := &condRepo{}
	us, _ := svcs(true)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	_, conflict, err := uc.Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, storedOutcome())
	if err != nil || !conflict || repo.condUpdates != 0 {
		t.Fatalf("a row gone since the snapshot must be a conflict without writing: conflict=%v err=%v", conflict, err)
	}
}

func TestUpdateIfUnchanged_GuardErrorIsError(t *testing.T) {
	repo := &condRepo{plainRepo: plainRepo{stored: storedOutcome()}, guardErr: errors.New("phase locked")}
	us, _ := svcs(true)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	_, conflict, err := uc.Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, storedOutcome())
	if err == nil || conflict || repo.condUpdates != 0 {
		t.Fatalf("guard failure must be an error before any write: conflict=%v err=%v", conflict, err)
	}
}

func TestConditionalWrites_TransactionalRepoWithoutPortFailsClosed(t *testing.T) {
	repo := &plainRepo{stored: storedOutcome()}
	us, cs := svcs(true)
	_, _, err := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us).
		Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, storedOutcome())
	if err == nil || repo.updates != 0 {
		t.Fatalf("update: transactional provider without the port must fail closed (err=%v updates=%d)", err, repo.updates)
	}
	_, _, err = NewCreateTaskOutcomeIfAbsentUseCase(CreateTaskOutcomeRepositories{TaskOutcome: repo}, cs).
		Execute(context.Background(), &pb.CreateTaskOutcomeRequest{Data: &pb.TaskOutcome{JobTaskId: "jt1", CriteriaVersionId: "cr1"}})
	if err == nil || repo.creates != 0 {
		t.Fatalf("create: transactional provider without the port must fail closed (err=%v creates=%d)", err, repo.creates)
	}
}

func TestUpdateIfUnchanged_NonTransactionalBestEffortComparesSnapshot(t *testing.T) {
	repo := &plainRepo{stored: storedOutcome()}
	us, _ := svcs(false)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	stale := storedOutcome()
	stale.NumericValue = fp(7)
	if _, conflict, err := uc.Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, stale); err != nil || !conflict || repo.updates != 0 {
		t.Fatalf("stale snapshot must conflict without writing: conflict=%v err=%v updates=%d", conflict, err, repo.updates)
	}
	if _, conflict, err := uc.Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, storedOutcome()); err != nil || conflict || repo.updates != 1 {
		t.Fatalf("matching snapshot must write: conflict=%v err=%v updates=%d", conflict, err, repo.updates)
	}
}

func TestCreateIfAbsent_SecondCreateForSameCellConflicts(t *testing.T) {
	repo := &condRepo{}
	_, cs := svcs(true)
	uc := NewCreateTaskOutcomeIfAbsentUseCase(CreateTaskOutcomeRepositories{TaskOutcome: repo}, cs)
	req := func() *pb.CreateTaskOutcomeRequest {
		return &pb.CreateTaskOutcomeRequest{Data: &pb.TaskOutcome{JobTaskId: "jt1", CriteriaVersionId: "cr1", NumericValue: fp(4)}}
	}
	resp, conflict, err := uc.Execute(context.Background(), req())
	if err != nil || conflict || resp == nil || resp.GetData()[0].GetId() != "new-id" {
		t.Fatalf("first create must succeed with a generated id: resp=%v conflict=%v err=%v", resp, conflict, err)
	}
	resp, conflict, err = uc.Execute(context.Background(), req())
	if err != nil || !conflict || resp != nil {
		t.Fatalf("second create for the same cell must conflict: resp=%v conflict=%v err=%v", resp, conflict, err)
	}
	if len(repo.guardedTasks) != 2 || repo.creates != 0 || repo.condCreates != 2 {
		t.Fatalf("both attempts must be guarded and go through the conditional writer (guards=%v plain=%d cond=%d)", repo.guardedTasks, repo.creates, repo.condCreates)
	}
}

// fix3-backend (codex-review-impl3 #7): the non-transactional fallback's
// snapshot comparison must cover every typed value column the postgres
// predicate compares, not just numeric/note/timestamp.
func TestSnapshotMatches_TypedNonNumericSameTimestamp(t *testing.T) {
	bp := func(v bool) *bool { return &v }
	base := func() *pb.TaskOutcome {
		o := storedOutcome()
		o.TextValue = sp("t")
		o.CategoricalValue = sp("c")
		o.PassFailValue = bp(true)
		return o
	}
	if !SnapshotMatches(base(), base()) {
		t.Fatal("identical snapshots must match")
	}
	cases := map[string]func(o *pb.TaskOutcome){
		"text changed":          func(o *pb.TaskOutcome) { o.TextValue = sp("t2") },
		"text cleared":          func(o *pb.TaskOutcome) { o.TextValue = nil },
		"categorical changed":   func(o *pb.TaskOutcome) { o.CategoricalValue = sp("c2") },
		"categorical cleared":   func(o *pb.TaskOutcome) { o.CategoricalValue = nil },
		"pass_fail flipped":     func(o *pb.TaskOutcome) { o.PassFailValue = bp(false) },
		"pass_fail cleared":     func(o *pb.TaskOutcome) { o.PassFailValue = nil },
		"numeric changed":       func(o *pb.TaskOutcome) { o.NumericValue = fp(4) },
		"note changed":          func(o *pb.TaskOutcome) { o.DeterminationNote = sp("other") },
		"date_modified changed": func(o *pb.TaskOutcome) { o.DateModified = ip(1001) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cur := base()
			mutate(cur) // same timestamp unless the case mutates it
			if SnapshotMatches(cur, base()) {
				t.Fatalf("%s: stale snapshot must NOT match", name)
			}
			if SnapshotMatches(base(), cur) {
				t.Fatalf("%s: comparison must be symmetric", name)
			}
		})
	}
}

func TestUpdateIfUnchanged_NonTransactionalTypedConflictWritesNothing(t *testing.T) {
	stored := storedOutcome()
	stored.TextValue = sp("current")
	repo := &plainRepo{stored: stored}
	us, _ := svcs(false)
	uc := NewUpdateTaskOutcomeIfUnchangedUseCase(UpdateTaskOutcomeRepositories{TaskOutcome: repo}, us)
	stale := storedOutcome()
	stale.TextValue = sp("stale") // same numeric/note/date_modified
	if _, conflict, err := uc.Execute(context.Background(), &pb.UpdateTaskOutcomeRequest{Data: &pb.TaskOutcome{Id: "o1", NumericValue: fp(1)}}, stale); err != nil || !conflict || repo.updates != 0 {
		t.Fatalf("typed stale snapshot must conflict without writing: conflict=%v err=%v updates=%d", conflict, err, repo.updates)
	}
}
