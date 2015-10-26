package email

import (
	"io"
	"io/ioutil"

	"github.com/theduke/go-apperror"

	. "github.com/app-kit/go-appkit"
	"github.com/app-kit/go-appkit/utils"
)

type Recipient struct {
	Email string
	Name  string
}

// Ensure Recipient implements EmailRecipient.
var _ EmailRecipient = (*Recipient)(nil)

func (r Recipient) GetEmail() string {
	return r.Email
}

func (r Recipient) GetName() string {
	return r.Name
}

type Part struct {
	MimeType string
	Content  []byte
	Reader   io.ReadCloser
	FilePath string
}

// Ensure Part implements EmailPart.
var _ EmailRecipient = (*Recipient)(nil)

func (p Part) GetMimeType() string {
	return p.MimeType
}

func (p Part) GetContent() []byte {
	if p.Reader != nil {
		p.Content, _ = ioutil.ReadAll(p.Reader)
		p.Reader.Close()
	}
	return p.Content
}

func (p Part) GetReader() io.ReadCloser {
	return p.Reader
}

func (p Part) GetFilePath() string {
	return p.FilePath
}

func (p *Part) SetFilePath(path string) {
	p.FilePath = path
}

type Mail struct {
	From Recipient
	To   []Recipient
	Cc   []Recipient
	Bcc  []Recipient

	BodyParts           []*Part
	Attachments         []*Part
	EmbeddedAttachments []*Part

	Headers map[string][]string
}

func NewMail() Email {
	return &Mail{
		Headers: make(map[string][]string),
	}
}

// Ensure Email implements Email interface.
var _ Email = (*Mail)(nil)

func (e *Mail) sliceToEmailRecipient(s []Recipient) []EmailRecipient {
	s2 := make([]EmailRecipient, 0)
	for _, r := range s {
		s2 = append(s2, r)
	}
	return s2
}

func (e *Mail) sliceToEmailPart(s []*Part) []EmailPart {
	s2 := make([]EmailPart, 0)
	for _, r := range s {
		s2 = append(s2, r)
	}
	return s2
}

func (e *Mail) SetFrom(email, name string) {
	e.From = Recipient{Email: email, Name: name}
}

func (e *Mail) GetFrom() EmailRecipient {
	return e.From
}

func (e *Mail) AddTo(email, name string) {
	e.To = append(e.To, Recipient{Email: email, Name: name})
}

func (e *Mail) GetTo() []EmailRecipient {
	return e.sliceToEmailRecipient(e.To)
}

func (e *Mail) AddCc(email, name string) {
	e.Cc = append(e.Cc, Recipient{Email: email, Name: name})
}

func (e *Mail) GetCc() []EmailRecipient {
	return e.sliceToEmailRecipient(e.Cc)
}

func (e *Mail) AddBcc(email, name string) {
	e.Bcc = append(e.Bcc, Recipient{Email: email, Name: name})
}

func (e *Mail) GetBcc() []EmailRecipient {
	return e.sliceToEmailRecipient(e.Bcc)
}

func (e *Mail) SetSubject(subject string) {
	e.SetHeader("Subject", subject)
}

func (e *Mail) GetSubject() string {
	s, ok := e.Headers["Subject"]
	if ok && len(s) > 0 {
		return s[0]
	}
	return ""
}

func (e *Mail) SetBody(contentType string, body []byte) {
	e.BodyParts = []*Part{&Part{MimeType: contentType, Content: body}}
}

func (e *Mail) AddBody(contentType string, body []byte) {
	e.BodyParts = append(e.BodyParts, &Part{MimeType: contentType, Content: body})
}

func (e *Mail) GetBodyParts() []EmailPart {
	return e.sliceToEmailPart(e.BodyParts)
}

func (e *Mail) Attach(contentType string, data []byte) apperror.Error {
	e.Attachments = append(e.Attachments, &Part{MimeType: contentType, Content: data})
	return nil
}

func (e *Mail) AttachReader(contentType string, reader io.ReadCloser) apperror.Error {
	e.Attachments = append(e.Attachments, &Part{MimeType: contentType, Reader: reader})
	return nil
}

func (e *Mail) AttachFile(path string) apperror.Error {
	content, err := utils.ReadFile(path)
	if err != nil {
		return err
	}
	e.Attachments = append(e.Attachments, &Part{Content: content})
	return nil
}

func (e *Mail) GetAttachments() []EmailPart {
	return e.sliceToEmailPart(e.Attachments)
}

func (e *Mail) Embed(contentType string, data []byte) apperror.Error {
	e.EmbeddedAttachments = append(e.EmbeddedAttachments, &Part{MimeType: contentType, Content: data})
	return nil
}

func (e *Mail) EmbedReader(contentType string, reader io.ReadCloser) apperror.Error {
	e.EmbeddedAttachments = append(e.EmbeddedAttachments, &Part{MimeType: contentType, Reader: reader})
	return nil
}

func (e *Mail) EmbedFile(path string) apperror.Error {
	content, err := utils.ReadFile(path)
	if err != nil {
		return err
	}
	e.Attachments = append(e.Attachments, &Part{Content: content})
	return nil
}

func (e *Mail) GetEmbeddedAttachments() []EmailPart {
	return e.sliceToEmailPart(e.EmbeddedAttachments)
}

func (e *Mail) SetHeader(name string, values ...string) {
	e.Headers[name] = values
}

func (e *Mail) SetHeaders(data map[string][]string) {
	e.Headers = data
}
