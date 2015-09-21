package resources

import (
	"fmt"

	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Service struct {
	debug bool
	deps  kit.Dependencies

	defaultBackend db.Backend
	resources      map[string]kit.Resource
}

// Ensure Service implements ResourceService.
var _ kit.ResourceService = (*Service)(nil)

func NewService(debug bool, deps kit.Dependencies) *Service {
	return &Service{
		debug:          debug,
		deps:           deps,
		defaultBackend: deps.DefaultBackend(),
		resources:      make(map[string]kit.Resource),
	}
}

func (s *Service) Debug() bool {
	return s.debug
}

func (s *Service) SetDebug(x bool) {
	s.debug = x
}

func (s *Service) Dependencies() kit.Dependencies {
	return s.deps
}

func (s *Service) SetDependencies(x kit.Dependencies) {
	s.deps = x
}

func (s *Service) SetDefaultBackend(b db.Backend) {
	s.defaultBackend = b
}

func (s *Service) Resource(name string) kit.Resource {
	return s.resources[name]
}

func (s *Service) RegisterResource(res kit.Resource) {
	if res.Backend() == nil {
		if s.defaultBackend == nil {
			s.deps.Logger().Panic("Registering resource without backend, but no default backend set on resources.Service")
		}
		s.defaultBackend.RegisterModel(res.Model())
		res.SetBackend(s.defaultBackend)
	}

	if res.Collection() == "" {
		s.deps.Logger().Panic("Registering resource without a model type")
	}

	s.resources[res.Collection()] = res
}

func (s *Service) Q(modelType string) (db.Query, kit.Error) {
	res := s.resources[modelType]
	if res == nil {
		return nil, kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", modelType),
		}
	}

	return res.Q(), nil
}

func (s *Service) FindOne(modelType string, id string) (db.Model, kit.Error) {
	res := s.resources[modelType]
	if res == nil {
		return nil, kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", modelType),
		}
	}

	return res.FindOne(id)
}

func (s *Service) Create(m db.Model, user kit.User) kit.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Create(m, user)
}

func (s *Service) Update(m db.Model, user kit.User) kit.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Update(m, user)
}

func (s *Service) Delete(m db.Model, user kit.User) kit.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return kit.AppError{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Delete(m, user)
}
