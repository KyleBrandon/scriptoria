package config

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
)

const DefaultLogLevel = slog.LevelInfo

// TODO: Update so that each storage config can have settings and add Processor configs
type Config struct {
	SourceStore    string   `json:"source_store"`
	DestStore      string   `json:"dest_store"`
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
