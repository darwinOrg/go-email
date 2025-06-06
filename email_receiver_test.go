package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
	"time"
)

func TestSearchEmails(t *testing.T) {
	ctx := dgctx.SimpleDgContext()
	client, err := NewImapEmailClient(ctx, os.Getenv("host"), 993, os.Getenv("username"), os.Getenv("password"))
	if err != nil {
		panic(err)
	}

	startTime := time.Now().Add(-24 * time.Hour)
	req := &SearchEmailReq{Since: startTime}
	emails, err := client.SearchEmails(ctx, req)
	if err != nil {
		panic(err)
	}
	if emails == nil {
		return
	}

	for email := range emails {
		dglogger.Debugf(ctx, "email：%+v", email)
	}
}
