// Command okf keeps a repository's knowledge bundle: it sets the bundle up
// with the shared baseline, checks it, serves it for review, and handles
// the reviewer's change requests.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/salkz/okf-kit/internal/kit"
	"github.com/salkz/okf-kit/internal/okf"
	"github.com/salkz/okf-kit/internal/review"
)

// version is set by the release build. A build from source is "dev".
var version = "dev"

// releaseVersion matches a plain release tag such as v1.2.3.
var releaseVersion = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// init takes the version from the module when the binary was not built by
// the release workflow, as with "go run github.com/salkz/okf-kit/cmd/okf@v1.2.3".
func init() {
	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && releaseVersion.MatchString(info.Main.Version) {
		version = info.Main.Version
	}
}

const usage = `usage: okf <command> [flags]

In a repository:
  init       set up the baseline and an empty local bundle
  update     move the baseline and managed files to this version of okf
  status     say whether a newer baseline has been released
  check      check the bundle; exits 1 if anything is wrong

Reviewing:
  serve      serve the bundle for review (-reviewer <id>, -addr, -bundle)
  requests   list the reviewer's open change requests
  resolve    mark a change request done (-id, -by, -response)

  version    print the version of okf

Commands run in the current directory, which must be the repository root.
`

func main() {
	log.SetFlags(0)
	log.SetPrefix("okf: ")
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

// errProblems is returned by check when the bundle is not sound; the
// problems themselves are already printed.
type errProblems int

func (e errProblems) Error() string {
	return fmt.Sprintf("%d problem(s)", int(e))
}

func run(command string, args []string, w io.Writer) error {
	now := time.Now()
	switch command {
	case "init":
		return kit.Init(".", version, now, w)
	case "update":
		return kit.Update(".", version, now, w)
	case "status":
		return kit.Status(".", w, kit.CacheFile(), now, kit.LatestRelease)
	case "check":
		return check(args, w)
	case "serve":
		return review.Serve(args, reviewOptions("."))
	case "requests":
		return review.ListRequests(w, args)
	case "resolve":
		return review.ResolveRequest(w, args, now)
	case "version", "-version", "--version":
		fmt.Fprintln(w, version)
		return nil
	case "help", "-h", "-help", "--help":
		fmt.Fprint(w, usage)
		return nil
	}
	return fmt.Errorf("unknown command %q\n\n%s", command, usage)
}

// reviewOptions makes the baseline read-only in a repository that uses the
// kit. In the kit's own repository there is no lock, and the baseline is
// reviewed like everything else.
func reviewOptions(repo string) review.Options {
	if lock, err := kit.ReadLock(repo); err != nil || lock == nil {
		return review.Options{}
	}
	return review.Options{Baseline: kit.BaselineDir, Upstream: kit.Upstream}
}

func check(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	upstream := fs.Bool("upstream", false, "for the kit's own repository: the baseline is edited here, so there is no okf.lock to hold it to")
	if err := fs.Parse(args); err != nil {
		return err
	}
	bundle := filepath.Join(".", kit.BundleDir)
	problems, err := okf.Check(bundle)
	if err != nil {
		return err
	}
	more, err := kit.Check(".", version, *upstream)
	if err != nil {
		return err
	}
	problems = append(problems, more...)
	for _, p := range problems {
		fmt.Fprintln(w, p)
	}
	if len(problems) > 0 {
		return errProblems(len(problems))
	}

	docs, err := okf.Load(bundle)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if d.Fields["status"] == "draft" {
			fmt.Fprintf(w, "note: %s is still a draft\n", d.Path)
		}
	}
	fmt.Fprintf(w, "The knowledge bundle is sound: %d concepts.\n", len(docs))
	return nil
}
