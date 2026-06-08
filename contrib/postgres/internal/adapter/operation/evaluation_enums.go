//go:build postgresql

package operation

import (
	enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/enums"
	evaluationpb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation"
	evaluationcyclepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_cycle"
	evaluationtemplatepb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/evaluation_template"
)

// Performance-evaluation enum↔lowercase-token translation.
//
// The eval migrations pin DB CHECK constraints to SHORT lowercase tokens
// (e.g. status IN ('draft','submitted','archived','signed_off');
// evaluation_type IN ('performance_review','csat',…); subject_type IN
// ('associate','client')). protojson serializes a proto enum to its SCREAMING
// name (EVALUATION_STATUS_DRAFT). Every eval adapter MUST translate proto-enum
// ↔ DB-token in BOTH directions before write / after read — exactly like
// subscription_seat's subscriptionSeatStatusEnumToToken /
// subscriptionSeatStatusFromString.
//
// The *EnumToToken functions overwrite (or delete on UNSPECIFIED) the
// protojson-serialized value in the write map. The *FromString functions map a
// DB token back to the proto enum (fail-safe UNSPECIFIED on unknown/empty).

// ---- EvaluationStatus ----

func evaluationStatusTokenFromEnum(s evaluationpb.EvaluationStatus) string {
	switch s {
	case evaluationpb.EvaluationStatus_EVALUATION_STATUS_DRAFT:
		return "draft"
	case evaluationpb.EvaluationStatus_EVALUATION_STATUS_SUBMITTED:
		return "submitted"
	case evaluationpb.EvaluationStatus_EVALUATION_STATUS_ARCHIVED:
		return "archived"
	case evaluationpb.EvaluationStatus_EVALUATION_STATUS_SIGNED_OFF:
		return "signed_off"
	default:
		return ""
	}
}

func evaluationStatusFromString(s string) evaluationpb.EvaluationStatus {
	switch s {
	case "draft":
		return evaluationpb.EvaluationStatus_EVALUATION_STATUS_DRAFT
	case "submitted":
		return evaluationpb.EvaluationStatus_EVALUATION_STATUS_SUBMITTED
	case "archived":
		return evaluationpb.EvaluationStatus_EVALUATION_STATUS_ARCHIVED
	case "signed_off":
		return evaluationpb.EvaluationStatus_EVALUATION_STATUS_SIGNED_OFF
	default:
		return evaluationpb.EvaluationStatus_EVALUATION_STATUS_UNSPECIFIED
	}
}

// ---- EvaluationType ----

func evaluationTypeTokenFromEnum(t evaluationpb.EvaluationType) string {
	switch t {
	case evaluationpb.EvaluationType_EVALUATION_TYPE_PERFORMANCE_REVIEW:
		return "performance_review"
	case evaluationpb.EvaluationType_EVALUATION_TYPE_CSAT:
		return "csat"
	case evaluationpb.EvaluationType_EVALUATION_TYPE_COURSE_EVAL:
		return "course_eval"
	case evaluationpb.EvaluationType_EVALUATION_TYPE_VENDOR_SCORECARD:
		return "vendor_scorecard"
	default:
		return ""
	}
}

func evaluationTypeFromString(s string) evaluationpb.EvaluationType {
	switch s {
	case "performance_review":
		return evaluationpb.EvaluationType_EVALUATION_TYPE_PERFORMANCE_REVIEW
	case "csat":
		return evaluationpb.EvaluationType_EVALUATION_TYPE_CSAT
	case "course_eval":
		return evaluationpb.EvaluationType_EVALUATION_TYPE_COURSE_EVAL
	case "vendor_scorecard":
		return evaluationpb.EvaluationType_EVALUATION_TYPE_VENDOR_SCORECARD
	default:
		return evaluationpb.EvaluationType_EVALUATION_TYPE_UNSPECIFIED
	}
}

// ---- RelationshipType ----

func relationshipTypeTokenFromEnum(t evaluationpb.RelationshipType) string {
	switch t {
	case evaluationpb.RelationshipType_RELATIONSHIP_TYPE_CLIENT_TO_ASSOCIATE:
		return "client_to_associate"
	case evaluationpb.RelationshipType_RELATIONSHIP_TYPE_STAFF_TO_CLIENT:
		return "staff_to_client"
	case evaluationpb.RelationshipType_RELATIONSHIP_TYPE_SELF:
		return "self"
	case evaluationpb.RelationshipType_RELATIONSHIP_TYPE_PEER:
		return "peer"
	case evaluationpb.RelationshipType_RELATIONSHIP_TYPE_MANAGER:
		return "manager"
	default:
		return ""
	}
}

func relationshipTypeFromString(s string) evaluationpb.RelationshipType {
	switch s {
	case "client_to_associate":
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_CLIENT_TO_ASSOCIATE
	case "staff_to_client":
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_STAFF_TO_CLIENT
	case "self":
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_SELF
	case "peer":
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_PEER
	case "manager":
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_MANAGER
	default:
		return evaluationpb.RelationshipType_RELATIONSHIP_TYPE_UNSPECIFIED
	}
}

// ---- EvaluatorType ----

func evaluatorTypeTokenFromEnum(t evaluationpb.EvaluatorType) string {
	switch t {
	case evaluationpb.EvaluatorType_EVALUATOR_TYPE_CLIENT:
		return "client"
	case evaluationpb.EvaluatorType_EVALUATOR_TYPE_STAFF:
		return "staff"
	default:
		return ""
	}
}

func evaluatorTypeFromString(s string) evaluationpb.EvaluatorType {
	switch s {
	case "client":
		return evaluationpb.EvaluatorType_EVALUATOR_TYPE_CLIENT
	case "staff":
		return evaluationpb.EvaluatorType_EVALUATOR_TYPE_STAFF
	default:
		return evaluationpb.EvaluatorType_EVALUATOR_TYPE_UNSPECIFIED
	}
}

// ---- SubjectType ----

func subjectTypeTokenFromEnum(t evaluationpb.SubjectType) string {
	switch t {
	case evaluationpb.SubjectType_SUBJECT_TYPE_ASSOCIATE:
		return "associate"
	case evaluationpb.SubjectType_SUBJECT_TYPE_CLIENT:
		return "client"
	default:
		return ""
	}
}

func subjectTypeFromString(s string) evaluationpb.SubjectType {
	switch s {
	case "associate":
		return evaluationpb.SubjectType_SUBJECT_TYPE_ASSOCIATE
	case "client":
		return evaluationpb.SubjectType_SUBJECT_TYPE_CLIENT
	default:
		return evaluationpb.SubjectType_SUBJECT_TYPE_UNSPECIFIED
	}
}

// ---- VisibilityType ----

func visibilityTypeTokenFromEnum(t evaluationpb.VisibilityType) string {
	switch t {
	case evaluationpb.VisibilityType_VISIBILITY_TYPE_INTERNAL_ONLY:
		return "internal_only"
	case evaluationpb.VisibilityType_VISIBILITY_TYPE_VISIBLE_TO_SUBJECT:
		return "visible_to_subject"
	case evaluationpb.VisibilityType_VISIBILITY_TYPE_VISIBLE_TO_SUBJECT_ANON:
		return "visible_to_subject_anon"
	default:
		return ""
	}
}

func visibilityTypeFromString(s string) evaluationpb.VisibilityType {
	switch s {
	case "internal_only":
		return evaluationpb.VisibilityType_VISIBILITY_TYPE_INTERNAL_ONLY
	case "visible_to_subject":
		return evaluationpb.VisibilityType_VISIBILITY_TYPE_VISIBLE_TO_SUBJECT
	case "visible_to_subject_anon":
		return evaluationpb.VisibilityType_VISIBILITY_TYPE_VISIBLE_TO_SUBJECT_ANON
	default:
		return evaluationpb.VisibilityType_VISIBILITY_TYPE_UNSPECIFIED
	}
}

// ---- EvaluationTemplateStatus ----

func evaluationTemplateStatusTokenFromEnum(s evaluationtemplatepb.EvaluationTemplateStatus) string {
	switch s {
	case evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DRAFT:
		return "draft"
	case evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_ACTIVE:
		return "active"
	case evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DEPRECATED:
		return "deprecated"
	default:
		return ""
	}
}

func evaluationTemplateStatusFromString(s string) evaluationtemplatepb.EvaluationTemplateStatus {
	switch s {
	case "draft":
		return evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DRAFT
	case "active":
		return evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_ACTIVE
	case "deprecated":
		return evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_DEPRECATED
	default:
		return evaluationtemplatepb.EvaluationTemplateStatus_EVALUATION_TEMPLATE_STATUS_UNSPECIFIED
	}
}

// ---- EvaluationCycleStatus ----

func evaluationCycleStatusTokenFromEnum(s evaluationcyclepb.EvaluationCycleStatus) string {
	switch s {
	case evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_OPEN:
		return "open"
	case evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_SIGN_OFF:
		return "sign_off"
	case evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_CLOSED:
		return "closed"
	default:
		return ""
	}
}

func evaluationCycleStatusFromString(s string) evaluationcyclepb.EvaluationCycleStatus {
	switch s {
	case "open":
		return evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_OPEN
	case "sign_off":
		return evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_SIGN_OFF
	case "closed":
		return evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_CLOSED
	default:
		return evaluationcyclepb.EvaluationCycleStatus_EVALUATION_CYCLE_STATUS_UNSPECIFIED
	}
}

// ---- CriteriaType (operation/enums) — snapshotted on evaluation_response ----

func criteriaTypeTokenFromEnum(t enumspb.CriteriaType) string {
	switch t {
	case enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE:
		return "numeric_range"
	case enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE:
		return "numeric_score"
	case enumspb.CriteriaType_CRITERIA_TYPE_PASS_FAIL:
		return "pass_fail"
	case enumspb.CriteriaType_CRITERIA_TYPE_CATEGORICAL:
		return "categorical"
	case enumspb.CriteriaType_CRITERIA_TYPE_TEXT:
		return "text"
	case enumspb.CriteriaType_CRITERIA_TYPE_MULTI_CHECK:
		return "multi_check"
	default:
		return ""
	}
}

func criteriaTypeFromString(s string) enumspb.CriteriaType {
	switch s {
	case "numeric_range":
		return enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_RANGE
	case "numeric_score":
		return enumspb.CriteriaType_CRITERIA_TYPE_NUMERIC_SCORE
	case "pass_fail":
		return enumspb.CriteriaType_CRITERIA_TYPE_PASS_FAIL
	case "categorical":
		return enumspb.CriteriaType_CRITERIA_TYPE_CATEGORICAL
	case "text":
		return enumspb.CriteriaType_CRITERIA_TYPE_TEXT
	case "multi_check":
		return enumspb.CriteriaType_CRITERIA_TYPE_MULTI_CHECK
	default:
		return enumspb.CriteriaType_CRITERIA_TYPE_UNSPECIFIED
	}
}

// setOrDeleteToken writes token at key, or deletes the key when token=="" so a
// partial UPDATE never writes an invalid (UNSPECIFIED) enum value that the DB
// CHECK would reject.
func setOrDeleteToken(data map[string]any, key, token string) {
	if token == "" {
		delete(data, key)
		return
	}
	data[key] = token
}
