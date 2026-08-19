package targeting

import "context"

func (s *Service) GetFlagState(ctx context.Context, organisationID, projectID, featureFlagID string) (FlagState, error) {
	return s.repository.GetFlagState(ctx, organisationID, projectID, featureFlagID)
}
