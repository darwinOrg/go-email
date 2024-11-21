package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
)

func TestSendEmail(t *testing.T) {
	sec := NewSendEmailClient(os.Getenv("host"), 587, os.Getenv("username"), os.Getenv("password"))
	sec.UseMonitor = false
	ctx := &dgctx.DgContext{TraceId: "123"}
	err := sec.SendEmail(ctx, &SendEmailRequest{
		To:          []string{os.Getenv("to")},
		Subject:     "Test Subject",
		Content:     "<html><body>test body</body></html>",
		Attachments: []string{os.Getenv("attachment")},
	})
	dglogger.Infof(ctx, "err: %v", err)
}
