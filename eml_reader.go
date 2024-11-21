package email

import (
	"bytes"
	"fmt"
	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/jhillyerd/enmime/v2"
	"io"
	"os"
	"path"
)

func ExtractEmlAttachments(ctx *dgctx.DgContext, srcPath string, dstPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		dglogger.Errorf(ctx, "read eml file failed, err: %v", err)
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	envelope, err := enmime.ReadEnvelope(file)
	if err != nil {
		dglogger.Errorf(ctx, "read eml envelope failed, err: %v", err)
		return err
	}
	if len(envelope.Attachments) == 0 {
		return nil
	}

	_ = utils.CreateDir(dstPath)

	var attachmentFiles []*os.File
	for i, part := range envelope.Attachments {
		filename := part.FileName
		if filename == "" {
			filename = fmt.Sprintf("attachment_%d", i)
		}

		attachmentFile, err := os.Create(path.Join(dstPath, filename))
		if err != nil {
			dglogger.Errorf(ctx, "failed to create file for attachment %d: %v", i, err)
			continue
		}
		attachmentFiles = append(attachmentFiles, attachmentFile)

		_, err = io.Copy(attachmentFile, bytes.NewReader(part.Content))
		if err != nil {
			dglogger.Errorf(ctx, "failed to save attachment %d: %v", i, err)
			continue
		}

		dglogger.Debugf(ctx, "saved attachment %d: %s", i, filename)
	}

	if len(attachmentFiles) > 0 {
		for _, attachmentFile := range attachmentFiles {
			_ = attachmentFile.Close()
		}
	}

	return nil
}
