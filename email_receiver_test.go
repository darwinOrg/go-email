package email

import (
	"os"
	"testing"
	"time"

	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
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

	startTime := time.Now().Add(-1 * time.Minute).Local()
	criteria := &SearchCriteria{Since: startTime}
	_ = client.ReceiveEmails(ctx, criteria, 1, 3, func(emailDTO *ReceiveEmailDTO) error {
		dglogger.Debugf(ctx, "emailDTO: %+v", emailDTO)
		return nil
	})
}
