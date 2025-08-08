package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"project-template/infrastructure/enums"
)

var openFile = func(name string) (io.ReadCloser, error) { return os.Open(name) }

type Config struct {
	AppName   string         `yaml:"AppName"`
	Version   string         `yaml:"Version"`
	Service   Service        `yaml:"ExternalService"`
	Server    ServerConfig   `yaml:"Server"`
	REST      ListenerConfig `yaml:"REST"`
	GRPC      ListenerConfig `yaml:"GRPC"`
	Kafka     KafkaConfig    `yaml:"Kafka"`
	Database  DatabaseConfig `yaml:"Database"`
	LogTarget LogConfig      `yaml:"LogTarget"`
	OpenAPI   OpenAPIConfig  `yaml:"OpenAPI"`
}

type Service struct {
	Example ServiceDetail `yaml:"Example"`
}

type GeoLocation struct {
	MaximumRadius int `yaml:"MaximumRadius"`
}

type ServiceDetail struct {
	Host string `yaml:"Host"`
}

type ServerConfig struct {
	Host          string         `yaml:"Host"`
	Port          string         `yaml:"Port"`
	Environment   string         `yaml:"Environment"`
	SkipTLSVerify bool           `yaml:"SkipTLSVerify"`
	Timeout       TimeoutConfig  `yaml:"Timeout"`
	Duration      int            `yaml:"Duration"`
	Endpoint      EndpointConfig `yaml:"Endpoint"`
	JWT           JWTConfig      `yaml:"JWT"`
}

// ListenerConfig holds host and port details for a network listener.
type ListenerConfig struct {
	Host string `yaml:"Host"`
	Port string `yaml:"Port"`
}

// KafkaConfig specifies connection details for Kafka consumers and producers.
type KafkaConfig struct {
	Brokers []string `yaml:"Brokers"`
	GroupID string   `yaml:"GroupID"`
	Topic   string   `yaml:"Topic"`
}
type JWTConfig struct {
	PrivateKey string `yaml:"PrivateKey"` // Not used on consumer service. Only used by Auth management
	PublicKey  string `yaml:"PublicKey"`
}

type EndpointConfig struct {
	Based string `yaml:"Based"`
}

type TimeoutConfig struct {
	Server int `yaml:"Server"`
	Read   int `yaml:"Read"`
	Write  int `yaml:"Write"`
	Idle   int `yaml:"Idle"`
}

type DatabaseConfig struct {
	Host          string              `yaml:"Host"`
	Port          string              `yaml:"Port"`
	User          string              `yaml:"User"`
	Password      string              `yaml:"Password"`
	DBName        string              `yaml:"DBName"`
	MigrationPath string              `yaml:"MigrationPath"`
	DirtyStrategy enums.DirtyStrategy `yaml:"DirtyStrategy"`
}

type LogConfig struct {
	Path            string            `yaml:"Path"`
	FileName        string            `yaml:"FileName"`
	PerLevelFiles   map[string]string `yaml:"PerLevelFiles"`
	VerboseLevel    string            `yaml:"VerboseLevel"`
	AlsoLogToStdout bool              `yaml:"AlsoLogToStdout"`
}

type OpenAPIConfig struct {
	Version string     `yaml:"Version"`
	Docs    DocsConfig `yaml:"Docs"`
}

type DocsConfig struct {
	Public   string `yaml:"Public"`
	Internal string `yaml:"Internal"`
}

// Cfg holds the loaded application configuration.
var Cfg *Config

// LoadConfig reads YAML configuration from the given path and populates Cfg.
func LoadConfig(filePath string) error {
	file, err := openFile(filePath)
	if err != nil {
		return fmt.Errorf("error opening config file: %w", err)
	}

	decoder := yaml.NewDecoder(file)
	Cfg = &Config{}
	decodeErr := decoder.Decode(Cfg)
	closeErr := file.Close()

	if decodeErr != nil {
		if closeErr != nil {
			return fmt.Errorf("error decoding config file: %v; additionally, error closing file: %w", decodeErr, closeErr)
		}
		return fmt.Errorf("error decoding config file: %w", decodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("error closing config file: %w", closeErr)
	}

	fmt.Println("Configuration loaded")
	return nil
}
