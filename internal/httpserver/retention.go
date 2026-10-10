package httpserver

import (
	"context"
	"time"
)

const (
	linkRetention = 30 * 24 * time.Hour
	retentionTick = time.Hour
)

// RunRetention purges delivered rows older than 30 days, then once an hour until ctx is done.
func (s *Server) RunRetention(ctx context.Context) {
	if err := s.purgeDelivered(ctx); err != nil && ctx.Err() == nil {
		s.log.Error("retention", "err", err)
	}
	t := time.NewTicker(retentionTick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.purgeDelivered(ctx); err != nil && ctx.Err() == nil {
				s.log.Error("retention", "err", err)
			}
		}
	}
}

func (s *Server) purgeDelivered(ctx context.Context) error {
	cutoff := time.Now().UTC().Add(-linkRetention)
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM link_deliveries WHERE acked_at IS NOT NULL AND acked_at < $1`,
		cutoff,
	); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`DELETE FROM links l
		 WHERE l.created_at < $1
		   AND NOT EXISTS (
		     SELECT 1 FROM link_deliveries d
		     WHERE d.link_id = l.id AND d.acked_at IS NULL
		   )`,
		cutoff,
	)
	return err
}
