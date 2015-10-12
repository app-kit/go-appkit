package gomail

import (
	"fmt"
	"os"

	"github.com/theduke/go-apperror"
	"gopkg.in/gomail.v2"

	kit "github.com/theduke/go-appkit"
	"github.com/theduke/go-appkit/email"
	"github.com/theduke/go-appkit/utils"
)

type Service struct {
	debug    bool
	registry kit.Registry

	defaultSender kit.EmailRecipient

	dialer *gomail.Dialer
}

// Ensure Service implements email.Service.
var _ kit.EmailService = (*Service)(nil)

func New(registry kit.Registry, host string, port int, user, password, defaultSenderEmail, defaultSenderName string) *Service {
	s := &Service{
		registry: registry,
		defaultSender: email.Recipient{
			Email: defaultSenderEmail,
			Name:  defaultSenderName,
		},

		dialer: gomail.NewPlainDialer(host, port, user, password),
	}

	return s
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

func setAddressHeader(msg *gomail.Message, name string, recipients []kit.EmailRecipient) {
	if len(recipients) < 1 {
		return
	}

	header := make([]string, 0)
	for _, recp := range recipients {
		header = append(header, msg.FormatAddress(recp.GetEmail(), recp.GetName()))
	}

	msg.SetHeader(name, header...)
}

func (s Service) buildMessage(e kit.Email) (*gomail.Message, []string, apperror.Error) {
	msg := gomail.NewMessage()

	msg.SetHeader("Subject", e.GetSubject())

	from := e.GetFrom()
	if from.GetEmail() != "" {
		msg.SetAddressHeader("From", from.GetEmail(), from.GetName())
	} else {
		msg.SetAddressHeader("From", s.defaultSender.GetEmail(), s.defaultSender.GetName())
	}

	setAddressHeader(msg, "To", e.GetTo())
	setAddressHeader(msg, "Cc", e.GetCc())
	setAddressHeader(msg, "Bcc", e.GetBcc())

	// Body.
	for _, part := range e.GetBodyParts() {
		msg.AddAlternative(part.GetMimeType(), string(part.GetContent()))
	}

	var files []string

	// Attachments.
	for _, part := range e.GetAttachments() {
		path, err := utils.WriteTmpFile(part.GetContent(), "")
		if err != nil {
			return nil, files, err
		}
		msg.Attach(path)
		files = append(files, path)
	}

	for _, part := range e.GetEmbeddedAttachments() {
		path, err := utils.WriteTmpFile(part.GetContent(), "")
		if err != nil {
			return nil, files, err
		}
		defer os.Remove(path)
		msg.Embed(path)
		files = append(files, path)
	}

	return msg, files, nil
}

func (s Service) Send(mail kit.Email) apperror.Error {
	err, errs := s.SendMultiple(mail)
	if err != nil {
		return err
	}
	return errs[0]
}

func (s Service) SendMultiple(emails ...kit.Email) (apperror.Error, []apperror.Error) {
	sender, err := s.dialer.Dial()
	if err != nil {
		msg := fmt.Sprintf("Could not connect to smtp server at %v:%v", s.dialer.Host, s.dialer.Port)
		return apperror.Wrap(err, "smtp_dial_failed", msg), nil
	}
	defer sender.Close()

	errs := make([]apperror.Error, 0)

	for _, email := range emails {
		msg, files, err := s.buildMessage(email)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		defer func(files []string) {
			for _, path := range files {
				os.Remove(path)
			}
		}(files)

		if err := gomail.Send(sender, msg); err != nil {
			errs = append(errs, apperror.Wrap(err, "smtp_send_error"))
			continue
		}

		errs = append(errs, nil)
	}

	return nil, errs
}
