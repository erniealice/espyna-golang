package charge_policy

// UseCases aggregates the charge policy use cases. Field names equal the esqyma RPC names
// (request name minus "Request"); each Execute takes and returns the matching proto messages.
type UseCases struct {
	CreateChargePolicy                     *CreateChargePolicyUseCase
	ReadChargePolicy                       *ReadChargePolicyUseCase
	UpdateChargePolicy                     *UpdateChargePolicyUseCase
	ListChargePolicies                     *ListChargePoliciesUseCase
	GetChargePolicyListPageData            *GetChargePolicyListPageDataUseCase
	ListPickerChargePolicies               *ListPickerChargePoliciesUseCase
	RetireChargePolicy                     *RetireChargePolicyUseCase
	GetChargePolicyInUseIds                *GetChargePolicyInUseIdsUseCase
	DeleteChargePolicy                     *DeleteChargePolicyUseCase
	ResolveChargePolicy                    *ResolveChargePolicyUseCase
	CreateDraftChargePolicyVersion         *CreateDraftChargePolicyVersionUseCase
	ReadChargePolicyVersion                *ReadChargePolicyVersionUseCase
	ListChargePolicyVersions               *ListChargePolicyVersionsUseCase
	UpdateChargePolicyVersion              *UpdateChargePolicyVersionUseCase
	DeleteChargePolicyVersion              *DeleteChargePolicyVersionUseCase
	ValidateChargePolicyVersionForApproval *ValidateChargePolicyVersionForApprovalUseCase
	ApproveChargePolicyVersion             *ApproveChargePolicyVersionUseCase
	CreateChargePolicyComponent            *CreateChargePolicyComponentUseCase
	UpdateChargePolicyComponent            *UpdateChargePolicyComponentUseCase
	DeleteChargePolicyComponent            *DeleteChargePolicyComponentUseCase
	ListChargePolicyComponents             *ListChargePolicyComponentsUseCase
	CreateChargePolicyPosting              *CreateChargePolicyPostingUseCase
	UpdateChargePolicyPosting              *UpdateChargePolicyPostingUseCase
	DeleteChargePolicyPosting              *DeleteChargePolicyPostingUseCase
	ListChargePolicyPostings               *ListChargePolicyPostingsUseCase
}

// NewUseCases wires the charge policy use cases from the shared repository and service groups.
func NewUseCases(r Repositories, s Services) *UseCases {
	return &UseCases{
		CreateChargePolicy:                     NewCreateChargePolicyUseCase(CreateChargePolicyRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, CreateChargePolicyServices(s)),
		ReadChargePolicy:                       NewReadChargePolicyUseCase(ReadChargePolicyRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion}, ReadChargePolicyServices(s)),
		UpdateChargePolicy:                     NewUpdateChargePolicyUseCase(UpdateChargePolicyRepositories{ChargePolicy: r.ChargePolicy}, UpdateChargePolicyServices(s)),
		ListChargePolicies:                     NewListChargePoliciesUseCase(ListChargePoliciesRepositories{ChargePolicy: r.ChargePolicy}, ListChargePoliciesServices(s)),
		GetChargePolicyListPageData:            NewGetChargePolicyListPageDataUseCase(GetChargePolicyListPageDataRepositories{ChargePolicy: r.ChargePolicy}, GetChargePolicyListPageDataServices(s)),
		ListPickerChargePolicies:               NewListPickerChargePoliciesUseCase(ListPickerChargePoliciesRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion}, ListPickerChargePoliciesServices(s)),
		RetireChargePolicy:                     NewRetireChargePolicyUseCase(RetireChargePolicyRepositories{ChargePolicy: r.ChargePolicy}, RetireChargePolicyServices(s)),
		GetChargePolicyInUseIds:                NewGetChargePolicyInUseIdsUseCase(GetChargePolicyInUseIdsRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion}, GetChargePolicyInUseIdsServices(s)),
		DeleteChargePolicy:                     NewDeleteChargePolicyUseCase(DeleteChargePolicyRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, DeleteChargePolicyServices(s)),
		ResolveChargePolicy:                    NewResolveChargePolicyUseCase(ResolveChargePolicyRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion}, ResolveChargePolicyServices(s)),
		CreateDraftChargePolicyVersion:         NewCreateDraftChargePolicyVersionUseCase(CreateDraftChargePolicyVersionRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, CreateDraftChargePolicyVersionServices(s)),
		ReadChargePolicyVersion:                NewReadChargePolicyVersionUseCase(ReadChargePolicyVersionRepositories{ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting}, ReadChargePolicyVersionServices(s)),
		ListChargePolicyVersions:               NewListChargePolicyVersionsUseCase(ListChargePolicyVersionsRepositories{ChargePolicyVersion: r.ChargePolicyVersion}, ListChargePolicyVersionsServices(s)),
		UpdateChargePolicyVersion:              NewUpdateChargePolicyVersionUseCase(UpdateChargePolicyVersionRepositories{ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor, TaxTreatment: r.TaxTreatment}, UpdateChargePolicyVersionServices(s)),
		DeleteChargePolicyVersion:              NewDeleteChargePolicyVersionUseCase(DeleteChargePolicyVersionRepositories{ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, DeleteChargePolicyVersionServices(s)),
		ValidateChargePolicyVersionForApproval: NewValidateChargePolicyVersionForApprovalUseCase(ValidateChargePolicyVersionForApprovalRepositories{ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting, Account: r.Account}, ValidateChargePolicyVersionForApprovalServices(s)),
		ApproveChargePolicyVersion:             NewApproveChargePolicyVersionUseCase(ApproveChargePolicyVersionRepositories{ChargePolicy: r.ChargePolicy, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor, Account: r.Account}, ApproveChargePolicyVersionServices(s)),
		CreateChargePolicyComponent:            NewCreateChargePolicyComponentUseCase(CreateChargePolicyComponentRepositories{ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, CreateChargePolicyComponentServices(s)),
		UpdateChargePolicyComponent:            NewUpdateChargePolicyComponentUseCase(UpdateChargePolicyComponentRepositories{ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, UpdateChargePolicyComponentServices(s)),
		DeleteChargePolicyComponent:            NewDeleteChargePolicyComponentUseCase(DeleteChargePolicyComponentRepositories{ChargePolicyComponent: r.ChargePolicyComponent, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, DeleteChargePolicyComponentServices(s)),
		ListChargePolicyComponents:             NewListChargePolicyComponentsUseCase(ListChargePolicyComponentsRepositories{ChargePolicyComponent: r.ChargePolicyComponent}, ListChargePolicyComponentsServices(s)),
		CreateChargePolicyPosting:              NewCreateChargePolicyPostingUseCase(CreateChargePolicyPostingRepositories{ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, CreateChargePolicyPostingServices(s)),
		UpdateChargePolicyPosting:              NewUpdateChargePolicyPostingUseCase(UpdateChargePolicyPostingRepositories{ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, UpdateChargePolicyPostingServices(s)),
		DeleteChargePolicyPosting:              NewDeleteChargePolicyPostingUseCase(DeleteChargePolicyPostingRepositories{ChargePolicyPosting: r.ChargePolicyPosting, ChargePolicyVersion: r.ChargePolicyVersion, ChargePolicyVersionEditor: r.ChargePolicyVersionEditor}, DeleteChargePolicyPostingServices(s)),
		ListChargePolicyPostings:               NewListChargePolicyPostingsUseCase(ListChargePolicyPostingsRepositories{ChargePolicyPosting: r.ChargePolicyPosting}, ListChargePolicyPostingsServices(s)),
	}
}
