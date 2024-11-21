package email

import (
	dgctx "github.com/darwinOrg/go-common/context"
	"os"
	"testing"
)

func TestExtractEmlAttachments(t *testing.T) {
	srcPath := os.Getenv("srcPath")
	dstPath := os.Getenv("dstPath")

	err := ExtractEmlAttachments(dgctx.SimpleDgContext(), srcPath, dstPath)
	if err != nil {
		panic(err)
	}
}
