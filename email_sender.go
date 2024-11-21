package email

import (
	"crypto/tls"
	"encoding/json"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/darwinOrg/go-monitor"
	"gopkg.in/gomail.v2"
	"time"
)

type SendEmailClient struct {
	dialer     *gomail.Dialer
	UseMonitor bool
}

type SendEmailRequest struct {
	To          []string
	Subject     string
	Content     string
	Attachments []string
}

func NewSendEmailClient(host string, port int, username string, password string) *SendEmailClient {
	dialer := &gomail.Dialer{
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		SSL:       false,
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &SendEmailClient{dialer: dialer, UseMonitor: true}
}

func (sec *SendEmailClient) SendEmail(ctx *dgctx.DgContext, request *SendEmailRequest) error {
	if sec.UseMonitor {
		monitor.HttpClientCounter(sec.dialer.Host)
	}
	start := time.Now().UnixMilli()

	m := gomail.NewMessage()
	m.SetHeader("From", sec.dialer.Username)
	m.SetHeader("To", request.To...)
	m.SetHeader("Subject", request.Subject)
	m.SetBody("text/html", request.Content)

	if request.Attachments != nil && len(request.Attachments) > 0 {
		for _, attachment := range request.Attachments {
			m.Attach(attachment)
		}
	}

	requestJson, _ := json.Marshal(request)
	err := sec.dialer.DialAndSend(m)
	cost := time.Now().UnixMilli() - start

	if sec.UseMonitor {
		e := "false"
		if err != nil {
			e = "true"
		}
		monitor.HttpClientDuration(sec.dialer.Host, e, cost)
	}

	dglogger.Infof(ctx, "send email: %s, cost: %d ms", requestJson, cost)

	return err
}
