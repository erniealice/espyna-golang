package firestore

import (
	"context"
	"fmt"
	"sort"
	"strings"

	cloudfirestore "cloud.google.com/go/firestore"
	"github.com/erniealice/espyna-golang/ports"
	"github.com/erniealice/espyna-golang/registry"
	"github.com/erniealice/espyna-golang/registry/entityid"
	"github.com/erniealice/espyna-golang/shared/identity"
	exportpb "github.com/erniealice/esqyma/pkg/schema/v1/service/operation/subscription_group_outcome_export"
)

const (
	landingPrincipalTypeStaff = int32(7)
	landingOriginSubscription = "ORIGIN_TYPE_SUBSCRIPTION"
)

func init() {
	registry.RegisterSubscriptionGroupOutcomeLandingFactory(func(input registry.SubscriptionGroupOutcomeLandingFactoryInput) any {
		client, ok := input.Connection.(*cloudfirestore.Client)
		if !ok || client == nil {
			return nil
		}
		return newFirestoreSubscriptionGroupOutcomeLandingQuery(client, input.TableConfig)
	})
}

// firestoreSubscriptionGroupOutcomeLandingQuery is the Firestore implementation
// of the same non-PII landing projection served by PostgreSQL. Firestore has no
// cross-collection join, so it loads workspace-bounded documents and performs
// the canonical joins in memory before returning aggregate rows.
type firestoreSubscriptionGroupOutcomeLandingQuery struct {
	client *cloudfirestore.Client
	tables *registry.TableConfig
}

func newFirestoreSubscriptionGroupOutcomeLandingQuery(client *cloudfirestore.Client, tables *registry.TableConfig) *firestoreSubscriptionGroupOutcomeLandingQuery {
	if tables == nil {
		tables = registry.NewDefaultTableConfig()
	}
	return &firestoreSubscriptionGroupOutcomeLandingQuery{client: client, tables: tables}
}

func (q *firestoreSubscriptionGroupOutcomeLandingQuery) ListSubscriptionGroupOutcomeLandingScoped(
	ctx context.Context,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) (*exportpb.ListSubscriptionGroupOutcomeLandingResponse, error) {
	empty := func() *exportpb.ListSubscriptionGroupOutcomeLandingResponse {
		return &exportpb.ListSubscriptionGroupOutcomeLandingResponse{
			Success: true,
			Rows:    []*exportpb.SubscriptionGroupOutcomeLandingRow{},
		}
	}
	if req == nil {
		return nil, fmt.Errorf("subscription group outcome landing request is required")
	}
	if q == nil || q.client == nil {
		return nil, fmt.Errorf("subscription group outcome landing requires Firestore")
	}
	requestIdentity, ok := identity.FromContext(ctx)
	if !ok || requestIdentity == nil || strings.TrimSpace(requestIdentity.WorkspaceID) == "" {
		return empty(), nil
	}

	snapshot, err := q.loadSnapshot(ctx, requestIdentity.WorkspaceID, !scope.WorkspaceWide && requestIdentity.PrincipalType == landingPrincipalTypeStaff)
	if err != nil {
		return nil, err
	}
	rows := projectFirestoreOutcomeLanding(snapshot, requestIdentity, req, scope)
	return &exportpb.ListSubscriptionGroupOutcomeLandingResponse{Success: true, Rows: rows}, nil
}

type firestoreLandingDocument map[string]any

type firestoreLandingSnapshot struct {
	priceSchedules                  []firestoreLandingDocument
	subscriptionGroups              []firestoreLandingDocument
	workspaceUsers                  []firestoreLandingDocument
	subscriptionGroupWorkspaceUsers []firestoreLandingDocument
	members                         []firestoreLandingDocument
	subscriptions                   []firestoreLandingDocument
	jobs                            []firestoreLandingDocument
	jobTemplates                    []firestoreLandingDocument
	jobPhases                       []firestoreLandingDocument
	jobTasks                        []firestoreLandingDocument
	taskOutcomes                    []firestoreLandingDocument
	subscriptionSeats               []firestoreLandingDocument
	productPlans                    []firestoreLandingDocument
	classEdges                      []firestoreLandingDocument
	productPlanStaffs               []firestoreLandingDocument
}

func (q *firestoreSubscriptionGroupOutcomeLandingQuery) loadSnapshot(ctx context.Context, workspaceID string, includeStaffGraph bool) (firestoreLandingSnapshot, error) {
	var snapshot firestoreLandingSnapshot
	workspaceCollections := []struct {
		entity string
		dest   *[]firestoreLandingDocument
	}{
		{entityid.PriceSchedule, &snapshot.priceSchedules},
		{entityid.SubscriptionGroup, &snapshot.subscriptionGroups},
		{entityid.WorkspaceUser, &snapshot.workspaceUsers},
		{entityid.SubscriptionGroupWorkspaceUser, &snapshot.subscriptionGroupWorkspaceUsers},
		{entityid.SubscriptionGroupMember, &snapshot.members},
		{entityid.Subscription, &snapshot.subscriptions},
		{entityid.Job, &snapshot.jobs},
		{entityid.JobTemplate, &snapshot.jobTemplates},
	}
	if includeStaffGraph {
		workspaceCollections = append(workspaceCollections,
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.JobPhase, &snapshot.jobPhases},
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.JobTask, &snapshot.jobTasks},
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.TaskOutcome, &snapshot.taskOutcomes},
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.SubscriptionSeat, &snapshot.subscriptionSeats},
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.SubscriptionGroupProductPlanStaff, &snapshot.classEdges},
			struct {
				entity string
				dest   *[]firestoreLandingDocument
			}{entityid.ProductPlanStaff, &snapshot.productPlanStaffs},
		)
	}
	for _, collection := range workspaceCollections {
		docs, err := q.loadWorkspaceDocuments(ctx, collection.entity, workspaceID)
		if err != nil {
			return firestoreLandingSnapshot{}, err
		}
		*collection.dest = docs
	}
	if includeStaffGraph {
		docs, err := q.loadDocuments(ctx, entityid.ProductPlan)
		if err != nil {
			return firestoreLandingSnapshot{}, err
		}
		snapshot.productPlans = docs
	}
	return snapshot, nil
}

func (q *firestoreSubscriptionGroupOutcomeLandingQuery) loadWorkspaceDocuments(ctx context.Context, entity, workspaceID string) ([]firestoreLandingDocument, error) {
	query := q.client.Collection(q.tables.TableName(entity)).Where("workspace_id", "==", workspaceID)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("subscription group outcome landing read %s: %w", entity, err)
	}
	return firestoreLandingDocuments(docs), nil
}

func (q *firestoreSubscriptionGroupOutcomeLandingQuery) loadDocuments(ctx context.Context, entity string) ([]firestoreLandingDocument, error) {
	docs, err := q.client.Collection(q.tables.TableName(entity)).Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("subscription group outcome landing read %s: %w", entity, err)
	}
	return firestoreLandingDocuments(docs), nil
}

func firestoreLandingDocuments(docs []*cloudfirestore.DocumentSnapshot) []firestoreLandingDocument {
	result := make([]firestoreLandingDocument, 0, len(docs))
	for _, doc := range docs {
		if doc == nil || !doc.Exists() {
			continue
		}
		data := firestoreLandingDocument(doc.Data())
		if strings.TrimSpace(landingString(data, "id")) == "" {
			data["id"] = doc.Ref.ID
		}
		result = append(result, data)
	}
	return result
}

func projectFirestoreOutcomeLanding(
	snapshot firestoreLandingSnapshot,
	requestIdentity *identity.RequestIdentity,
	req *exportpb.ListSubscriptionGroupOutcomeLandingRequest,
	scope ports.SubscriptionGroupOutcomeExportScope,
) []*exportpb.SubscriptionGroupOutcomeLandingRow {
	if requestIdentity == nil || strings.TrimSpace(requestIdentity.WorkspaceID) == "" || req == nil {
		return []*exportpb.SubscriptionGroupOutcomeLandingRow{}
	}

	schedules := landingDocumentsByID(snapshot.priceSchedules)
	groups := landingDocumentsByID(snapshot.subscriptionGroups)
	visibleGroups := make(map[string]struct{})
	if scope.WorkspaceWide {
		for groupID := range groups {
			visibleGroups[groupID] = struct{}{}
		}
	} else {
		workspaceUserID := strings.TrimSpace(requestIdentity.WorkspaceUserID)
		workspaceUser, ok := landingDocumentsByID(snapshot.workspaceUsers)[workspaceUserID]
		if !ok || !landingBool(workspaceUser, "active") ||
			landingString(workspaceUser, "workspace_id") != requestIdentity.WorkspaceID ||
			landingString(workspaceUser, "user_id") != requestIdentity.UserID {
			return []*exportpb.SubscriptionGroupOutcomeLandingRow{}
		}
		for _, grant := range snapshot.subscriptionGroupWorkspaceUsers {
			if landingBool(grant, "active") &&
				landingString(grant, "workspace_user_id") == workspaceUserID &&
				landingString(grant, "workspace_id") == requestIdentity.WorkspaceID {
				groupID := landingString(grant, "subscription_group_id")
				if _, ok := groups[groupID]; ok {
					visibleGroups[groupID] = struct{}{}
				}
			}
		}
	}

	reachableJobs := map[string]struct{}(nil)
	if !scope.WorkspaceWide && requestIdentity.PrincipalType == landingPrincipalTypeStaff {
		reachableJobs = firestoreStaffReachableJobs(snapshot, requestIdentity.WorkspaceID, requestIdentity.PrincipalID)
	}

	subscriptions := landingDocumentsByID(snapshot.subscriptions)
	templates := landingDocumentsByID(snapshot.jobTemplates)
	membersByGroup := make(map[string][]firestoreLandingDocument)
	for _, member := range snapshot.members {
		membersByGroup[landingString(member, "subscription_group_id")] = append(membersByGroup[landingString(member, "subscription_group_id")], member)
	}
	jobsByOrigin := make(map[string][]firestoreLandingDocument)
	for _, job := range snapshot.jobs {
		if landingString(job, "origin_type") == landingOriginSubscription {
			jobsByOrigin[landingString(job, "origin_id")] = append(jobsByOrigin[landingString(job, "origin_id")], job)
		}
	}

	rows := make([]*exportpb.SubscriptionGroupOutcomeLandingRow, 0, len(visibleGroups))
	for groupID := range visibleGroups {
		group := groups[groupID]
		scheduleID := landingString(group, "price_schedule_id")
		schedule, ok := schedules[scheduleID]
		if !ok {
			continue
		}
		if req.PriceScheduleActive != nil && landingBool(schedule, "active") != *req.PriceScheduleActive {
			continue
		}
		groupActive := landingBool(group, "active")
		memberIDs := make(map[string]struct{})
		templateIDs := make(map[string]struct{})
		for _, member := range membersByGroup[groupID] {
			subscription := subscriptions[landingString(member, "subscription_id")]
			if subscription == nil || landingString(subscription, "client_id") != landingString(member, "client_id") {
				continue
			}
			if groupActive && (!landingBool(member, "active") || !landingBool(subscription, "active")) {
				continue
			}
			memberIDs[landingString(member, "id")] = struct{}{}
			for _, job := range jobsByOrigin[landingString(member, "subscription_id")] {
				if landingString(job, "client_id") != landingString(member, "client_id") {
					continue
				}
				if reachableJobs != nil {
					if _, ok := reachableJobs[landingString(job, "id")]; !ok {
						continue
					}
				}
				templateID := landingString(job, "job_template_id")
				if templateID == "" {
					continue
				}
				if groupActive {
					template := templates[templateID]
					if !landingBool(job, "active") || template == nil || !landingBool(template, "active") {
						continue
					}
				}
				templateIDs[templateID] = struct{}{}
			}
		}
		rows = append(rows, &exportpb.SubscriptionGroupOutcomeLandingRow{
			PriceScheduleId:         scheduleID,
			PriceScheduleName:       landingString(schedule, "name"),
			PriceScheduleActive:     landingBool(schedule, "active"),
			PriceScheduleSortOrder:  landingInt32Pointer(schedule, "sort_order"),
			SubscriptionGroupId:     groupID,
			SubscriptionGroupName:   landingString(group, "name"),
			SubscriptionGroupActive: groupActive,
			MemberCount:             int64(len(memberIDs)),
			JobTemplateCount:        int64(len(templateIDs)),
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if cmp := landingOptionalInt32Compare(left.PriceScheduleSortOrder, right.PriceScheduleSortOrder); cmp != 0 {
			return cmp < 0
		}
		if cmp := strings.Compare(landingSortName(left.PriceScheduleName), landingSortName(right.PriceScheduleName)); cmp != 0 {
			return cmp < 0
		}
		if left.PriceScheduleId != right.PriceScheduleId {
			return left.PriceScheduleId < right.PriceScheduleId
		}
		if cmp := strings.Compare(landingSortName(left.SubscriptionGroupName), landingSortName(right.SubscriptionGroupName)); cmp != 0 {
			return cmp < 0
		}
		return left.SubscriptionGroupId < right.SubscriptionGroupId
	})
	return rows
}

func firestoreStaffReachableJobs(snapshot firestoreLandingSnapshot, workspaceID, staffID string) map[string]struct{} {
	reachable := make(map[string]struct{})
	if strings.TrimSpace(staffID) == "" {
		return reachable
	}
	jobs := landingDocumentsByID(snapshot.jobs)
	templates := landingDocumentsByID(snapshot.jobTemplates)
	phaseJobs := make(map[string]string)
	for _, phase := range snapshot.jobPhases {
		phaseJobs[landingString(phase, "id")] = landingString(phase, "job_id")
	}
	taskJobs := make(map[string]string)
	for _, task := range snapshot.jobTasks {
		jobID := phaseJobs[landingString(task, "job_phase_id")]
		if jobID == "" || jobs[jobID] == nil {
			continue
		}
		taskJobs[landingString(task, "id")] = jobID
		if landingString(task, "assigned_to") == staffID {
			reachable[jobID] = struct{}{}
		}
	}
	for _, outcome := range snapshot.taskOutcomes {
		if landingString(outcome, "recorded_by") != staffID && landingString(outcome, "reviewed_by") != staffID {
			continue
		}
		if jobID := taskJobs[landingString(outcome, "job_task_id")]; jobID != "" {
			reachable[jobID] = struct{}{}
		}
	}

	productPlans := landingDocumentsByID(snapshot.productPlans)
	for _, seat := range snapshot.subscriptionSeats {
		if !landingBool(seat, "active") || landingString(seat, "status") != "active" || landingString(seat, "staff_id") != staffID {
			continue
		}
		productID := landingString(productPlans[landingString(seat, "product_plan_id")], "product_id")
		if productID == "" {
			continue
		}
		for _, job := range snapshot.jobs {
			if landingString(job, "origin_type") != landingOriginSubscription || landingString(job, "origin_id") != landingString(seat, "subscription_id") {
				continue
			}
			template := templates[landingString(job, "job_template_id")]
			if template != nil && landingString(template, "output_product_id") == productID {
				reachable[landingString(job, "id")] = struct{}{}
			}
		}
	}

	productPlanStaffs := landingDocumentsByID(snapshot.productPlanStaffs)
	activeMembersByGroup := make(map[string][]firestoreLandingDocument)
	for _, member := range snapshot.members {
		if landingBool(member, "active") {
			activeMembersByGroup[landingString(member, "subscription_group_id")] = append(activeMembersByGroup[landingString(member, "subscription_group_id")], member)
		}
	}
	for _, edge := range snapshot.classEdges {
		if !landingBool(edge, "active") {
			continue
		}
		resolvedStaffID := landingString(edge, "staff_id")
		if linkedID := landingString(edge, "product_plan_staff_id"); linkedID != "" {
			linked := productPlanStaffs[linkedID]
			if linked == nil || !landingBool(linked, "active") {
				continue
			}
			resolvedStaffID = landingString(linked, "staff_id")
		}
		if resolvedStaffID != staffID {
			continue
		}
		productID := landingString(productPlans[landingString(edge, "product_plan_id")], "product_id")
		if productID == "" {
			continue
		}
		for _, member := range activeMembersByGroup[landingString(edge, "subscription_group_id")] {
			for _, job := range snapshot.jobs {
				if landingString(job, "origin_id") == landingString(member, "subscription_id") && landingString(job, "output_product_id") == productID {
					reachable[landingString(job, "id")] = struct{}{}
				}
			}
		}
	}
	return reachable
}

func landingDocumentsByID(docs []firestoreLandingDocument) map[string]firestoreLandingDocument {
	result := make(map[string]firestoreLandingDocument, len(docs))
	for _, doc := range docs {
		if id := landingString(doc, "id"); id != "" {
			result[id] = doc
		}
	}
	return result
}

func landingString(doc firestoreLandingDocument, key string) string {
	if doc == nil {
		return ""
	}
	value, _ := doc[key].(string)
	return strings.TrimSpace(value)
}

func landingBool(doc firestoreLandingDocument, key string) bool {
	if doc == nil {
		return false
	}
	value, _ := doc[key].(bool)
	return value
}

func landingInt32Pointer(doc firestoreLandingDocument, key string) *int32 {
	if doc == nil {
		return nil
	}
	var value int32
	switch number := doc[key].(type) {
	case int:
		value = int32(number)
	case int32:
		value = number
	case int64:
		value = int32(number)
	case float32:
		value = int32(number)
	case float64:
		value = int32(number)
	default:
		return nil
	}
	return &value
}

func landingOptionalInt32Compare(left, right *int32) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}
	if *left < *right {
		return -1
	}
	if *left > *right {
		return 1
	}
	return 0
}

func landingSortName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

var _ ports.SubscriptionGroupOutcomeLandingQueryService = (*firestoreSubscriptionGroupOutcomeLandingQuery)(nil)
