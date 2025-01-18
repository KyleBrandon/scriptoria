package config

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
)

const DefaultLogLevel = slog.LevelInfo

type Config struct {
	OriginPatterns []string `json:"origin_patterns"`
}

func LoadConfigSettings(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}

	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
