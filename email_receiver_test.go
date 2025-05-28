package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
	"time"
)

func TestSearchEmails(t *testing.T) {
	ctx := &dgctx.DgContext{TraceId: "123"}
	cli, err := NewImapEmailClient(ctx, os.Getenv("host"), 993, os.Getenv("username"), os.Getenv("password"))
	if err != nil {
		panic(err)
	}

	startTime := time.Now().Add(time.Hour * 7)
	emails, err := cli.SearchEmails(ctx, &SearchEmailReq{
		SentSince: startTime,
	})

	dglogger.Infof(ctx, "emails: %v", emails)
}
