package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	commonpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/common"
	clientpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client"
	clientattributepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/client_attribute"
	userpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/entity/user"
)

// --- helpers ---------------------------------------------------------------

func newTestGate() *actiongate.ActionGatekeeper {
	// NoOp authorizer => IsEnabled()==false => the gate permits every action.
	return actiongate.NewActionGatekeeper(ports.NewNoOpAuthorizer(), ports.NewNoOpTranslator())
}

func p32(v int32) *int32       { return &v }
func pf64(v float64) *float64  { return &v }
func pbool(v bool) *bool       { return &v }

// defn builds a client-APPLICABLE active definition (module "entity", which
// isApplicableAttributeModule accepts). Tests that need a disallowed module set
// Module explicitly (see the forged-module cases).
func defn(id, code, dataType string, active bool) *commonpb.Attribute {
	return &commonpb.Attribute{Id: id, Code: code, DataType: dataType, Active: active, Module: "entity"}
}

// --- fake repos ------------------------------------------------------------

type fakeAttributeRepo struct {
	commonpb.UnimplementedAttributeDomainServiceServer
	defs []*commonpb.Attribute
	err  error
}

func (f *fakeAttributeRepo) ListAttributes(ctx context.Context, req *commonpb.ListAttributesRequest) (*commonpb.ListAttributesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &commonpb.ListAttributesResponse{Data: f.defs}, nil
}

type fakeAttributeValueRepo struct {
	commonpb.UnimplementedAttributeValueDomainServiceServer
	byAttr map[string][]*commonpb.AttributeValue
}

func (f *fakeAttributeValueRepo) ListAttributeValues(ctx context.Context, req *commonpb.ListAttributeValuesRequest) (*commonpb.ListAttributeValuesResponse, error) {
	// Extract attribute_id filter.
	attrID := ""
	if req.GetFilters() != nil {
		for _, tf := range req.GetFilters().GetFilters() {
			if tf.GetField() == "attribute_id" {
				attrID = tf.GetStringFilter().GetValue()
			}
		}
	}
	return &commonpb.ListAttributeValuesResponse{Data: f.byAttr[attrID]}, nil
}

type fakeClientAttributeRepo struct {
	clientattributepb.UnimplementedClientAttributeDomainServiceServer
	existing []*clientattributepb.ClientAttribute
	created  []*clientattributepb.ClientAttribute
	updated  []*clientattributepb.ClientAttribute
	deleted  []string
}

func (f *fakeClientAttributeRepo) ListClientAttributes(ctx context.Context, req *clientattributepb.ListClientAttributesRequest) (*clientattributepb.ListClientAttributesResponse, error) {
	return &clientattributepb.ListClientAttributesResponse{Data: f.existing}, nil
}
func (f *fakeClientAttributeRepo) CreateClientAttribute(ctx context.Context, req *clientattributepb.CreateClientAttributeRequest) (*clientattributepb.CreateClientAttributeResponse, error) {
	f.created = append(f.created, req.GetData())
	return &clientattributepb.CreateClientAttributeResponse{Data: []*clientattributepb.ClientAttribute{req.GetData()}}, nil
}
func (f *fakeClientAttributeRepo) UpdateClientAttribute(ctx context.Context, req *clientattributepb.UpdateClientAttributeRequest) (*clientattributepb.UpdateClientAttributeResponse, error) {
	f.updated = append(f.updated, req.GetData())
	return &clientattributepb.UpdateClientAttributeResponse{Data: []*clientattributepb.ClientAttribute{req.GetData()}}, nil
}
func (f *fakeClientAttributeRepo) DeleteClientAttribute(ctx context.Context, req *clientattributepb.DeleteClientAttributeRequest) (*clientattributepb.DeleteClientAttributeResponse, error) {
	f.deleted = append(f.deleted, req.GetData().GetId())
	return &clientattributepb.DeleteClientAttributeResponse{Success: true}, nil
}

type fakeClientRepo struct {
	clientpb.UnimplementedClientDomainServiceServer
	created *clientpb.Client
	updated *clientpb.Client
	read    *clientpb.Client
}

func (f *fakeClientRepo) CreateClient(ctx context.Context, req *clientpb.CreateClientRequest) (*clientpb.CreateClientResponse, error) {
	f.created = req.GetData()
	return &clientpb.CreateClientResponse{Data: []*clientpb.Client{req.GetData()}}, nil
}
func (f *fakeClientRepo) UpdateClient(ctx context.Context, req *clientpb.UpdateClientRequest) (*clientpb.UpdateClientResponse, error) {
	f.updated = req.GetData()
	return &clientpb.UpdateClientResponse{Data: []*clientpb.Client{req.GetData()}}, nil
}
func (f *fakeClientRepo) ReadClient(ctx context.Context, req *clientpb.ReadClientRequest) (*clientpb.ReadClientResponse, error) {
	if f.read == nil {
		return &clientpb.ReadClientResponse{}, nil
	}
	return &clientpb.ReadClientResponse{Data: []*clientpb.Client{f.read}}, nil
}

type seqIDGen struct{ n int }

func (s *seqIDGen) GenerateID() string { s.n++; return fmt.Sprintf("ca-%d", s.n) }
func (s *seqIDGen) GenerateIDWithPrefix(prefix string) string {
	s.n++
	return fmt.Sprintf("%s-%d", prefix, s.n)
}
func (s *seqIDGen) IsEnabled() bool        { return true }
func (s *seqIDGen) GetProviderInfo() string { return "seq-test" }

// --- classifyAttributeControl ----------------------------------------------

func TestClassifyAttributeControl(t *testing.T) {
	cases := []struct {
		dataType string
		control  string
		integer  bool
		ok       bool
	}{
		{"text", "text", false, true},
		{"string", "text", false, true},
		{"free_text", "text", false, true},
		{"integer", "number", true, true},
		{"number", "number", false, true},
		{"decimal", "number", false, true},
		{"free_number", "number", false, true},
		{"option", "select", false, true},
		{"enum", "select", false, true},
		{"text_list", "select", false, true},
		{"boolean", "boolean", false, true},
		{"date", "date", false, true},
		{"  OPTION ", "select", false, true}, // trim + case-insensitive
		{"geojson", "", false, false},        // unknown => fail-closed
		{"", "", false, false},
	}
	for _, c := range cases {
		got, ok := classifyAttributeControl(c.dataType)
		if ok != c.ok {
			t.Errorf("classify(%q) ok=%v want %v", c.dataType, ok, c.ok)
			continue
		}
		if ok && (got.control != c.control || got.integer != c.integer) {
			t.Errorf("classify(%q)=%+v want control=%s integer=%v", c.dataType, got, c.control, c.integer)
		}
	}
}

// --- validateAttributeValue ------------------------------------------------

func TestValidateAttributeValue(t *testing.T) {
	enum := map[string]bool{"male": true, "female": true}
	cases := []struct {
		name    string
		def     *commonpb.Attribute
		kind    attrControl
		value   string
		enum    map[string]bool
		want    string
		wantErr bool
	}{
		{"text ok", &commonpb.Attribute{Code: "lrn"}, attrControl{control: "text"}, "12345", nil, "12345", false},
		{"text min length fail", &commonpb.Attribute{Code: "lrn", MinLength: p32(3)}, attrControl{control: "text"}, "ab", nil, "", true},
		{"text max length fail", &commonpb.Attribute{Code: "lrn", MaxLength: p32(4)}, attrControl{control: "text"}, "abcde", nil, "", true},
		{"text max length rune-count ok", &commonpb.Attribute{Code: "n", MaxLength: p32(2)}, attrControl{control: "text"}, "áé", nil, "áé", false}, // 2 runes, 4 bytes
		{"number ok", &commonpb.Attribute{Code: "age"}, attrControl{control: "number"}, "21", nil, "21", false},
		{"number non-numeric fail", &commonpb.Attribute{Code: "age"}, attrControl{control: "number"}, "xx", nil, "", true},
		// W3-HIGH-3: strconv.ParseFloat accepts NaN/Inf/-Inf; a bounds-only check
		// leaves both < min and > max false, so these MUST be rejected explicitly.
		{"number NaN rejected", &commonpb.Attribute{Code: "age"}, attrControl{control: "number"}, "NaN", nil, "", true},
		{"number +Inf rejected", &commonpb.Attribute{Code: "age", MaxValue: pf64(120)}, attrControl{control: "number"}, "Inf", nil, "", true},
		{"number -Inf rejected", &commonpb.Attribute{Code: "age", MinValue: pf64(1)}, attrControl{control: "number"}, "-Inf", nil, "", true},
		{"number Infinity word rejected", &commonpb.Attribute{Code: "age"}, attrControl{control: "number"}, "Infinity", nil, "", true},
		{"integer non-integral fail", &commonpb.Attribute{Code: "age"}, attrControl{control: "number", integer: true}, "1.5", nil, "", true},
		{"number below min fail", &commonpb.Attribute{Code: "age", MinValue: pf64(10)}, attrControl{control: "number"}, "9", nil, "", true},
		{"number above max fail", &commonpb.Attribute{Code: "age", MaxValue: pf64(10)}, attrControl{control: "number"}, "11", nil, "", true},
		{"number in range ok", &commonpb.Attribute{Code: "age", MinValue: pf64(1), MaxValue: pf64(120)}, attrControl{control: "number"}, "30", nil, "30", false},
		{"select member ok", &commonpb.Attribute{Code: "gender"}, attrControl{control: "select"}, "male", enum, "male", false},
		{"select non-member fail", &commonpb.Attribute{Code: "gender"}, attrControl{control: "select"}, "banana", enum, "", true},
		// W3-MED-7: boolean accepts ONLY the canonical true/false the drawer emits;
		// the former yes/no/1/0/on/off superset was a browser/server divergence.
		{"boolean true canonical ok", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "true", nil, "true", false},
		{"boolean false canonical ok", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "false", nil, "false", false},
		{"boolean rejects yes alias", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "Yes", nil, "", true},
		{"boolean rejects 0 alias", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "0", nil, "", true},
		{"boolean rejects on alias", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "on", nil, "", true},
		{"boolean invalid fail", &commonpb.Attribute{Code: "b"}, attrControl{control: "boolean"}, "maybe", nil, "", true},
		{"date ok", &commonpb.Attribute{Code: "d"}, attrControl{control: "date"}, "2026-07-17", nil, "2026-07-17", false},
		{"date invalid fail", &commonpb.Attribute{Code: "d"}, attrControl{control: "date"}, "17/07/2026", nil, "", true},
		{"blank optional clears", &commonpb.Attribute{Code: "gender"}, attrControl{control: "select"}, "", enum, "", false},
		{"blank required fail", &commonpb.Attribute{Code: "gender", Required: pbool(true)}, attrControl{control: "select"}, "", enum, "", true},
	}
	for _, c := range cases {
		got, err := validateAttributeValue(c.def, c.kind, c.value, c.enum)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v wantErr=%v", c.name, err, c.wantErr)
			continue
		}
		if !c.wantErr && got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

// --- resolveAndValidateAttributes ------------------------------------------

func newAttrRepos(defs []*commonpb.Attribute, values map[string][]*commonpb.AttributeValue, existing []*clientattributepb.ClientAttribute) (AttributeRepositories, *fakeClientAttributeRepo) {
	car := &fakeClientAttributeRepo{existing: existing}
	return AttributeRepositories{
		Attribute:       &fakeAttributeRepo{defs: defs},
		AttributeValue:  &fakeAttributeValueRepo{byAttr: values},
		ClientAttribute: car,
	}, car
}

func TestResolveAndValidateAttributes(t *testing.T) {
	genderDefs := []*commonpb.Attribute{defn("attr-gender", "gender", "option", true)}
	genderVals := map[string][]*commonpb.AttributeValue{
		"attr-gender": {
			{Id: "v1", AttributeId: "attr-gender", Value: "male", Active: true},
			{Id: "v2", AttributeId: "attr-gender", Value: "female", Active: true},
		},
	}
	ctx := context.Background()

	t.Run("valid select resolves", func(t *testing.T) {
		repos, _ := newAttrRepos(genderDefs, genderVals, nil)
		got, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}})
		if err != nil || len(got) != 1 || got[0].attributeID != "attr-gender" || got[0].value != "male" {
			t.Fatalf("got %+v err %v", got, err)
		}
	})

	t.Run("inactive definition rejected", func(t *testing.T) {
		repos, _ := newAttrRepos([]*commonpb.Attribute{defn("attr-gender", "gender", "option", false)}, genderVals, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}}); err == nil {
			t.Fatal("expected error for inactive/unknown definition")
		}
	})

	t.Run("unknown code rejected", func(t *testing.T) {
		repos, _ := newAttrRepos(genderDefs, genderVals, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "lrn", Value: "x"}}); err == nil {
			t.Fatal("expected error for unknown code")
		}
	})

	t.Run("inactive enum option rejected", func(t *testing.T) {
		vals := map[string][]*commonpb.AttributeValue{
			"attr-gender": {{Id: "v1", AttributeId: "attr-gender", Value: "male", Active: false}},
		}
		repos, _ := newAttrRepos(genderDefs, vals, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}}); err == nil {
			t.Fatal("expected error: inactive option is not a member")
		}
	})

	t.Run("unknown data_type rejected", func(t *testing.T) {
		repos, _ := newAttrRepos([]*commonpb.Attribute{defn("a1", "weird", "geojson", true)}, nil, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "weird", Value: "x"}}); err == nil {
			t.Fatal("expected error for unknown data_type (fail-closed)")
		}
	})

	t.Run("ambiguous code rejected", func(t *testing.T) {
		dups := []*commonpb.Attribute{
			defn("a1", "gender", "option", true),
			defn("a2", "gender", "option", true),
		}
		repos, _ := newAttrRepos(dups, genderVals, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}}); err == nil {
			t.Fatal("expected ambiguity error for duplicate active codes")
		}
	})

	t.Run("duplicate submission rejected", func(t *testing.T) {
		repos, _ := newAttrRepos(genderDefs, genderVals, nil)
		_, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{
			{Code: "gender", Value: "male"}, {Code: "gender", Value: "female"},
		})
		if err == nil {
			t.Fatal("expected error for the same code submitted twice")
		}
	})

	// W3-HIGH-2: an ACTIVE definition whose module is outside {entity,client,general}
	// (e.g. "product") is NOT client-applicable; a forged POST carrying its code must
	// be rejected as "not an active definition", never written to client_attribute.
	t.Run("forged disallowed-module code rejected", func(t *testing.T) {
		prod := []*commonpb.Attribute{
			{Id: "attr-color", Code: "color", DataType: "text", Module: "product", Active: true},
		}
		repos, car := newAttrRepos(prod, nil, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "color", Value: "red"}}); err == nil {
			t.Fatal("expected rejection: product-module code is not client-applicable")
		}
		if len(car.created) != 0 || len(car.updated) != 0 {
			t.Fatalf("no client_attribute write on forged-module rejection: created=%d updated=%d", len(car.created), len(car.updated))
		}
	})

	// W3-MED-5: a REQUIRED applicable definition OMITTED entirely from the submission
	// is rejected even though the submission itself carried no bad value — browser
	// `required` is backed by a server guarantee. The sweep must run for the empty
	// submission too (nil attrs).
	t.Run("required applicable attribute omitted is rejected", func(t *testing.T) {
		reqDef := []*commonpb.Attribute{
			{Id: "attr-lrn", Code: "lrn", DataType: "text", Module: "entity", Active: true, Required: pbool(true)},
		}
		repos, _ := newAttrRepos(reqDef, nil, nil)
		if _, err := resolveAndValidateAttributes(ctx, repos, nil); err == nil {
			t.Fatal("expected error: required attribute omitted from a present section")
		}
		// A non-blank submission for it succeeds.
		if _, err := resolveAndValidateAttributes(ctx, repos, []*commonpb.AttributeCodeValue{{Code: "lrn", Value: "123456"}}); err != nil {
			t.Fatalf("required attribute supplied should resolve: %v", err)
		}
	})
}

// --- syncClientAttributes --------------------------------------------------

func TestSyncClientAttributes(t *testing.T) {
	ctx := context.Background()

	t.Run("create new row", func(t *testing.T) {
		car := &fakeClientAttributeRepo{}
		repos := AttributeRepositories{ClientAttribute: car}
		err := syncClientAttributes(ctx, repos, &seqIDGen{}, "c1", []resolvedAttribute{{attributeID: "attr-gender", value: "male"}})
		if err != nil || len(car.created) != 1 || car.created[0].GetValue() != "male" || car.created[0].GetClientId() != "c1" {
			t.Fatalf("created=%+v err=%v", car.created, err)
		}
	})

	t.Run("update changed row", func(t *testing.T) {
		car := &fakeClientAttributeRepo{existing: []*clientattributepb.ClientAttribute{
			{Id: "row1", ClientId: "c1", AttributeId: "attr-gender", Value: "male", Active: true},
		}}
		repos := AttributeRepositories{ClientAttribute: car}
		err := syncClientAttributes(ctx, repos, &seqIDGen{}, "c1", []resolvedAttribute{{attributeID: "attr-gender", value: "female"}})
		if err != nil || len(car.updated) != 1 || car.updated[0].GetValue() != "female" || len(car.created) != 0 {
			t.Fatalf("updated=%+v created=%+v err=%v", car.updated, car.created, err)
		}
	})

	t.Run("unchanged row is a no-op", func(t *testing.T) {
		car := &fakeClientAttributeRepo{existing: []*clientattributepb.ClientAttribute{
			{Id: "row1", ClientId: "c1", AttributeId: "attr-gender", Value: "male", Active: true},
		}}
		repos := AttributeRepositories{ClientAttribute: car}
		err := syncClientAttributes(ctx, repos, &seqIDGen{}, "c1", []resolvedAttribute{{attributeID: "attr-gender", value: "male"}})
		if err != nil || len(car.updated) != 0 || len(car.created) != 0 || len(car.deleted) != 0 {
			t.Fatalf("expected no-op, got updated=%d created=%d deleted=%d err=%v", len(car.updated), len(car.created), len(car.deleted), err)
		}
	})

	t.Run("blank value clears existing row", func(t *testing.T) {
		car := &fakeClientAttributeRepo{existing: []*clientattributepb.ClientAttribute{
			{Id: "row1", ClientId: "c1", AttributeId: "attr-gender", Value: "male", Active: true},
		}}
		repos := AttributeRepositories{ClientAttribute: car}
		err := syncClientAttributes(ctx, repos, &seqIDGen{}, "c1", []resolvedAttribute{{attributeID: "attr-gender", value: ""}})
		if err != nil || len(car.deleted) != 1 || car.deleted[0] != "row1" {
			t.Fatalf("deleted=%+v err=%v", car.deleted, err)
		}
	})

	t.Run("blank value with no existing row is a no-op", func(t *testing.T) {
		car := &fakeClientAttributeRepo{}
		repos := AttributeRepositories{ClientAttribute: car}
		err := syncClientAttributes(ctx, repos, &seqIDGen{}, "c1", []resolvedAttribute{{attributeID: "attr-gender", value: ""}})
		if err != nil || len(car.deleted) != 0 || len(car.created) != 0 {
			t.Fatalf("expected no-op, err=%v", err)
		}
	})
}

// --- Execute-level: create + update integration ----------------------------

func permissiveCreateUC(clientRepo *fakeClientRepo, attrs AttributeRepositories) *CreateClientUseCase {
	return NewCreateClientUseCase(
		CreateClientRepositories{Client: clientRepo, Attributes: attrs},
		CreateClientServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: newTestGate(),
			IDGenerator:      &seqIDGen{},
		},
	)
}

func validClientData() *clientpb.Client {
	return &clientpb.Client{
		User: &userpb.User{FirstName: "Ada", LastName: "Lovelace", EmailAddress: "ada@example.com"},
	}
}

func genderReposForExec() (AttributeRepositories, *fakeClientAttributeRepo) {
	defs := []*commonpb.Attribute{defn("attr-gender", "gender", "option", true)}
	vals := map[string][]*commonpb.AttributeValue{
		"attr-gender": {
			{Id: "v1", AttributeId: "attr-gender", Value: "male", Active: true},
			{Id: "v2", AttributeId: "attr-gender", Value: "female", Active: true},
		},
	}
	return newAttrRepos(defs, vals, nil)
}

func TestCreateClient_WithValidAttribute(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{}
	attrs, car := genderReposForExec()
	uc := permissiveCreateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.CreateClientRequest{
		Data:       validClientData(),
		Attributes: []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientRepo.created == nil {
		t.Fatal("client was not created")
	}
	if len(car.created) != 1 || car.created[0].GetValue() != "male" {
		t.Fatalf("client_attribute not persisted: %+v", car.created)
	}
}

func TestCreateClient_InvalidAttribute_RollsBackBeforeWrite(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{}
	attrs, car := genderReposForExec()
	uc := permissiveCreateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.CreateClientRequest{
		Data:       validClientData(),
		Attributes: []*commonpb.AttributeCodeValue{{Code: "gender", Value: "banana"}},
	})
	if err == nil {
		t.Fatal("expected validation error for enum non-member")
	}
	if clientRepo.created != nil {
		t.Fatal("client must NOT be created when attribute validation fails (atomicity)")
	}
	if len(car.created) != 0 {
		t.Fatal("no client_attribute should be written on validation failure")
	}
}

func TestCreateClient_AttributesSubmittedButPipelineUnwired_FailsClosed(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{}
	uc := permissiveCreateUC(clientRepo, AttributeRepositories{}) // not wired

	_, err := uc.Execute(ctx, &clientpb.CreateClientRequest{
		Data:       validClientData(),
		Attributes: []*commonpb.AttributeCodeValue{{Code: "gender", Value: "male"}},
	})
	if err == nil {
		t.Fatal("expected fail-closed error when attributes submitted but repos unwired")
	}
	if clientRepo.created != nil {
		t.Fatal("client must not be created when the pipeline is unwired")
	}
}

func permissiveUpdateUC(clientRepo *fakeClientRepo, attrs AttributeRepositories) *UpdateClientUseCase {
	return NewUpdateClientUseCase(
		UpdateClientRepositories{Client: clientRepo, Attributes: attrs},
		UpdateClientServices{
			Authorizer:       ports.NewNoOpAuthorizer(),
			Transactor:       ports.NewNoOpTransactor(),
			Translator:       ports.NewNoOpTranslator(),
			ActionGatekeeper: newTestGate(),
			IDGenerator:      &seqIDGen{},
		},
	)
}

func TestUpdateClient_OmittedSection_NeverWipesAttributes(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{read: &clientpb.Client{Id: "c1", Active: true}}
	attrs, car := genderReposForExec()
	// Pre-existing gender row that MUST survive an update whose section is absent.
	car.existing = []*clientattributepb.ClientAttribute{
		{Id: "row1", ClientId: "c1", AttributeId: "attr-gender", Value: "male", Active: true},
	}
	uc := permissiveUpdateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.UpdateClientRequest{
		Data:              &clientpb.Client{Id: "c1", Active: true},
		AttributesPresent: false, // section omitted
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(car.deleted) != 0 || len(car.updated) != 0 || len(car.created) != 0 {
		t.Fatalf("omitted section must not touch attributes: deleted=%d updated=%d created=%d", len(car.deleted), len(car.updated), len(car.created))
	}
}

func TestUpdateClient_PresentSection_BlankClears(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{read: &clientpb.Client{Id: "c1", Active: true}}
	attrs, car := genderReposForExec()
	car.existing = []*clientattributepb.ClientAttribute{
		{Id: "row1", ClientId: "c1", AttributeId: "attr-gender", Value: "male", Active: true},
	}
	uc := permissiveUpdateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.UpdateClientRequest{
		Data:              &clientpb.Client{Id: "c1", Active: true},
		AttributesPresent: true,
		Attributes:        []*commonpb.AttributeCodeValue{{Code: "gender", Value: ""}}, // cleared
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(car.deleted) != 1 || car.deleted[0] != "row1" {
		t.Fatalf("blank present value must clear the row: deleted=%+v", car.deleted)
	}
}

// productModuleReposForExec wires a single ACTIVE product-module definition — a
// code a forged POST could carry but the client path must refuse (W3-HIGH-2).
func productModuleReposForExec() (AttributeRepositories, *fakeClientAttributeRepo) {
	defs := []*commonpb.Attribute{
		{Id: "attr-color", Code: "color", DataType: "text", Module: "product", Active: true},
	}
	return newAttrRepos(defs, nil, nil)
}

// requiredTextReposForExec wires a single REQUIRED entity-module text definition
// (W3-MED-5) so an omitted submission must be rejected.
func requiredTextReposForExec() (AttributeRepositories, *fakeClientAttributeRepo) {
	defs := []*commonpb.Attribute{
		{Id: "attr-lrn", Code: "lrn", DataType: "text", Module: "entity", Active: true, Required: pbool(true)},
	}
	return newAttrRepos(defs, nil, nil)
}

func TestCreateClient_ForgedDisallowedModule_RejectedNoWrite(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{}
	attrs, car := productModuleReposForExec()
	uc := permissiveCreateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.CreateClientRequest{
		Data:              validClientData(),
		AttributesPresent: true,
		Attributes:        []*commonpb.AttributeCodeValue{{Code: "color", Value: "red"}}, // forged product-module code
	})
	if err == nil {
		t.Fatal("expected rejection of a disallowed-module attribute code")
	}
	if clientRepo.created != nil {
		t.Fatal("client must NOT be created when a forged-module attribute is submitted")
	}
	if len(car.created) != 0 || len(car.updated) != 0 {
		t.Fatalf("no client_attribute write on forged-module rejection: created=%d updated=%d", len(car.created), len(car.updated))
	}
}

func TestCreateClient_RequiredAttributeOmitted_Rejected(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{}
	attrs, car := requiredTextReposForExec()
	uc := permissiveCreateUC(clientRepo, attrs)

	// Section present (marker) but the required "lrn" is omitted entirely.
	_, err := uc.Execute(ctx, &clientpb.CreateClientRequest{
		Data:              validClientData(),
		AttributesPresent: true,
	})
	if err == nil {
		t.Fatal("expected rejection: required attribute omitted from a present section")
	}
	if clientRepo.created != nil {
		t.Fatal("client must NOT be created when a required attribute is missing")
	}
	if len(car.created) != 0 {
		t.Fatal("no client_attribute should be written when required coverage fails")
	}
}

func TestUpdateClient_RequiredAttributeOmitted_Rejected(t *testing.T) {
	ctx := context.Background()
	clientRepo := &fakeClientRepo{read: &clientpb.Client{Id: "c1", Active: true}}
	attrs, car := requiredTextReposForExec()
	uc := permissiveUpdateUC(clientRepo, attrs)

	_, err := uc.Execute(ctx, &clientpb.UpdateClientRequest{
		Data:              &clientpb.Client{Id: "c1", Active: true},
		AttributesPresent: true, // section present, required "lrn" omitted
	})
	if err == nil {
		t.Fatal("expected rejection: required attribute omitted on update")
	}
	if len(car.created) != 0 || len(car.updated) != 0 || len(car.deleted) != 0 {
		t.Fatalf("no client_attribute mutation on required-coverage failure: created=%d updated=%d deleted=%d", len(car.created), len(car.updated), len(car.deleted))
	}
}
