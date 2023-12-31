package usenet

import (
	"testing"
	"time"
)

func TestLoadMetadataFromMap(t *testing.T) {
	// Test case 1: Valid nzb file
	t.Run("Valid nzb file", func(t *testing.T) {
		input := map[string]string{
			"file_name":      "test_file",
			"file_size":      "100",
			"mod_time":       "2006-01-02 15:04:05",
			"file_extension": "txt",
			"chunk_size":     "10",
			"subject":        "test_file [10/10] size=10",
		}
		expectedMetadata := &Metadata{
			FileName:      "test_file",
			FileExtension: "txt",
			FileSize:      100,
			ChunkSize:     10,
			ModTime:       time.Date(2006, 1, 2, 15, 04, 05, 0, time.UTC),
		}
		metadata, err := LoadMetadataFromMap(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if metadata != *expectedMetadata {
			t.Errorf("unexpected metadata: got %v, want %v", metadata, expectedMetadata)
		}
	})

	// Test case 2: Missing required metadata
	t.Run("Missing required metadata", func(t *testing.T) {
		input := map[string]string{
			"file_extension": "txt",
			"chunk_size":     "10",
		}
		_, err := LoadMetadataFromMap(input)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}
	})

	// Test case 3: Invalid file size
	t.Run("Invalid file size", func(t *testing.T) {
		input := map[string]string{
			"file_name":      "test_file",
			"file_size":      "invalid",
			"mod_time":       "2006-01-02 15:04:05",
			"file_extension": "txt",
			"chunk_size":     "10",
			"subject":        "test_file [10/10] size=10",
		}
		_, err := LoadMetadataFromMap(input)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}
	})

	// Test case 5: Invalid chunk size
	t.Run("Invalid chunk size", func(t *testing.T) {
		input := map[string]string{
			"file_name":      "test_file",
			"file_size":      "100",
			"mod_time":       "2006-01-02 15:04:05",
			"file_extension": "txt",
			"chunk_size":     "invalid",
			"subject":        "test_file [10/10] size=10",
		}
		_, err := LoadMetadataFromMap(input)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}
	})

	// Test case 6: Missing chunk size, fallback to old subject format
	t.Run("Missing chunk size, fallback to old subject format", func(t *testing.T) {
		input := map[string]string{
			"file_name":      "test_file",
			"file_size":      "100",
			"mod_time":       "2006-01-02 15:04:05",
			"file_extension": "txt",
			"subject":        "test_file [10/10] size=10",
		}
		expectedMetadata := &Metadata{
			FileName:      "test_file",
			FileExtension: "txt",
			FileSize:      100,
			ChunkSize:     10,
			ModTime:       time.Date(2006, 1, 2, 15, 04, 05, 0, time.UTC),
		}
		metadata, err := LoadMetadataFromMap(input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if metadata != *expectedMetadata {
			t.Errorf("unexpected metadata: got %v, want %v", metadata, expectedMetadata)
		}
	})

	// Test case 7: Invalid mod time
	t.Run("Invalid mod time", func(t *testing.T) {
		input := map[string]string{
			"file_name":      "test_file",
			"file_size":      "100",
			"mod_time":       "invalid",
			"file_extension": "txt",
			"chunk_size":     "10",
			"subject":        "test_file [10/10] size=10",
		}
		_, err := LoadMetadataFromMap(input)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}
	})

	// Test case 8: Missing file extension
	t.Run("Missing file extension", func(t *testing.T) {
		input := map[string]string{
			"file_name":  "test_file",
			"file_size":  "100",
			"mod_time":   "2006-01-02 15:04:05",
			"chunk_size": "10",
			"subject":    "test_file [10/10] size=10",
		}
		_, err := LoadMetadataFromMap(input)
		if err == nil {
			t.Errorf("expected error, but got nil")
		}
	})
}
