package acess_log

import (
	"context"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Log(ctx context.Context, entry AccessLog) error {
	return s.repo.Save(ctx, entry)
}
