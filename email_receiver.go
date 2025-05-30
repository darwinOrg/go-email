package email

import (
	"bytes"
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/emersion/go-imap"
	id "github.com/emersion/go-imap-id"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"io"
	"os"
	"path"
	"time"
)

var (
	bodySection = &imap.BodySectionName{}
	fetchDetail = []imap.FetchItem{bodySection.FetchItem(), imap.FetchEnvelope, imap.FetchUid}
)

type ImapEmailClient struct {
	server   string
	username string
	client   *client.Client
}

type SearchEmailReq struct {
	// Time and timezone are ignored
	Since      time.Time // Internal date is since this date
	Before     time.Time // Internal date is before this date
	SentSince  time.Time // Date header field is since this date
	SentBefore time.Time // Date header field is before this date

	Body []string // Each string is in the body
	Text []string // Each string is in the text (header + body)

	WithFlags    []string // Each flag is present
	WithoutFlags []string // Each flag is not present
}

type ReceiveEmailDTO struct {
	EmailAddress string   `json:"emailAddress"` // 接收邮件地址
	SendDate     string   `json:"sendDate"`     // 发送日期
	Subject      string   `json:"subject"`      // 主题
	ToName       []string `json:"toName"`       // 收件人名称
	ToAddress    []string `json:"toAddress"`    // 收件人地址
	FromName     string   `json:"fromName"`     // 发送人名称
	FromAddress  string   `json:"fromAddress"`  // 发送人地址
	CcName       []string `json:"ccName"`       // 抄送人名称
	CcAddress    []string `json:"ccAddress"`    // 抄送人地址
	Content      string   `json:"content"`      // 正文
	ReplyName    []string `json:"replyName"`    // 回复人名称
	ReplyAddress []string `json:"replyAddress"` // 回复人地址
	Attachments  []string `json:"attachments"`  // 附件列表
}

func NewImapEmailClient(ctx *dgctx.DgContext, host string, port int, username, password string) (*ImapEmailClient, error) {
	server := fmt.Sprintf("%s:%d", host, port)
	cli, err := client.DialTLS(server, nil)
	if err != nil {
		dglogger.Errorf(ctx, "dial imap server failed | server: %s | err: %v", server, err)
		return nil, err
	}

	err = cli.Login(username, password)
	if err != nil {
		dglogger.Errorf(ctx, "login imap server failed | server: %s | username: %s | err: %v", server, username, err)
		return nil, err
	}

	// some mail server need ID info
	idClient := id.NewClient(cli)
	_, err = idClient.ID(
		id.ID{
			id.FieldName:    "IMAPClient",
			id.FieldVersion: "3.1.0",
		},
	)

	return &ImapEmailClient{
		server:   server,
		username: username,
		client:   cli,
	}, nil
}

func (c *ImapEmailClient) SearchEmails(ctx *dgctx.DgContext, req *SearchEmailReq) ([]*ReceiveEmailDTO, error) {
	criteria := imap.NewSearchCriteria()
	criteria.Since = req.Since
	criteria.Before = req.Before
	criteria.SentSince = req.SentSince
	criteria.SentBefore = req.SentBefore
	criteria.Body = req.Body
	criteria.Text = req.Text
	criteria.WithFlags = req.WithFlags
	criteria.WithoutFlags = req.WithoutFlags

	return c.SearchByCriteria(ctx, criteria)
}

func (c *ImapEmailClient) SearchByCriteria(ctx *dgctx.DgContext, criteria *imap.SearchCriteria) ([]*ReceiveEmailDTO, error) {
	// 选择收件箱
	_, err := c.client.Select("INBOX", true)
	if err != nil {
		dglogger.Errorf(ctx, "select inbox failed | err: %v", err)
		return nil, err
	}

	seqNums, err := c.client.Search(criteria)
	if err != nil {
		dglogger.Errorf(ctx, "search email failed | err: %v", err)
		return nil, err
	}
	if len(seqNums) == 0 {
		return []*ReceiveEmailDTO{}, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(seqNums[0], seqNums[len(seqNums)-1]+1)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.client.Fetch(seqSet, fetchDetail, messages)
	}()

	var emails []*ReceiveEmailDTO

logMessages:
	for {
		select {
		case msg := <-messages:
			emailDTO, pe := parseMessage(ctx, msg)
			if pe != nil {
				return nil, pe
			}
			if emailDTO == nil {
				continue
			}
			emails = append(emails, emailDTO)
		case de := <-done:
			if de != nil {
				dglogger.Errorf(ctx, "fetch email failed | err: %v", de)
				return nil, de
			}
			break logMessages
		}
	}

	return emails, nil
}

func (c *ImapEmailClient) Close() error {
	return c.client.Logout()
}

func parseMessage(ctx *dgctx.DgContext, msg *imap.Message) (*ReceiveEmailDTO, error) {
	if msg == nil {
		return nil, nil
	}

	body := msg.GetBody(&imap.BodySectionName{})
	if body == nil {
		return nil, nil
	}

	emailDTO := &ReceiveEmailDTO{}
	envelope := msg.Envelope

	// 基本信息
	emailDTO.EmailAddress = envelope.To[0].Address()
	emailDTO.SendDate = envelope.Date.String()
	emailDTO.Subject = envelope.Subject

	// 收件人
	for _, addr := range envelope.To {
		emailDTO.ToName = append(emailDTO.ToName, addr.PersonalName)
		emailDTO.ToAddress = append(emailDTO.ToAddress, addr.Address())
	}

	// 发件人
	fromAddr := envelope.From[0]
	emailDTO.FromName = fromAddr.PersonalName
	emailDTO.FromAddress = fromAddr.Address()

	// 抄送人
	for _, addr := range envelope.Cc {
		emailDTO.CcName = append(emailDTO.CcName, addr.PersonalName)
		emailDTO.CcAddress = append(emailDTO.CcAddress, addr.Address())
	}

	// 解析邮件内容
	msgReader, err := mail.CreateReader(body)
	if err != nil {
		dglogger.Errorf(ctx, "create mail reader failed | err: %v", err)
		return nil, err
	}

	for {
		p, err := msgReader.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			dglogger.Errorf(ctx, "read mail part failed | err: %v", err)
			return nil, err
		}

		switch h := p.Header.(type) {
		case *mail.AttachmentHeader:
			filename, _ := h.Filename()

			randomLetter, _ := utils.RandomLetter(8)
			outDir := path.Join(os.TempDir(), randomLetter)
			_ = utils.CreateDir(outDir)

			outPath := path.Join(outDir, filename)
			outFile, err := os.Create(outPath)
			if err != nil {
				dglogger.Errorf(ctx, "create attachment file failed | filename: %s | err: %v", filename, err)
				return nil, err
			}
			defer func() {
				_ = outFile.Close()
			}()

			if _, err := io.Copy(outFile, p.Body); err != nil {
				dglogger.Errorf(ctx, "copy attachment file failed | filename: %s | err: %v", filename, err)
				return nil, err
			}

			emailDTO.Attachments = append(emailDTO.Attachments, outPath)
		case *mail.InlineHeader:
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, p.Body); err != nil {
				dglogger.Errorf(ctx, "copy inline file failed | err: %v", err)
				return nil, err
			}
			emailDTO.Content += buf.String()
		default:
			buf := new(bytes.Buffer)
			if _, err := io.Copy(buf, p.Body); err != nil {
				dglogger.Errorf(ctx, "copy default file failed | err: %v", err)
				return nil, err
			}
			emailDTO.Content += buf.String()
		}
	}

	return emailDTO, nil
}
