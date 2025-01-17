package email

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/emersion/go-imap"
	id "github.com/emersion/go-imap-id"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"io"
	"log"
	"os"
	"time"
)

type ReceiveEmailClient struct {
	server   string
	username string
	client   *client.Client
}

type SearchEmailReq struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type ReceiveEmailDTO struct {
	EmailAddress string   `json:"emailAddress"` // 接收邮件地址
	Scene        string   `json:"scene"`        // 场景
	SendDate     string   `json:"sendDate"`     // 日期
	ReceiveDate  string   `json:"receiveDate"`  // 日期
	Subject      string   `json:"subject"`      // 主题
	ToName       []string `json:"toName"`       // 收件人
	ToAddress    []string `json:"toAddress"`    // 收件人
	FromAddress  string   `json:"fromAddress"`  // 发送人
	FromName     string   `json:"fromName"`     // 发件邮箱地址
	CcName       []string `json:"ccName"`       // 抄送人
	CcAddress    []string `json:"ccAddress"`    // 抄送人
	Content      string   `json:"content"`      // 正文
	ReplyName    []string `json:"replyName"`    // 回复人
	ReplyAddress []string `json:"replyAddress"` // 回复地址
	Attachments  []string `json:"attachments"`  // 附件列表
}

func NewReceiveEmailClient(ctx *dgctx.DgContext, host string, port int, username, password string) (*ReceiveEmailClient, error) {
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

	return &ReceiveEmailClient{
		server:   server,
		username: username,
		client:   cli,
	}, nil
}

func (r *ReceiveEmailClient) SearchEmails(ctx *dgctx.DgContext, req *SearchEmailReq) ([]*ReceiveEmailDTO, error) {
	searchCriteria := imap.NewSearchCriteria()

	if req.StartDate != "" {
		startDate, err := time.Parse(time.DateOnly, req.StartDate)
		if err != nil {
			dglogger.Errorf(ctx, "parse start date failed | date: %s | err: %v", req.StartDate, err)
			return nil, err
		}
		searchCriteria.Since = startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse(time.DateOnly, req.EndDate)
		if err != nil {
			dglogger.Errorf(ctx, "parse start date failed | date: %s | err: %v", req.EndDate, err)
			return nil, err
		}
		// 因为Before是非包含的，所以加一天
		endDate = endDate.AddDate(0, 0, 1)
		searchCriteria.Before = endDate
	}

	seqNums, err := r.client.Search(searchCriteria)
	if err != nil {
		dglogger.Errorf(ctx, "search email failed | err: %v", err)
		return nil, err
	}
	if len(seqNums) == 0 {
		return []*ReceiveEmailDTO{}, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddRange(seqNums[0], seqNums[len(seqNums)-1])

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- r.client.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}, messages)
	}()

logMessages:
	for {
		select {
		case msg := <-messages:
			body := msg.GetBody(&imap.BodySectionName{})
			if body == nil {
				log.Fatal("Server didn't return message body")
			}

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
					dglogger.Debugf(ctx, "found attachment: %s", filename)

					outFile, err := os.Create(filename)
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
				}
			}
		case e := <-done:
			if e != nil {
				dglogger.Errorf(ctx, "fetch email failed | err: %v", e)
			}
			break logMessages
		}
	}

	return nil, nil
}

func (r *ReceiveEmailClient) Close() error {
	return r.client.Logout()
}
