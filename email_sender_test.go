package email

import (
	"os"
	"testing"

	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
)

func TestSendEmail(t *testing.T) {
	cli := NewSendEmailClient(os.Getenv("host"), 587, os.Getenv("username"), os.Getenv("password"))
	cli.UseMonitor = false
	ctx := &dgctx.DgContext{TraceId: "123"}
	err := cli.SendEmail(ctx, &SendEmailRequest{
		To:          []string{os.Getenv("to")},
		Subject:     "Test Subject",
		Content:     "<html><body>test body</body></html>",
		Attachments: []string{os.Getenv("attachment")},
	})
	dglogger.Infof(ctx, "err: %v", err)
}
