package email

import (
	"os"
	"testing"

	dgctx "github.com/darwinOrg/go-common/context"
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

	err = client.SearchTest(ctx)
	if err != nil {
		panic(err)
	}
}
