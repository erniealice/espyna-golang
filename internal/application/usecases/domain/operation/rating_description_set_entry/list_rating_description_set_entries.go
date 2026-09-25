package rating_description_set_entry

import (
	"context"
	"errors"

	"github.com/erniealice/espyna-golang/internal/application/ports"
	"github.com/erniealice/espyna-golang/internal/application/shared/actiongate"
	contextutil "github.com/erniealice/espyna-golang/internal/application/shared/context"
	"github.com/erniealice/espyna-golang/registry/entityid"
	pb "github.com/erniealice/esqyma/pkg/schema/v1/domain/operation/rating_description_set_entry"
)

type ListRatingDescriptionSetEntriesRepositories struct {
	RatingDescriptionSetEntry pb.RatingDescriptionSetEntryDomainServiceServer
}

type ListRatingDescriptionSetEntriesServices struct {
	Authorizer       ports.Authorizer
	Transactor       ports.Transactor
	Translator       ports.Translator
	IDGenerator      ports.IDGenerator
	ActionGatekeeper *actiongate.ActionGatekeeper
}

type ListRatingDescriptionSetEntriesUseCase struct {
	repositories ListRatingDescriptionSetEntriesRepositories
	services     ListRatingDescriptionSetEntriesServices
}

func NewListRatingDescriptionSetEntriesUseCase(r ListRatingDescriptionSetEntriesRepositories, s ListRatingDescriptionSetEntriesServices) *ListRatingDescriptionSetEntriesUseCase {
	return &ListRatingDescriptionSetEntriesUseCase{repositories: r, services: s}
}

// Authorization (W3 follow-up finding, 2026-09-25): checked against the
// PARENT set (rating_description_set:read) — entries carry no permission
// codes of their own. See create_rating_description_set_entry.go's doc
// comment.
func (uc *ListRatingDescriptionSetEntriesUseCase) Execute(ctx context.Context, req *pb.ListRatingDescriptionSetEntriesRequest) (*pb.ListRatingDescriptionSetEntriesResponse, error) {
	if err := uc.services.ActionGatekeeper.Check(ctx, &actiongate.CheckActionRequest{Entity: entityid.RatingDescriptionSet, Action: entityid.ActionRead}); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, errors.New(contextutil.GetTranslatedMessageWithContext(ctx, uc.services.Translator, "rating_description_set_entry.validation.request_required", "Request is required [DEFAULT]"))
	}
	return uc.repositories.RatingDescriptionSetEntry.ListRatingDescriptionSetEntries(ctx, req)
}
