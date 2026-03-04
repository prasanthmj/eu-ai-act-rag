package ingestion

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	rawDir       = "ingestion/data/raw"
	processedDir = "ingestion/data/processed"
)

// EnsureDataDirs creates the raw and processed data directories.
func EnsureDataDirs() error {
	for _, dir := range []string{rawDir, processedDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	return nil
}

// SaveRawJSON writes data as JSON to the raw data directory.
func SaveRawJSON(filename string, data any) error {
	return saveJSON(filepath.Join(rawDir, filename), data)
}

// SaveProcessedJSON writes data as JSON to the processed data directory.
func SaveProcessedJSON(filename string, data any) error {
	return saveJSON(filepath.Join(processedDir, filename), data)
}

func saveJSON(path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	return nil
}
