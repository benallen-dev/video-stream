package config

import (
	"os"
	"path"

	yaml "github.com/goccy/go-yaml"

	"video-stream/log"
)

type Config struct {
	LogLevel string
	Channels map[string][]string
}

func getConfigFilePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	return path.Join(cwd, "config.yaml"), nil
}

func Read() (Config, error) {
	cfgFile, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	
	yml, err := os.ReadFile(cfgFile)

	cfg := Config{}

	if err = yaml.Unmarshal([]byte(yml), &cfg); err != nil {
		log.Warn("could not unmarshal yaml", "msg", err.Error())
		return cfg, err
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

