package resources

import (
	"fmt"

	"github.com/theduke/go-apperror"
	db "github.com/theduke/go-dukedb"

	kit "github.com/theduke/go-appkit"
)

type Service struct {
	debug    bool
	registry kit.Registry

	defaultBackend db.Backend
	resources      map[string]kit.Resource
}

// Ensure Service implements ResourceService.
var _ kit.ResourceService = (*Service)(nil)

func NewService(debug bool, registry kit.Registry) *Service {
	return &Service{
		debug:          debug,
		registry:       registry,
		defaultBackend: registry.DefaultBackend(),
		resources:      make(map[string]kit.Resource),
	}
}

func (s *Service) Debug() bool {
	return s.debug
}

func (s *Service) SetDebug(x bool) {
	s.debug = x
}

func (s *Service) Registry() kit.Registry {
	return s.registry
}

func (s *Service) SetRegistry(x kit.Registry) {
	s.registry = x
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
			s.registry.Logger().Panic("Registering resource without backend, but no default backend set on resources.Service")
		}
		s.defaultBackend.RegisterModel(res.Model())
		res.SetBackend(s.defaultBackend)
	}

	if res.Collection() == "" {
		s.registry.Logger().Panic("Registering resource without a model type")
	}

	s.resources[res.Collection()] = res
}

func (s *Service) Q(modelType string) (db.Query, apperror.Error) {
	res := s.resources[modelType]
	if res == nil {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", modelType),
		}
	}

	return res.Q(), nil
}

func (s *Service) FindOne(modelType string, id string) (kit.Model, apperror.Error) {
	res := s.resources[modelType]
	if res == nil {
		return nil, &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", modelType),
		}
	}

	return res.FindOne(id)
}

func (s *Service) Create(m kit.Model, user kit.User) apperror.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Create(m, user)
}

func (s *Service) Update(m kit.Model, user kit.User) apperror.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Update(m, user)
}

func (s *Service) Delete(m kit.Model, user kit.User) apperror.Error {
	res := s.resources[m.Collection()]
	if res == nil {
		return &apperror.Err{
			Code:    "unknown_resource",
			Message: fmt.Sprintf("The resource %v was not registered with service", m.Collection()),
		}
	}

	return res.Delete(m, user)
}
