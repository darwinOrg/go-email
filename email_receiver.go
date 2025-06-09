package email

import (
	"bytes"
	"fmt"
	dgcoll "github.com/darwinOrg/go-common/collection"
	dgctx "github.com/darwinOrg/go-common/context"
	dgerr "github.com/darwinOrg/go-common/enums/error"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/emersion/go-imap"
	id "github.com/emersion/go-imap-id"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"io"
	"path"
	"strings"
	"time"
)

var (
	bodySection = &imap.BodySectionName{}
	fetchItems  = []imap.FetchItem{bodySection.FetchItem(), imap.FetchEnvelope, imap.FetchUid}

	serverConfigError    = dgerr.SimpleDgError("服务器配置错误")
	accountPasswordError = dgerr.SimpleDgError("账号密码错误")
	searchEmailError     = dgerr.SimpleDgError("搜索邮件错误")
	selectInboxError     = dgerr.SimpleDgError("选择收件箱错误")
)

type ImapEmailClient struct {
	server   string
	username string
	client   *client.Client
}

type SearchCriteria struct {
	// Time and timezone are ignored
	Since      *time.Time // Internal date is since this date
	Before     *time.Time // Internal date is before this date
	SentSince  *time.Time // Date header field is since this date
	SentBefore *time.Time // Date header field is before this date

	Body []string // Each string is in the body
	Text []string // Each string is in the text (header + body)

	WithFlags    []string // Each flag is present
	WithoutFlags []string // Each flag is not present

	Subject           string   // 邮件主题，包含
	NeedBody          bool     // 返回结果是否需要Body
	AttachmentExtends []string // 附近扩展名
}

type ReceiveEmailDTO struct {
	EmailAddress string        `json:"emailAddress"` // 接收邮件地址
	SendDate     string        `json:"sendDate"`     // 发送日期
	Subject      string        `json:"subject"`      // 主题
	ToName       []string      `json:"toName"`       // 收件人名称
	ToAddress    []string      `json:"toAddress"`    // 收件人地址
	FromName     string        `json:"fromName"`     // 发送人名称
	FromAddress  string        `json:"fromAddress"`  // 发送人地址
	CcName       []string      `json:"ccName"`       // 抄送人名称
	CcAddress    []string      `json:"ccAddress"`    // 抄送人地址
	Content      string        `json:"content"`      // 正文
	ReplyName    []string      `json:"replyName"`    // 回复人名称
	ReplyAddress []string      `json:"replyAddress"` // 回复人地址
	Attachments  []*Attachment `json:"attachments"`  // 附件列表
}

type Attachment struct {
	FileName string `json:"fileName"` // 文件名称
	Body     []byte `json:"-"`        // 文件内容
}

func init() {
	imap.CharsetReader = charset.Reader
}

func NewImapEmailClient(ctx *dgctx.DgContext, host string, port int, username, password string) (*ImapEmailClient, error) {
	server := fmt.Sprintf("%s:%d", host, port)
	cli, err := client.DialTLS(server, nil)
	if err != nil {
		dglogger.Errorf(ctx, "dial imap server failed | server: %s | err: %v", server, err)
		return nil, serverConfigError
	}

	err = cli.Login(username, password)
	if err != nil {
		dglogger.Errorf(ctx, "login imap server failed | server: %s | username: %s | err: %v", server, username, err)
		return nil, accountPasswordError
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

func (c *ImapEmailClient) ReceiveEmails(ctx *dgctx.DgContext, criteria *SearchCriteria, maxRetryTimes int, handler func(emailDTO *ReceiveEmailDTO) error) error {
	sc := imap.NewSearchCriteria()

	if criteria.Since != nil {
		sc.Since = *criteria.Since
	}

	if criteria.Before != nil {
		sc.Before = *criteria.Before
	}

	if criteria.SentSince != nil {
		sc.SentSince = *criteria.SentSince
	}

	if criteria.SentBefore != nil {
		sc.SentBefore = *criteria.SentBefore
	}

	sc.Body = criteria.Body
	sc.Text = criteria.Text
	sc.WithFlags = criteria.WithFlags
	sc.WithoutFlags = criteria.WithoutFlags

	messages, err := c.SearchByCriteria(ctx, sc)
	if err != nil || messages == nil {
		return err
	}

	for message := range messages {
		go func() {
			emailDTO, err := filterAndParseMessage(ctx, message, criteria)
			if err != nil || emailDTO == nil {
				return
			}

			for i := 0; i < maxRetryTimes; i++ {
				if err := handler(emailDTO); err == nil {
					break
				} else {
					time.Sleep(time.Duration(utils.RandomIntInRange(1000, 5000)) * time.Millisecond)
				}
			}
		}()
	}

	return nil
}

func (c *ImapEmailClient) SearchByCriteria(ctx *dgctx.DgContext, criteria *imap.SearchCriteria) (chan *imap.Message, error) {
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
		return nil, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(seqNums[0], seqNums[len(seqNums)-1]+1)

	messages := make(chan *imap.Message, 10)
	go func() {
		_ = c.client.Fetch(seqSet, fetchItems, messages)
	}()

	return messages, nil
}

func (c *ImapEmailClient) SearchTest(ctx *dgctx.DgContext) error {
	_, err := c.client.Select("INBOX", true)
	if err != nil {
		dglogger.Errorf(ctx, "select inbox failed | err: %v", err)
		return selectInboxError
	}

	criteria := &imap.SearchCriteria{SentSince: time.Now().Add(24 * time.Hour)}
	_, err = c.client.Search(criteria)
	if err != nil {
		dglogger.Errorf(ctx, "search email failed | err: %v", err)
		return searchEmailError
	}

	return nil
}

func (c *ImapEmailClient) Close() error {
	return c.client.Logout()
}

func filterAndParseMessage(ctx *dgctx.DgContext, msg *imap.Message, criteria *SearchCriteria) (*ReceiveEmailDTO, error) {
	if msg == nil {
		return nil, nil
	}

	envelope := msg.Envelope

	if criteria.Subject != "" {
		if !strings.Contains(envelope.Subject, criteria.Subject) {
			return nil, nil
		}
	}

	if criteria.Since != nil {
		if envelope.Date.Before(*criteria.Since) {
			return nil, nil
		}
	}

	if criteria.SentSince != nil {
		if envelope.Date.Before(*criteria.SentSince) {
			return nil, nil
		}
	}

	emailDTO := &ReceiveEmailDTO{}

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

	body := msg.GetBody(&imap.BodySectionName{})
	if body == nil {
		return nil, nil
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

			if filename != "" {
				if len(criteria.AttachmentExtends) > 0 {
					fileExt := path.Ext(filename)
					if !dgcoll.Contains(criteria.AttachmentExtends, strings.ToLower(fileExt)) {
						return nil, nil
					}
				}

				buf, err := copyBodyToBytesBuffer(ctx, p.Body)
				if err != nil {
					continue
				}

				emailDTO.Attachments = append(emailDTO.Attachments, &Attachment{
					FileName: filename,
					Body:     buf.Bytes(),
				})
			}
		default:
			if criteria.NeedBody {
				buf, err := copyBodyToBytesBuffer(ctx, p.Body)
				if err != nil {
					continue
				}
				emailDTO.Content += buf.String()
			}
		}
	}

	return emailDTO, nil
}

func copyBodyToBytesBuffer(ctx *dgctx.DgContext, body io.Reader) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, body); err != nil {
		dglogger.Errorf(ctx, "copy body failed | err: %v", err)
		return nil, err
	}

	return buf, nil
}
