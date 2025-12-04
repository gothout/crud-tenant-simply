package auditoria_log

import "context"

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Log(ctx context.Context, entry AuditLog) error {
	return s.repo.Save(ctx, entry)
}
