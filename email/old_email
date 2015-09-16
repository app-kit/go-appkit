package email

import (
	//"crypto/tls"
	"strconv"
	"strings"
	"time"
	//"net"
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"

	"github.com/mxk/go-imap/imap"
	"gopkg.in/gomail.v2-unstable"
)

type Client struct {
	Username string
	Password string

	ImapHost   string
	ImapPort   int
	ImapClient *imap.Client
}

func NewClient() (*Client, error) {
	c := Client{}

	if err := c.Init(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Client) Init() error {
	if c.Username == "" {
		return errors.New("no_username")
	}
	if c.Password == "" {
		return errors.New("no_password")
	}
	if c.ImapHost == "" {
		return errors.New("no_imap_host")
	}

	return nil
}

func (c *Client) ImapConnect() error {
	client, err := imap.Dial(c.ImapHost + ":" + strconv.Itoa(c.ImapPort))
	if err != nil {
		return err
	}

	_, err = imap.Wait(client.Login(c.Username, c.Password))
	if err != nil {
		return err
	}

	c.ImapClient = client
	return nil
}

func (c *Client) MarkMessageSeen(mailbox string, uid uint32) error {
	_, err := c.ImapClient.Select(mailbox, false)
	if err != nil {
		return err
	}

	set, _ := imap.NewSeqSet("")
	set.AddNum(uint32(uid))

	_, err = imap.Wait(c.ImapClient.UIDStore(set, "+FLAGS", "(\\Seen)"))
	if err != nil {
		return err
	}
	_, err = imap.Wait(c.ImapClient.UIDFetch(set, "FLAGS"))

	return err
}

func (c *Client) GetStatus(mailbox string) (*imap.MailboxStatus, error) {
	cmd, err := imap.Wait(c.ImapClient.Status("INBOX", "MESSAGES", "UNSEEN"))
	if err != nil {
		return nil, err
	}

	return cmd.Data[0].MailboxStatus(), nil
}

func (c *Client) GetMessage(id uint32) (*Email, error) {
	mails, err := c.GetMessages("INBOX", true, []uint32{id})
	if err != nil {
		return nil, err
	}

	if len(mails) < 1 {
		return nil, errors.New("not_found")
	}

	return &mails[0], nil
}

func (c *Client) GetMessages(mailbox string, withBody bool, ids []uint32) ([]Email, error) {
	cmd, err := c.ImapClient.Select(mailbox, true)
	if err != nil {
		return nil, err
	}

	set, _ := imap.NewSeqSet("")
	if ids == nil {
		set.Add("1:*")
	} else {
		for _, id := range ids {
			set.AddNum(id)
		}
	}

	if withBody {
		cmd, err = imap.Wait(c.ImapClient.UIDFetch(set, "FLAGS", "UID", "BODY.PEEK[HEADER.FIELDS (SUBJECT DATE FROM)]", "RFC822"))
	} else {
		cmd, err = imap.Wait(c.ImapClient.UIDFetch(set, "FLAGS", "UID", "BODY.PEEK[HEADER.FIELDS (SUBJECT DATE FROM)]"))
	}
	if err != nil {
		return nil, err
	}

	emails := make([]Email, 0, len(cmd.Data))

	for _, data := range cmd.Data {
		info := data.MessageInfo()

		flags := imap.AsFlagSet(info.Attrs["FLAGS"])
		log.Printf("flags: %v\n", flags)
		read := flags["\\Seen"]

		field := info.Attrs["BODY[HEADER.FIELDS (SUBJECT DATE FROM)]"]

		tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(imap.AsBytes(field))))
		header, err := tp.ReadMIMEHeader()
		if err != nil {
			fmt.Printf("Could not read header: %v\n", err)
			continue
		}

		// Mon, 02 Sep 2013 06:48:05 +0200
		date, _ := time.Parse(time.RFC1123Z, header.Get("Date"))

		email := Email{
			UID:     info.UID,
			Subject: DecodeStr(header.Get("Subject")),
			From:    DecodeStr(header.Get("From")),
			Date:    date,
			Read:    read,
		}

		if withBody {
			fullData := imap.AsBytes(info.Attrs["RFC822"])

			if msg, _ := mail.ReadMessage(bytes.NewReader(fullData)); msg != nil {
				//fmt.Printf("%+v\n",msg.Header)

				mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))

				isMulti := err == nil && strings.HasPrefix(mediaType, "multipart/")
				if isMulti {
					email.Attachments = ExtractAttachments(info.UID, params, msg)
					log.Printf("Attachments: %v\n", len(email.Attachments))

					// Look for plain + html content.
					for index, att := range email.Attachments {
						if att.Mime == "text/plain" {
							email.Body = att.StrContent
							email.Attachments = append(email.Attachments[:index], email.Attachments[index+1:]...)
							break
						}
					}

					for index, att := range email.Attachments {
						if att.Mime == "text/html" {
							email.HtmlBody = att.StrContent
							email.Attachments = append(email.Attachments[:index], email.Attachments[index+1:]...)
							break
						}
					}
				} else {
					rawBody, _ := ioutil.ReadAll(msg.Body)
					email.Body = DecodeStr(string(rawBody))
				}
			}
		}

		emails = append(emails, email)
	}

	return emails, nil
}

func ExtractAttachments(msgId uint32, params map[string]string, msg *mail.Message) []EmailAttachment {
	attachments := make([]EmailAttachment, 0)

	mr := multipart.NewReader(msg.Body, params["boundary"])
	index := 0
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		content, err := ioutil.ReadAll(p)
		if err != nil || content == nil {
			continue
		}

		fmt.Printf("Attachment header:\n %v\n\n", p.Header)

		rawContentType, ok := p.Header["Content-Type"]
		if !ok {
			log.Printf("Attachment is missing Content-Type")
			continue
		}
		contentType := StrBefore(rawContentType[0], ";")

		encoding := ""
		rawEncoding, ok := p.Header["Content-Transfer-Encoding"]
		if ok {
			encoding = rawEncoding[0]
		} else {
			encoding = strings.TrimSpace(ExtractAttr(rawContentType[0], "charset"))
		}

		if encoding == "" {
			log.Printf("Could not determine attachment encoding.")
			continue
		}

		attachment := EmailAttachment{
			Encoding: encoding,
			Mime:     contentType,
			EmailID:  strconv.FormatUint(uint64(msgId), 10),
			ID:       fmt.Sprintf("%v-%v", msgId, index),
		}
		index += 1

		filename := ExtractAttr(contentType, "file")
		if filename == "" {
			rawDisposition, ok := p.Header["Content-Disposition"]
			if ok {
				filename = ExtractAttr(rawDisposition[0], "filename")
			}
		}

		if filename != "" {
			attachment.Filename = filename
		}

		if contentType == "text/plain" || encoding == "utf-8" {
			attachment.StrContent = DecodeStr(string(content))
		} else if encoding == "base64" {
			attachment.Content = string(content)
		} else {
			fmt.Printf("Unhandled encoding: %v\n", encoding)
			continue
		}

		isHtml := strings.HasSuffix(filename, "html") || strings.HasSuffix(filename, "htm") || contentType == "text/html"
		if isHtml {
			if encoding == "base64" {

				rawDecodedContent := make([]byte, len(content))
				length, err := base64.StdEncoding.Decode(rawDecodedContent, content)
				if err == nil {
					decodedContent := string(rawDecodedContent[:length])
					attachment.StrContent = decodedContent
					attachment.Content = ""
					attachment.Mime = "text/html"
					attachment.Encoding = ""
				} else {
					fmt.Printf("Decoding error: %v\n", err)
				}
			}
		}

		attachments = append(attachments, attachment)

	}

	return attachments
}
