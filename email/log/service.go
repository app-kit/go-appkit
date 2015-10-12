// A email service implementation that just logs the mails.
package log

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/theduke/go-apperror"

	kit "github.com/theduke/go-appkit"
)

type Service struct {
	debug         bool
	registry      kit.Registry
	defaultSender kit.EmailRecipient
}

// Ensure Service implements email.Service.
var _ kit.EmailService = (*Service)(nil)

func New(registry kit.Registry, defaultSender kit.EmailRecipient) *Service {
	return &Service{
		registry:      registry,
		defaultSender: defaultSender,
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

func (s *Service) SetDefaultFrom(r kit.EmailRecipient) {
	s.defaultSender = r
}

func (s Service) Send(e kit.Email) apperror.Error {
	err, errs := s.SendMultiple(e)
	if err != nil {
		return err
	}
	return errs[0]
}

func (s Service) SendMultiple(emails ...kit.Email) (apperror.Error, []apperror.Error) {
	for _, e := range emails {
		from := e.GetFrom().GetEmail()
		if from == "" {
			from = s.defaultSender.GetEmail()
		}

		var recipients []string

		for _, recp := range e.GetTo() {
			recipients = append(recipients, recp.GetEmail())
		}
		for _, recp := range e.GetCc() {
			recipients = append(recipients, recp.GetEmail())
		}
		for _, recp := range e.GetBcc() {
			recipients = append(recipients, recp.GetEmail())
		}

		msg := fmt.Sprintf("Sending mail from %v to %v - subject %v", from, recipients, e.GetSubject())
		for _, part := range e.GetBodyParts() {
			msg += "\n####################\n"
			msg += string(part.GetContent())
		}
		msg += "\n####################\n\n"

		s.registry.Logger().WithFields(log.Fields{
			"action":  "send_email",
			"from":    from,
			"to":      recipients,
			"subject": e.GetSubject(),
		}).Debug(msg)
	}

	return nil, make([]apperror.Error, len(emails))
}
