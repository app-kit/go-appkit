package email

import(
	"io"

	log "github.com/Sirupsen/logrus"
	
	"github.com/theduke/go-appkit/error"
)

type EmailRecipient interface {
	GetEmail() string
	GetName() string
}

type EmailPart interface {
	GetMimeType() string
	GetContent() []byte
	GetFilePath() string
	GetReader() io.ReadCloser
}

type Email interface {
	SetFrom(email, name string)
	GetFrom() EmailRecipient

	AddTo(email, name string)
	GetTo() []EmailRecipient

	AddCc(email, name string)
	GetCc() []EmailRecipient

	AddBcc(email, name string)
	GetBcc() []EmailRecipient

	SetSubject(string)
	GetSubject() string

	SetBody(contentType string, body []byte)
	AddBody(contentType string, body []byte)
	GetBodyParts() []EmailPart

	Attach(contentType string, data []byte) error.Error
	AttachReader(contentType string, reader io.ReadCloser) error.Error
	AttachFile(path string) error.Error

	GetAttachments() []EmailPart

	Embed(contentType string, data []byte) error.Error
	EmbedReader(contentType string, reader io.ReadCloser) error.Error
	EmbedFile(path string) error.Error

	GetEmbeddedAttachments() []EmailPart

	SetHeader(name string, values ...string)
	SetHeaders(map[string][]string)
}

type EmailService interface {
	SetDefaultFrom(EmailRecipient)
	SetLogger(*log.Logger)

	Send(Email) error.Error
	SendMultiple(...Email) (error.Error, []error.Error)
}
