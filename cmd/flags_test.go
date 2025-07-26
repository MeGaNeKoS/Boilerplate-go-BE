package cmd

import (
	"flag"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/bouk/monkey"
)

func TestPrintVersion(t *testing.T) {
	branch = "b"
	gitCommit = "c"
	buildTime = "t"
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	printVersion()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	out := string(data)
	if !strings.Contains(out, "Branch: b") || !strings.Contains(out, "Git Commit: c") {
		t.Fatalf("unexpected output %s", out)
	}
}

func TestParseFlagsVersion(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine = fs
	os.Args = []string{"cmd", "--version"}
	done := make(chan struct{})
	p, exit := parseFlags(done)
	if !exit || p != "" {
		t.Fatalf("unexpected %v %v", exit, p)
	}
	select {
	case <-done:
	default:
		t.Fatal("done not closed")
	}
}

func TestParseFlagsConfig(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine = fs
	os.Args = []string{"cmd", "--config", "p.yaml"}
	p, exit := parseFlags(nil)
	if exit || p != "p.yaml" {
		t.Fatalf("unexpected %v %v", exit, p)
	}
}

func TestParseFlagsNilConfig(t *testing.T) {
	patchStr := monkey.PatchInstanceMethod(reflect.TypeOf(flag.CommandLine), "String", func(_ *flag.FlagSet, name, value, usage string) *string {
		return nil
	})
	patchExit := monkey.Patch(os.Exit, func(code int) { panic("exit") })
	defer patchStr.Unpatch()
	defer patchExit.Unpatch()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine = fs
	os.Args = []string{"cmd"}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected os.Exit")
		}
	}()
	parseFlags(nil)
}

func TestPrintVersionDefaults(t *testing.T) {
	branch, gitCommit, buildTime = "", "", ""
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	printVersion()
	w.Close()
	os.Stdout = old
	data, _ := io.ReadAll(r)
	out := string(data)
	if !strings.Contains(out, "Branch: unknown") || !strings.Contains(out, "Build Time: unknown") {
		t.Fatalf("unexpected output %s", out)
	}
}

func TestStartCommandsVersion(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	flag.CommandLine = fs
	os.Args = []string{"cmd", "--version"}
	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}
