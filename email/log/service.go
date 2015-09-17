// A email service implementation that just logs the mails.
package log

import(
	log "github.com/Sirupsen/logrus"

	. "github.com/theduke/go-appkit/error"
	. "github.com/theduke/go-appkit"
)

type Service struct {
	logger *log.Logger
	defaultSender EmailRecipient
}

func New(logger *log.Logger, defaultSender EmailRecipient) *Service {
	return &Service{
		logger: logger,
		defaultSender: defaultSender,
	}
}

func (s *Service) SetDefaultFrom(r EmailRecipient) {
	s.defaultSender = r
}


func (s *Service) SetLogger(l *log.Logger) {
	s.logger = l
}

func (s Service) Send(e Email) Error {
	err, errs := s.SendMultiple(e)
	if err != nil {
		return err
	}
	return errs[0]
}

func (s Service) SendMultiple(emails ...Email) (Error, []Error) {
	for _, e := range emails {
		from := e.GetFrom().GetEmail()
		if from == "" {
			s.defaultSender.GetEmail()
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

		s.logger.WithFields(log.Fields{
			"action": "send_email",
			"from": from,
			"to": recipients,
			"subject": e.GetSubject(),
		}).Debugf("Sending mail from %v to %v - subject %v", from, recipients, e.GetSubject())
	}

	return nil, make([]Error, len(emails))
}
