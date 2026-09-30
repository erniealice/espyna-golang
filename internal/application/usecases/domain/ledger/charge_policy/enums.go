package charge_policy

import enumspb "github.com/erniealice/esqyma/pkg/schema/v1/domain/ledger/enums"

const (
	enumspbVersionDraft      = enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_DRAFT
	enumspbVersionApproved   = enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_APPROVED
	enumspbVersionSuperseded = enumspb.ChargePolicyVersionStatus_CHARGE_POLICY_VERSION_STATUS_SUPERSEDED
)
