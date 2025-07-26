package main

import (
	"os"

	"project-template/docs"
	"project-template/infrastructure/config"
)

func main() {
	cfgPath := "config.yaml"
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	var cfg *config.Config
	if err := config.LoadConfig(cfgPath); err == nil {
		cfg = config.Cfg
	}

	docs.BuildSpec(cfg)
	_ = os.WriteFile("docs/openapi.json", docs.SpecBytes(), 0644)
	_ = os.WriteFile("docs/openapi.yaml", docs.YAMLSpecBytes(), 0644)
}
