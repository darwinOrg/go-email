package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	dglogger "github.com/darwinOrg/go-logger"
	"os"
	"testing"
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
