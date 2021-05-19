package audit

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	GH_OWNER = "cdnjs"
	GH_REPO  = "logs"
	GH_NAME  = "robocdnjs"
	GH_EMAIL = "cdnjs-github@cloudflare.com"
)

var (
	GH_TOKEN  = os.Getenv("AUDIT_GH_TOKEN")
	GH_BRANCH = os.Getenv("AUDIT_GH_BRANCH")
)

func getPath(pkgName, version, stage string) string {
	firstLetter := pkgName[0:1]
	return fmt.Sprintf("packages/%s/%s/%s/%s.log", firstLetter, pkgName, version, stage)
}

func getClient(ctx context.Context) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: GH_TOKEN},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func create(ctx context.Context, pkgName string, version string, stage string,
	content *bytes.Buffer) error {
	message := fmt.Sprintf("add %s %s (%s)", pkgName, version, stage)
	file := getPath(pkgName, version, stage)

	client := getClient(ctx)

	commitOption := &github.RepositoryContentFileOptions{
		Branch:  github.String(GH_BRANCH),
		Message: github.String(message),
		Committer: &github.CommitAuthor{
			Name:  github.String(GH_NAME),
			Email: github.String(GH_EMAIL),
		},
		Author: &github.CommitAuthor{
			Name:  github.String(GH_NAME),
			Email: github.String(GH_EMAIL),
		},
		Content: content.Bytes(),
	}

	c, resp, err := client.Repositories.CreateFile(ctx, GH_OWNER, GH_REPO, file, commitOption)
	if err != nil {
		if strings.Contains(err.Error(), "422 Invalid request") {
			// that errors means that the file already exists and we need to override it
			if err := override(ctx, pkgName, version, stage, content); err != nil {
				return errors.Wrap(err, "could not override audit")
			}
			return nil
		} else {
			return errors.Wrap(err, "could not create file")
		}
	}
	log.Printf("audit created: resp.Status=%v commit=%s", resp.Status, *c.SHA)
	return nil
}

func override(ctx context.Context, pkgName string, version string, stage string,
	content *bytes.Buffer) error {
	message := fmt.Sprintf("update %s %s (%s)", pkgName, version, stage)
	file := getPath(pkgName, version, stage)
	client := getClient(ctx)

	currContent, err := get(ctx, pkgName, version, stage)
	if err != nil {
		return errors.Wrap(err, "failed to get current file")
	}

	commitOption := &github.RepositoryContentFileOptions{
		Branch:  github.String(GH_BRANCH),
		Message: github.String(message),
		Committer: &github.CommitAuthor{
			Name:  github.String(GH_NAME),
			Email: github.String(GH_EMAIL),
		},
		Author: &github.CommitAuthor{
			Name:  github.String(GH_NAME),
			Email: github.String(GH_EMAIL),
		},
		Content: content.Bytes(),
		SHA:     currContent.SHA,
	}

	c, resp, err := client.Repositories.UpdateFile(ctx, GH_OWNER, GH_REPO, file, commitOption)
	if err != nil {
		return errors.Wrap(err, "could not override file")
	}
	log.Printf("audit overriden: resp.Status=%v commit=%s", resp.Status, *c.SHA)
	return nil
}

func get(ctx context.Context, pkgName string, version string, stage string) (*github.RepositoryContent, error) {
	client := getClient(ctx)
	file := getPath(pkgName, version, stage)
	opts := github.RepositoryContentGetOptions{
		Ref: GH_BRANCH,
	}
	res, _, _, err := client.Repositories.GetContents(ctx, GH_OWNER, GH_REPO, file, &opts)
	if err != nil {
		return nil, errors.Wrap(err, "could not get file")
	}
	return res, nil
}

func NewVersionDetected(ctx context.Context, pkgName string, version string) error {
	content := bytes.NewBufferString("")
	fmt.Fprintf(content, "New version: %s\n", version)

	if err := create(ctx, pkgName, version, "new-version", content); err != nil {
		return errors.Wrap(err, "could not create audit log file")
	}
	return nil
}

func ProcessedVersion(ctx context.Context, pkgName string, version string, log string) error {
	content := bytes.NewBufferString("")
	fmt.Fprintf(content, "%s", log)

	if err := create(ctx, pkgName, version, "processing", content); err != nil {
		return errors.Wrap(err, "could not create audit log file")
	}
	return nil
}

func WroteKV(ctx context.Context, pkgName string, version string,
	sris map[string]string, keys []string, config string) error {

	content := bytes.NewBufferString("")
	fmt.Fprintf(content, "config: %s\n", config)
	fmt.Fprint(content, "KV keys:\n")
	for _, key := range keys {
		fmt.Fprintf(content, "- %s\n", key)
	}
	fmt.Fprint(content, "SRIs:\n")
	for name, sri := range sris {
		fmt.Fprintf(content, "- %s: %s\n", name, sri)
	}

	if err := create(ctx, pkgName, version, "kv-publish", content); err != nil {
		return errors.Wrap(err, "could not create audit log file")
	}
	return nil
}
