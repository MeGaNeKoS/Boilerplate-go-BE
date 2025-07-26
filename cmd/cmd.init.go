package cmd

import (
	"flag"
	"fmt"
	"log"

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
)

var (
	gitCommit string
	buildTime string
	branch    string
)

// printVersion outputs build information to stdout.
func printVersion() {
	if branch == "" {
		branch = "unknown"
	}
	if gitCommit == "" {
		gitCommit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}

	fmt.Printf("Branch: %s\nGit Commit: %s\nBuild Time: %s\n", branch, gitCommit, buildTime)
}

// parseFlags handles common CLI flags used by all server entry points.
// It returns the config file path and true when the program should exit
// after printing the version information.
func parseFlags(doneChan chan struct{}) (string, bool) {
	showVersion := flag.Bool("version", false, "Show build version and exit")
	showShort := flag.Bool("v", false, "Show build version and exit (shorthand)")
	filePath := flag.String("config", "config.yaml", "Path to the configuration file")

	flag.Parse()
	if *showVersion || *showShort {
		printVersion()
		if doneChan != nil {
			close(doneChan)
		}
		return "", true
	}
	if filePath == nil {
		log.Fatalf("Failed to load config.yaml")
	}
	return *filePath, false
}

// initDependencies loads configuration and prepares shared services.
func initDependencies(filePath string) (logger.Logger, error) {
	if err := config.LoadConfig(filePath); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	loggerInstance, err := logger.NewLogger(config.Cfg.LogTarget, "-1", "-1")
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}

	database := db.GetDBInstance()
	database.SetLogger(loggerInstance)
	if err = database.New(); err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}
	if err = db.RunMigrations(config.Cfg.Database.MigrationPath); err != nil {
		return nil, fmt.Errorf("db migrate: %w", err)
	}

	if err = utils.InitializeJWTService(false, false); err != nil {
		return nil, fmt.Errorf("init jwt: %w", err)
	}
	return loggerInstance, nil
}
