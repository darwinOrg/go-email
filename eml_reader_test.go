package email

import (
	"os"
	"testing"

	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
)

func TestExtractEmlContent(t *testing.T) {
	srcPath := os.Getenv("srcPath")
	dstPath := os.Getenv("dstPath")

	ec, err := ExtractEmlContent(dgctx.SimpleDgContext(), srcPath, dstPath)
	if err != nil {
		panic(err)
	}
	dglogger.Infof(dgctx.SimpleDgContext(), "ec: %v", ec)
}
