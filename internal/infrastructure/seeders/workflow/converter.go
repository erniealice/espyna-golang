package workflow

import (
	"encoding/json"

	activitypb "leapfor.xyz/esqyma/golang/v1/domain/workflow/activity_template"
	stagepb "leapfor.xyz/esqyma/golang/v1/domain/workflow/stage_template"
	workflowpb "leapfor.xyz/esqyma/golang/v1/domain/workflow/workflow_template"
	"leapfor.xyz/vya"
)

// ConvertWorkflowTemplate converts a vya WorkflowTemplate to protobuf
func ConvertWorkflowTemplate(tmpl *vya.WorkflowTemplate, id, workspaceID string) *workflowpb.WorkflowTemplate {
	// Serialize input schema to JSON
	var inputSchemaJSON *string
	if tmpl.InputSchema != nil {
		data, _ := json.Marshal(tmpl.InputSchema)
		str := string(data)
		inputSchemaJSON = &str
	}

	version := tmpl.Version
	isSystem := tmpl.IsSystem
	systemID := tmpl.ID

	return &workflowpb.WorkflowTemplate{
		Id:              id,
		Name:            tmpl.Name,
		Description:     &tmpl.Description,
		WorkspaceId:     &workspaceID,
		BusinessType:    tmpl.BusinessType,
		Status:          "active",
		Version:         &version,
		InputSchemaJson: inputSchemaJSON,
		Active:          true,
		// System template tracking fields
		SystemId: &systemID,
		IsSystem: &isSystem,
	}
}

// ConvertStageTemplate converts a vya StageTemplate to protobuf
func ConvertStageTemplate(stage *vya.StageTemplate, id, workflowTemplateID string) *stagepb.StageTemplate {
	return &stagepb.StageTemplate{
		Id:                  id,
		WorkflowTemplateId:  workflowTemplateID,
		Name:                stage.Name,
		Description:         &stage.Description,
		OrderIndex:          &stage.OrderIndex,
		StageType:           stage.StageType,
		IsRequired:          &stage.IsRequired,
		ConditionExpression: &stage.ConditionExpression,
		Status:              "active",
		Active:              true,
	}
}

// ConvertActivityTemplate converts a vya ActivityTemplate to protobuf
func ConvertActivityTemplate(activity *vya.ActivityTemplate, id, stageTemplateID string) *activitypb.ActivityTemplate {
	// Serialize schemas to JSON
	var inputSchemaJSON, outputSchemaJSON, configJSON *string

	if activity.InputSchema != nil {
		data, _ := json.Marshal(activity.InputSchema)
		str := string(data)
		inputSchemaJSON = &str
	}

	if activity.OutputSchema != nil {
		data, _ := json.Marshal(activity.OutputSchema)
		str := string(data)
		outputSchemaJSON = &str
	}

	if activity.Config != nil {
		data, _ := json.Marshal(activity.Config)
		str := string(data)
		configJSON = &str
	}

	return &activitypb.ActivityTemplate{
		Id:                       id,
		StageTemplateId:          stageTemplateID,
		Name:                     activity.Name,
		Description:              &activity.Description,
		OrderIndex:               &activity.OrderIndex,
		ActivityType:             activity.ActivityType,
		IsRequired:               &activity.IsRequired,
		UseCaseCode:              &activity.UseCaseCode,
		RollbackUseCaseCode:      &activity.RollbackUseCaseCode,
		AssigneeType:             &activity.AssigneeType,
		DefaultAssigneeId:        &activity.DefaultAssigneeID,
		EstimatedDurationMinutes: &activity.EstimatedDurationMinutes,
		ConditionExpression:      &activity.ConditionExpression,
		InputSchemaJson:          inputSchemaJSON,
		OutputSchemaJson:         outputSchemaJSON,
		ConfigurationJson:        configJSON,
		Status:                   "active",
		Active:                   true,
	}
}
