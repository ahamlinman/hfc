package cmd

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var statusCmd = &cobra.Command{
	Use:    "status",
	Short:  "Summarize the deployment status of all stacks",
	PreRun: initializePreRun,
	Run:    runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	const (
		minwidth = 1
		tabwidth = 8
		padding  = 2
		padchar  = ' '
		flags    = 0
	)
	tw := tabWriter{
		Writer: tabwriter.NewWriter(os.Stdout, minwidth, tabwidth, padding, padchar, flags),
	}
	defer func() {
		if err := tw.Flush(); err != nil {
			log.Fatal(err)
		}
	}()

	latestPackageRaw, err := os.ReadFile(rootState.LatestLambdaPackagePath())
	latestPackage := strings.TrimSpace(string(latestPackageRaw))
	switch {
	default:
		tw.WriteColumn("(build)")
		tw.WriteColumn(latestPackage)
		tw.EndLine()
	case errors.Is(err, fs.ErrNotExist):
		tw.WriteColumn("(build)")
		tw.WriteColumn("(none)")
		tw.EndLine()
	case err != nil:
		log.Fatal(err)
	}

	if len(rootConfig.Stacks) == 0 {
		return
	}

	cfnClient := cloudformation.NewFromConfig(awsConfig)
	var group errgroup.Group
	group.SetLimit(5) // TODO: This is arbitrary, is there a specific limit that makes sense?
	stackS3Keys := make([]string, len(rootConfig.Stacks))
	for i, stack := range rootConfig.Stacks {
		group.Go(func() error {
			// Errors here are intentionally not hard failures. One misconfigured or
			// not-yet-deployed stack should not prevent reporting for other stacks.
			if key, err := getStackS3Key(context.Background(), cfnClient, stack.Name); err == nil {
				stackS3Keys[i] = key
			}
			return nil
		})
	}
	group.Wait()

	for i, stack := range rootConfig.Stacks {
		tw.WriteColumn(stack.Name)

		key := stackS3Keys[i]
		if key == "" {
			tw.WriteColumn("(unknown)")
			tw.EndLine()
			continue
		}

		tw.WriteColumn(key)
		if key == latestPackage {
			tw.WriteColumn("(current)")
		} else {
			tw.WriteColumn("(not-current)")
		}
		tw.EndLine()
	}
}

type tabWriter struct {
	*tabwriter.Writer
	inLine bool
	err    error
}

func (b *tabWriter) Write(buf []byte) (n int, err error) {
	if b.err != nil {
		return 0, b.err
	}
	n, b.err = b.Writer.Write(buf)
	return n, b.err
}

func (b *tabWriter) Flush() error {
	if b.err != nil {
		return b.err
	}
	return b.Writer.Flush()
}

func (b *tabWriter) WriteColumn(s string) error {
	if b.inLine {
		b.Write([]byte("\t"))
	}
	b.Write([]byte(s))
	b.inLine = true
	return b.err
}

func (b *tabWriter) EndLine() error {
	b.Write([]byte("\n"))
	b.inLine = false
	return b.err
}
