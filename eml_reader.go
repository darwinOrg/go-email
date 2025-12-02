package email

import (
	"bytes"
	"io"
	"os"
	"path"
	"strings"

	dgctx "github.com/darwinOrg/go-common/context"
	"github.com/darwinOrg/go-common/utils"
	dglogger "github.com/darwinOrg/go-logger"
	"github.com/jhillyerd/enmime/v2"
)

type EmlContent struct {
	Subject string `json:"subject"`
	From    string `json:"from"`
	To      string `json:"to"`
	Date    string `json:"date"`
	Text    string `json:"text"`
	Html    string `json:"html"`
}

func ExtractEmlContent(ctx *dgctx.DgContext, srcPath string, dstPath string) (*EmlContent, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		dglogger.Errorf(ctx, "read eml file failed, err: %v", err)
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	envelope, err := enmime.ReadEnvelope(file)
	if err != nil {
		dglogger.Errorf(ctx, "read eml envelope failed, err: %v", err)
		return nil, err
	}

	ec := &EmlContent{
		Subject: envelope.GetHeader("Subject"),
		From:    envelope.GetHeader("From"),
		To:      envelope.GetHeader("To"),
		Date:    envelope.GetHeader("Date"),
		Text:    envelope.Text,
		Html:    envelope.HTML,
	}
	_ = utils.CreateDir(dstPath)

	var files []*os.File
	var parts []*enmime.Part

	if envelope.Root != nil {
		parts = append(parts, envelope.Root)
	}
	if len(envelope.Attachments) > 0 {
		parts = append(parts, envelope.Attachments...)
	}
	if len(envelope.Inlines) > 0 {
		parts = append(parts, envelope.Inlines...)
	}
	if len(envelope.OtherParts) > 0 {
		parts = append(parts, envelope.OtherParts...)
	}

	for i, part := range parts {
		filename := part.FileName
		if filename == "" {
			continue
		}
		filename = strings.ReplaceAll(filename, " ", "_")
		emailFile, err := os.Create(path.Join(dstPath, filename))
		if err != nil {
			dglogger.Errorf(ctx, "failed to create file for part %d: %v", i, err)
			continue
		}
		files = append(files, emailFile)

		_, err = io.Copy(emailFile, bytes.NewReader(part.Content))
		if err != nil {
			dglogger.Errorf(ctx, "failed to save part file %d: %v", i, err)
			continue
		}

		dglogger.Debugf(ctx, "saved part %d: %s", i, filename)
	}

	if len(files) > 0 {
		for _, emailFile := range files {
			_ = emailFile.Close()
		}
	}

	return ec, nil
}
