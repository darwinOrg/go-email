package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"sync"
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

	wg := sync.WaitGroup{}

	for i := 0; i < 1; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			receiveEmails(ctx, client)
		}()
	}

	wg.Wait()
}

func receiveEmails(ctx *dgctx.DgContext, client *ImapEmailClient) {
	startTime := time.Now().Add(-24 * time.Hour)
	criteria := &SearchCriteria{Since: startTime}
	_ = client.ReceiveEmails(ctx, criteria, 3, func(emailDTO *ReceiveEmailDTO) error {
		dglogger.Debugf(ctx, "emailDTO: %+v", emailDTO)
		return nil
	})
}
