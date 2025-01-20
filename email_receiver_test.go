package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
)

func TestSearchEmails(t *testing.T) {
	ctx := &dgctx.DgContext{TraceId: "123"}
	cli, err := NewImapEmailClient(ctx, os.Getenv("host"), 993, os.Getenv("username"), os.Getenv("password"))
	if err != nil {
		panic(err)
	}
	emails, err := cli.SearchEmails(ctx, &SearchEmailReq{
		StartDate: "2022-08-01",
		EndDate:   "2024-12-02",
	})
	dglogger.Infof(ctx, "emails: %v", emails)
}
