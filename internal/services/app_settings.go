package services

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"web-manager/internal/db"
	"web-manager/internal/domain"
	"web-manager/internal/repository"
)

// AppSettingsProvider holds the latest app_settings snapshot for CORS and other middleware.
type AppSettingsProvider struct {
	mu      sync.RWMutex
	appName string
	origins []string
	smtp    *domain.SMTPSettings
	extra   map[string]any
}

func NewAppSettingsProvider() *AppSettingsProvider {
	return &AppSettingsProvider{}
}

func (p *AppSettingsProvider) Apply(doc *domain.AppSettings) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.appName = doc.AppName
	p.origins = append([]string(nil), doc.Origins...)
	if doc.SMTP != nil {
		cp := *doc.SMTP
		p.smtp = &cp
	} else {
		p.smtp = nil
	}
	if doc.Extra != nil {
		p.extra = make(map[string]any, len(doc.Extra))
		for k, v := range doc.Extra {
			p.extra[k] = v
		}
	} else {
		p.extra = nil
	}
}

func (p *AppSettingsProvider) AppName() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.appName
}

func (p *AppSettingsProvider) Origins() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]string(nil), p.origins...)
}

func (p *AppSettingsProvider) AllowOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.origins) == 0 {
		return true
	}
	for _, o := range p.origins {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}

func (p *AppSettingsProvider) SMTP() *domain.SMTPSettings {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.smtp == nil {
		return nil
	}
	cp := *p.smtp
	return &cp
}

// AppSettingsService loads app_settings from Mongo and seeds a default row when the collection is empty.
type AppSettingsService struct {
	repo *repository.AppSettingsRepository
	log  *zap.Logger
}

func NewAppSettingsService(m *db.Mongo, logger *zap.Logger) *AppSettingsService {
	return &AppSettingsService{
		repo: repository.NewAppSettingsRepository(m),
		log:  logger,
	}
}

// LoadOrSeedDefault reads key=default (or latest document), or inserts a dev-friendly default if the collection is empty.
func (s *AppSettingsService) LoadOrSeedDefault(ctx context.Context, prov *AppSettingsProvider) error {
	n, err := s.repo.Count(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		seed := defaultAppSettingsDocument()
		if err := s.repo.Insert(ctx, seed); err != nil {
			return err
		}
		prov.Apply(seed)
		if s.log != nil {
			s.log.Info("seeded app_settings (collection was empty)", zap.Strings("origins", seed.Origins))
		}
		return nil
	}

	doc, err := s.repo.FindDefault(ctx)
	if err != nil {
		return err
	}
	prov.Apply(doc)
	if s.log != nil {
		s.log.Info("loaded app_settings from Mongo",
			zap.String("appName", doc.AppName),
			zap.Strings("origins", doc.Origins),
		)
	}
	return nil
}

func defaultAppSettingsDocument() *domain.AppSettings {
	now := time.Now().UTC()
	return &domain.AppSettings{
		Key:     domain.AppSettingsKeyDefault,
		AppName: "Web Manager",
		Origins: []string{
			"http://localhost:3000",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}
