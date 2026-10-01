package bssci

import "context"

// evaluateRoaming reports whether the serving station's tenant serves a
// foreign endpoint; without a roaming service nothing roams.
func (s *Server) evaluateRoaming(ctx context.Context, epEui []byte, servingTenantID int64) (bool, error) {
	if s.roamingSvc == nil {
		return false, nil
	}
	isRoaming, _, err := s.roamingSvc.DetectAndValidateRoaming(ctx, epEui, servingTenantID)
	return isRoaming, err
}
