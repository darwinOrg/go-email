package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
	"time"
)

func TestReceiveEmails(t *testing.T) {
	ctx := dgctx.SimpleDgContext()
	client, err := NewImapEmailClient(ctx, os.Getenv("host"), 993, os.Getenv("username"), os.Getenv("password"))
	if err != nil {
		panic(err)
	}
	err = client.SearchTest(ctx)
	if err != nil {
		panic(err)
	}

	startTime := time.Now().Add(-24 * time.Hour)
	criteria := &SearchCriteria{SentSince: startTime, Subject: "设计、交互"}
	_ = client.ReceiveEmails(ctx, criteria, 3, func(emailDTO *ReceiveEmailDTO) error {
		dglogger.Debugf(ctx, "emailDTO: %+v", emailDTO)
		return nil
	})
}
