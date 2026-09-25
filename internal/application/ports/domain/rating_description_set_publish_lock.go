package domain

import "context"

// RatingDescriptionSetPublishLocker is the hand-written lock-protocol port the
// publish use case type-asserts on the rating_description_set repository
// (codex-review-impl2 #1; same hand-written-port pattern as
// RatingDescriptionSetLifecycleRepository). It enforces the descriptor lock
// order  score_scale → score_scale_band → rating_description_set  shared with
// the score_scale / score_scale_band meaning guards (see the postgres adapter
// score_scale.go "Descriptor lock protocol"):
//
//  1. LockScoreScaleForPublish — BEFORE the set row lock: reads the set's
//     score_scale_id and takes FOR SHARE on that score_scale row, so a
//     concurrent meaning-changing scale/band edit (FOR UPDATE on the same
//     scale) serializes against the publish.
//  2. (caller) LockRatingDescriptionSetForUpdate.
//  3. VerifyRatingDescriptionSetPublishScale — AFTER the set lock: the set
//     still declares the locked scale, the scale and every entry's band are
//     owned by the caller's workspace (or global, workspace_id IS NULL), and
//     every entry's band belongs to that scale. Otherwise INVALID_CONFIG.
//
// Both methods require an ambient transaction and a trusted workspace on ctx
// and fail closed otherwise.
type RatingDescriptionSetPublishLocker interface {
	LockScoreScaleForPublish(ctx context.Context, setID string) (scoreScaleID string, err error)
	VerifyRatingDescriptionSetPublishScale(ctx context.Context, setID, scoreScaleID string) error
}

// RatingDescriptionSetScaleValidator validates, at set create/update, that a
// score_scale is active and owned by the caller's workspace or global
// (workspace_id IS NULL); otherwise an INVALID_CONFIG error (codex-review-
// impl2 #4). Type-asserted by the create/update use cases; a repository
// without it fails closed.
type RatingDescriptionSetScaleValidator interface {
	ValidateRatingDescriptionSetScoreScale(ctx context.Context, scoreScaleID string) error
}
