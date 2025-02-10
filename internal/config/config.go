package config

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"
)

const DefaultLogLevel = slog.LevelInfo

type (
	StorageBundle struct {
		SourceFolder          string `json:"source_folder"`
		ArchiveFolder         string `json:"archive_folder"`
		DestAttachmentsFolder string `json:"dest_attachments_folder"`
		DestNotesFolder       string `json:"dest_notes_folder"`
	}

	// TODO: Update so that each storage config can have settings and add Processor configs
	Config struct {
		TempStorageFolder string          `json:"temp_storage_folder"`
		SourceStore       string          `json:"source_store"`
		Bundles           []StorageBundle `json:"bundles"`
	}
)

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
