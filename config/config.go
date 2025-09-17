package config

import (
	"errors"
	"os"
	"path"
	"time"

	yaml "github.com/goccy/go-yaml"

	"video-stream/log"
)

type Config struct {
	LogLevel        string              `yaml:"logLevel"`
	Channels        map[string][]string `yaml:"channels"`
	ScheduleHorizon time.Duration       `yaml:"scheduleHorizon"`
}

var Current Config

func init() {
	cfg, err := readConfigFile()
	if err != nil {
		log.Fatal("[config::init] Could not read config:", "msg", err.Error())
	}

	Current = cfg
}

func getConfigFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	return path.Join(cwd, "config.yaml"), nil
}

func readConfigFile() (Config, error) {
	cfg := Config{}

	cfgFile, err := getConfigFilePath()
	if err != nil {
		return cfg, err
	}

	// Stat the file
	if _, err := os.Stat(cfgFile); errors.Is(err, os.ErrNotExist) {
		log.Warnf("File '%s' does not exist, cannot read config!", cfgFile)
		return cfg, err
		// file doesn't exist
	}

	yml, err := os.ReadFile(cfgFile)

	if err = yaml.Unmarshal([]byte(yml), &cfg); err != nil {
		log.Warn("could not unmarshal yaml", "msg", err.Error())
		return cfg, err
	}

	// Ensure defaults if properties are unset
	if cfg.ScheduleHorizon == 0 {
		cfg.ScheduleHorizon = time.Duration(2 * time.Hour)
	}

	return cfg, nil
}

func Write(cfg Config) error {
	bytes, err := yaml.Marshal(cfg)
	if err != nil {
		log.Warn("could not marshal yaml", "msg", err.Error())
		return err
	}

	cfgFile, err := getConfigFilePath()
	if err != nil {
		log.Warn("couldn't get config file path")
		return err
	}

	err = os.WriteFile(cfgFile, bytes, 0644)
	if err != nil {
		log.Warn("could not write config file", "msg", err.Error())
		return err
	}
	return nil
}

func (c Config) String() (string, error) {
	bytes, err := yaml.Marshal(c)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
