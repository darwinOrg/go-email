package email

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"io"
	"log"
	"time"
)

type ImapEmailClient struct {
	server   string
	username string
	client   *imapclient.Client
}

type SearchEmailReq struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
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
	cli, err := imapclient.DialTLS(server, &imapclient.Options{})
	if err != nil {
		dglogger.Errorf(ctx, "dial imap server failed | server: %s | err: %v", server, err)
		return nil, err
	}

	err = cli.Login(username, password).Wait()
	if err != nil {
		dglogger.Errorf(ctx, "login imap server failed | server: %s | username: %s | err: %v", server, username, err)
		return nil, err
	}

	_, err = cli.Select("INBOX", nil).Wait()
	if err != nil {
		dglogger.Errorf(ctx, "select inbox failed | err: %v", err)
		return nil, err
	}

	// some mail server need ID info
	//idClient := id.NewClient(cli)
	//_, err = idClient.ID(
	//	id.ID{
	//		id.FieldName:    "IMAPClient",
	//		id.FieldVersion: "3.1.0",
	//	},
	//)

	return &ImapEmailClient{
		server:   server,
		username: username,
		client:   cli,
	}, nil
}

func (r *ImapEmailClient) SearchEmails(ctx *dgctx.DgContext, req *SearchEmailReq) ([]*ReceiveEmailDTO, error) {
	searchCriteria := &imap.SearchCriteria{}

	if req.StartDate != "" {
		startDate, err := time.Parse(time.DateOnly, req.StartDate)
		if err != nil {
			dglogger.Errorf(ctx, "parse start date failed | date: %s | err: %v", req.StartDate, err)
			return nil, err
		}
		searchCriteria.SentSince = startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(time.DateOnly, req.EndDate)
		if err != nil {
			dglogger.Errorf(ctx, "parse start date failed | date: %s | err: %v", req.EndDate, err)
			return nil, err
		}
		// 因为Before是非包含的，所以加一天
		endDate = endDate.AddDate(0, 0, 1)
		searchCriteria.SentBefore = endDate
	}

	searchData, err := r.client.Search(searchCriteria, &imap.SearchOptions{ReturnAll: true}).Wait()
	if err != nil {
		dglogger.Errorf(ctx, "search email failed | err: %v", err)
		return nil, err
	}
	if searchData.Count == 0 {
		return []*ReceiveEmailDTO{}, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(searchData.Min, searchData.Max)

	fetchOptions := &imap.FetchOptions{
		UID:           true,
		Envelope:      true,
		InternalDate:  true,
		BodySection:   []*imap.FetchItemBodySection{{}},
		BinarySection: []*imap.FetchItemBinarySection{{}},
	}

	fetchCmd := r.client.Fetch(seqSet, fetchOptions)
	defer func() {
		_ = fetchCmd.Close()
	}()

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		for {
			item := msg.Next()
			if item == nil {
				break
			}

			switch it := item.(type) {
			case imapclient.FetchItemDataUID:
				log.Printf("UID: %v", it.UID)
			case imapclient.FetchItemDataBodySection:
				b, err := io.ReadAll(it.Literal)
				if err != nil {
					log.Fatalf("failed to read body section: %v", err)
				}
				log.Printf("Body:\n%v", string(b))
			}
		}
	}

	//messages := make(chan *imap., 10)
	//done := make(chan error, 1)
	//go func() {
	//	fetchCmd := r.client.Fetch(seqSet, fetchOptions)
	//}()

	var emails []*ReceiveEmailDTO

	//logMessages:
	//	for {
	//		select {
	//		case msg := <-messages:
	//			emailDTO, pe := parseMessage(ctx, msg)
	//			if pe != nil {
	//				return nil, pe
	//			}
	//			if emailDTO == nil {
	//				continue
	//			}
	//			emails = append(emails, emailDTO)
	//		case de := <-done:
	//			if de != nil {
	//				dglogger.Errorf(ctx, "fetch email failed | err: %v", de)
	//				return nil, de
	//			}
	//			break logMessages
	//		}
	//	}

	return emails, nil
}

func (r *ImapEmailClient) Close() error {
	return r.client.Logout().Wait()
}

//func parseMessage(ctx *dgctx.DgContext, msg *imap.Message) (*ReceiveEmailDTO, error) {
//	if msg == nil {
//		return nil, nil
//	}
//
//	envelope := msg.Envelope
//
//	emailDTO := &ReceiveEmailDTO{}
//	// 基本信息
//	emailDTO.EmailAddress = envelope.To[0].Address()
//	emailDTO.SendDate = envelope.Date.String()
//	emailDTO.Subject = envelope.Subject
//
//	// 收件人
//	for _, addr := range envelope.To {
//		emailDTO.ToName = append(emailDTO.ToName, addr.PersonalName)
//		emailDTO.ToAddress = append(emailDTO.ToAddress, addr.Address())
//	}
//
//	// 发件人
//	fromAddr := envelope.From[0]
//	emailDTO.FromName = fromAddr.PersonalName
//	emailDTO.FromAddress = fromAddr.Address()
//
//	// 抄送人
//	for _, addr := range envelope.Cc {
//		emailDTO.CcName = append(emailDTO.CcName, addr.PersonalName)
//		emailDTO.CcAddress = append(emailDTO.CcAddress, addr.Address())
//	}
//
//	body := msg.GetBody(&imap.BodySectionName{})
//	if body == nil {
//		return nil, nil
//	}
//
//	// 解析邮件内容
//	msgReader, err := mail.CreateReader(body)
//	if err != nil {
//		dglogger.Errorf(ctx, "create mail reader failed | err: %v", err)
//		return nil, err
//	}
//
//	for {
//		p, err := msgReader.NextPart()
//		if err == io.EOF {
//			break
//		} else if err != nil {
//			dglogger.Errorf(ctx, "read mail part failed | err: %v", err)
//			return nil, err
//		}
//
//		switch h := p.Header.(type) {
//		case *mail.AttachmentHeader:
//			filename, _ := h.Filename()
//			emailDTO.Attachments = append(emailDTO.Attachments, filename)
//
//			randomLetter, _ := utils.RandomLetter(4)
//			outDir := path.Join(os.TempDir(), randomLetter)
//			_ = utils.CreateDir(outDir)
//
//			outFile, err := os.Create(path.Join(outDir, filename))
//			if err != nil {
//				dglogger.Errorf(ctx, "create attachment file failed | filename: %s | err: %v", filename, err)
//				return nil, err
//			}
//			defer func() {
//				_ = outFile.Close()
//			}()
//
//			if _, err := io.Copy(outFile, p.Body); err != nil {
//				dglogger.Errorf(ctx, "copy attachment file failed | filename: %s | err: %v", filename, err)
//				return nil, err
//			}
//		case *mail.InlineHeader:
//			buf := new(bytes.Buffer)
//			if _, err := io.Copy(buf, p.Body); err != nil {
//				dglogger.Errorf(ctx, "copy inline file failed | err: %v", err)
//				return nil, err
//			}
//			emailDTO.Content += buf.String()
//		default:
//			buf := new(bytes.Buffer)
//			if _, err := io.Copy(buf, p.Body); err != nil {
//				dglogger.Errorf(ctx, "copy default file failed | err: %v", err)
//				return nil, err
//			}
//			emailDTO.Content += buf.String()
//		}
//	}
//
//	return emailDTO, nil
//}
