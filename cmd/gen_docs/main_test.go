package main

import (
	"errors"
	"os"
	"testing"

	"github.com/bouk/monkey"
	"project-template/docs"
	"project-template/infrastructure/config"
)

func TestMainDefaultPath(t *testing.T) {
	var calledCfg string
	patchLoad := monkey.Patch(config.LoadConfig, func(path string) error {
		calledCfg = path
		return errors.New("boom")
	})
	var gotCfg *config.Config
	patchBuild := monkey.Patch(docs.BuildSpec, func(c *config.Config) { gotCfg = c })
	patchJSON := monkey.Patch(docs.SpecBytes, func() []byte { return []byte("json") })
	patchYAML := monkey.Patch(docs.YAMLSpecBytes, func() []byte { return []byte("yaml") })
	files := []string{}
	patchWrite := monkey.Patch(os.WriteFile, func(name string, data []byte, perm os.FileMode) error {
		files = append(files, name)
		return nil
	})
	defer patchLoad.Unpatch()
	defer patchBuild.Unpatch()
	defer patchJSON.Unpatch()
	defer patchYAML.Unpatch()
	defer patchWrite.Unpatch()

	os.Args = []string{"gen_docs"}
	main()

	if calledCfg != "config.yaml" {
		t.Fatalf("load config path %s", calledCfg)
	}
	if gotCfg != nil {
		t.Fatalf("expected nil cfg")
	}
	if len(files) != 2 || files[0] != "docs/openapi.json" || files[1] != "docs/openapi.yaml" {
		t.Fatalf("files %v", files)
	}
}

func TestMainWithConfig(t *testing.T) {
	cfg := &config.Config{Version: "1"}
	patchLoad := monkey.Patch(config.LoadConfig, func(path string) error {
		config.Cfg = cfg
		return nil
	})
	var got *config.Config
	patchBuild := monkey.Patch(docs.BuildSpec, func(c *config.Config) { got = c })
	patchJSON := monkey.Patch(docs.SpecBytes, func() []byte { return []byte("x") })
	patchYAML := monkey.Patch(docs.YAMLSpecBytes, func() []byte { return []byte("y") })
	patchWrite := monkey.Patch(os.WriteFile, func(string, []byte, os.FileMode) error { return nil })
	defer patchLoad.Unpatch()
	defer patchBuild.Unpatch()
	defer patchJSON.Unpatch()
	defer patchYAML.Unpatch()
	defer patchWrite.Unpatch()

	os.Args = []string{"gen_docs", "cfg.yaml"}
	main()

	if got != cfg {
		t.Fatalf("expected cfg")
	}
}
