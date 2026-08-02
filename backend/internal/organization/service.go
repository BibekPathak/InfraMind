package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/inframind/backend/pkg/uuidv7"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, req CreateOrganizationRequest) (*Organization, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("organization name is required")
	}
	if req.Slug == "" {
		req.Slug = slugify(req.Name)
	}

	o := &Organization{
		ID:       uuidv7.New(),
		Name:     req.Name,
		Slug:     req.Slug,
		Settings: req.Settings,
	}

	if err := s.repo.Create(ctx, o); err != nil {
		return nil, fmt.Errorf("create organization: %w", err)
	}
	return o, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Organization, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*Organization, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) List(ctx context.Context) ([]Organization, error) {
	return s.repo.List(ctx)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateOrganizationRequest) (*Organization, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Slug = req.Slug
	if req.Settings != nil {
		existing.Settings = req.Settings
	}

	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}
	return existing, nil
}

func slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, s)
	return s
}
