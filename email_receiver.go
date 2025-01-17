package email

import (
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	id "github.com/emersion/go-imap-id"
	"github.com/emersion/go-imap/client"
)

type ReceiveEmailClient struct {
	server   string
	username string
	client   *client.Client
}

func NewReceiveEmailClient(ctx *dgctx.DgContext, host string, port int, username, password string) (*ReceiveEmailClient, error) {
	server := fmt.Sprintf("%s:%d", host, port)
	cli, err := client.DialTLS(server, nil)
	if err != nil {
		dglogger.Errorf(ctx, "dial imap server failed | server: %s | err: %v", server, err)
		return nil, err
	}

	err = cli.Login(username, password)
	if err != nil {
		dglogger.Errorf(ctx, "login imap server failed | server: %s | username: %s | err: %v", server, username, err)
		return nil, err
	}

	// some mail server need ID info
	idClient := id.NewClient(cli)
	_, err = idClient.ID(
		id.ID{
			id.FieldName:    "IMAPClient",
			id.FieldVersion: "3.1.0",
		},
	)

	return &ReceiveEmailClient{
		server:   server,
		username: username,
		client:   cli,
	}, nil
}

func (r *ReceiveEmailClient) Close() error {
	return r.client.Logout()
}
