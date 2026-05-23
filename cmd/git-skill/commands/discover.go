package commands

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/niradler/git-skill/internal/gitops"
)

func Discover(p Profile, args []string, stdout, stderr io.Writer) error {
	if len(args) != 1 {
		return errors.New("discover <url>")
	}
	url := args[0]
	assets, err := gitops.ListRemote(url)
	if err != nil {
		return fmt.Errorf("ls-remote %s: %w", url, err)
	}
	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "KIND\tNAME\tLATEST_TAG\tCOMMIT")
	for _, ra := range assets {
		latest := "(none)"
		if len(ra.Versions) > 0 {
			latest = ra.Versions[len(ra.Versions)-1]
		}
		short := ra.Commit
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", ra.Kind, ra.Name, latest, short)
	}
	return tw.Flush()
}
