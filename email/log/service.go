// A email service implementation that just logs the mails.
package log

import (
	log "github.com/Sirupsen/logrus"

	kit "github.com/theduke/go-appkit"
)

type Service struct {
	debug         bool
	deps          kit.Dependencies
	defaultSender kit.EmailRecipient
}

// Ensure Service implements email.Service.
var _ kit.EmailService = (*Service)(nil)

func New(deps kit.Dependencies, defaultSender kit.EmailRecipient) *Service {
	return &Service{
		deps:          deps,
		defaultSender: defaultSender,
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

func (s *Service) SetDefaultFrom(r kit.EmailRecipient) {
	s.defaultSender = r
}

func (s Service) Send(e kit.Email) kit.Error {
	err, errs := s.SendMultiple(e)
	if err != nil {
		return err
	}
	return errs[0]
}

func (s Service) SendMultiple(emails ...kit.Email) (kit.Error, []kit.Error) {
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

		s.deps.Logger().WithFields(log.Fields{
			"action":  "send_email",
			"from":    from,
			"to":      recipients,
			"subject": e.GetSubject(),
		}).Debugf("Sending mail from %v to %v - subject %v", from, recipients, e.GetSubject())
	}

	return nil, make([]kit.Error, len(emails))
}
