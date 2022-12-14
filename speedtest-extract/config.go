package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	ApiKey               string `yaml:"api_key"`
	ApiSecret            string `yaml:"api_secret"`
	ExtractUrl           string `yaml:"extract_url"`
	StorageDirectory     string `yaml:"storage_directory"`
	CacheFilename        string `yaml:"cache_filename"`
	CacheDurationMinutes int    `yaml:"cache_duration_minutes"`
}

var DefaultConfig = Config{
	ApiKey:               "my-api-key",
	ApiSecret:            "my-api-secret",
	ExtractUrl:           "https://intelligence.speedtest.net/extracts",
	StorageDirectory:     ".",
	CacheFilename:        ".extracts-cache.json",
	CacheDurationMinutes: -1,
}

var DefaultConfigFile = "speedtest-extract.yaml"

func ReadConfig(configFile string) (*Config, error) {
	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(contents, &config)
	if err != nil {
		return nil, err
	}

	if len(config.ApiKey) == 0 || len(config.ApiSecret) == 0 {
		return nil, ErrMissingAuth
	}
	if config.ApiKey == DefaultConfig.ApiKey || config.ApiSecret == DefaultConfig.ApiSecret {
		return nil, ErrDefaultConfig
	}
	if len(config.ExtractUrl) == 0 {
		config.ExtractUrl = DefaultConfig.ExtractUrl
	}
	if len(config.StorageDirectory) == 0 || config.StorageDirectory == "" {
		config.StorageDirectory = DefaultConfig.StorageDirectory
	}
	if len(config.CacheFilename) == 0 {
		config.CacheFilename = DefaultConfig.CacheFilename
	}
	if config.CacheDurationMinutes == 0 {
		config.CacheDurationMinutes = DefaultConfig.CacheDurationMinutes
	}

	return &config, nil
}

func WriteConfig() error {
	configYaml, err := yaml.Marshal(DefaultConfig)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(DefaultConfigFile, configYaml, 0600)
	if err != nil {
		return err
	}
	return nil
}
