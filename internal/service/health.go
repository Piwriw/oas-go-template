package service

import (
	"context"
	"fmt"
)

// Ready reports whether configured dependencies are reachable. A nil database
// means the dependency is intentionally disabled and never fails readiness.
// Failures wrap one of the sentinel errors in this package around the private
// cause — the cause must never leak into API responses.
func (s *Service) Ready(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDBHandle, err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("%w: %w", ErrDBPing, err)
	}
	return nil
}
