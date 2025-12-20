package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warriorguo/videoslicer/pkg/models"
)

func TestNewLocal(t *testing.T) {
	baseDir := "/tmp/test"
	storage := NewLocal(baseDir)
	
	if storage == nil {
		t.Fatal("Expected storage to be created, got nil")
	}
	if storage.baseDir != baseDir {
		t.Errorf("Expected baseDir %s, got %s", baseDir, storage.baseDir)
	}
}

func TestGetTaskPath(t *testing.T) {
	baseDir := "/tmp/test"
	storage := NewLocal(baseDir)
	taskID := "test_task_123"
	
	expected := filepath.Join(baseDir, taskID)
	actual := storage.GetTaskPath(taskID)
	
	if actual != expected {
		t.Errorf("Expected task path %s, got %s", expected, actual)
	}
}

func TestEnsureTaskDir(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	taskID := "test_task_123"
	
	taskPath, err := storage.EnsureTaskDir(taskID)
	if err != nil {
		t.Fatalf("Failed to ensure task directory: %v", err)
	}
	
	expected := filepath.Join(tempDir, taskID)
	if taskPath != expected {
		t.Errorf("Expected task path %s, got %s", expected, taskPath)
	}
	
	// Check if directory actually exists
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		t.Errorf("Task directory was not created: %s", taskPath)
	}
}

func TestSaveManifest(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	
	manifest := &models.VideoManifest{
		TaskID: "test_task",
		Input: models.VideoInputInfo{
			Filename:    "test.mp4",
			DurationSec: 60.0,
			SizeBytes:   1000000,
			Width:       1920,
			Height:      1080,
			FPS:         30.0,
			Codec:       "h264",
		},
		Params: models.ProcessingParams{
			SegmentSec:       8,
			FrameIntervalSec: 2,
			FrameFormat:      "jpg",
			ZipFormat:        "zip",
		},
		Clips: []models.ClipInfo{
			{
				ClipID:   "c000",
				StartSec: 0,
				EndSec:   8,
				File:     "clips/c000.mp4",
				Frames:   []string{"frames/c000/f0001.jpg", "frames/c000/f0002.jpg"},
			},
		},
		Stats: models.ProcessingStats{
			ClipCount:  1,
			FrameCount: 2,
		},
		CreatedAt: time.Now(),
	}
	
	manifestPath := filepath.Join(tempDir, "manifest.json")
	err := storage.SaveManifest(manifestPath, manifest)
	if err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}
	
	// Verify the file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created: %s", manifestPath)
	}
	
	// Verify the content
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest file: %v", err)
	}
	
	var loadedManifest models.VideoManifest
	if err := json.Unmarshal(data, &loadedManifest); err != nil {
		t.Fatalf("Failed to parse manifest JSON: %v", err)
	}
	
	if loadedManifest.TaskID != manifest.TaskID {
		t.Errorf("Expected TaskID %s, got %s", manifest.TaskID, loadedManifest.TaskID)
	}
}

func TestPackageResult_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	
	err := storage.PackageResult(tempDir, "result.rar", "rar")
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
}

func TestPackageResult_Zip(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	
	// Create some test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "subdir", "test2.txt")
	
	os.MkdirAll(filepath.Dir(testFile2), 0755)
	
	os.WriteFile(testFile1, []byte("test content 1"), 0644)
	os.WriteFile(testFile2, []byte("test content 2"), 0644)
	
	resultPath := filepath.Join(tempDir, "result.zip")
	err := storage.PackageResult(tempDir, resultPath, "zip")
	if err != nil {
		t.Fatalf("Failed to create zip package: %v", err)
	}
	
	// Verify zip file exists
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Errorf("Zip file was not created: %s", resultPath)
	}
}

func TestPackageResult_TarGz(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	
	// Create some test files
	testFile1 := filepath.Join(tempDir, "test1.txt")
	testFile2 := filepath.Join(tempDir, "subdir", "test2.txt")
	
	os.MkdirAll(filepath.Dir(testFile2), 0755)
	
	os.WriteFile(testFile1, []byte("test content 1"), 0644)
	os.WriteFile(testFile2, []byte("test content 2"), 0644)
	
	resultPath := filepath.Join(tempDir, "result.tar.gz")
	err := storage.PackageResult(tempDir, resultPath, "tar.gz")
	if err != nil {
		t.Fatalf("Failed to create tar.gz package: %v", err)
	}
	
	// Verify tar.gz file exists
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Errorf("Tar.gz file was not created: %s", resultPath)
	}
}

func TestDeleteTask(t *testing.T) {
	tempDir := t.TempDir()
	storage := NewLocal(tempDir)
	
	// Create a task directory with some files
	taskDir := filepath.Join(tempDir, "test_task")
	os.MkdirAll(taskDir, 0755)
	
	testFile := filepath.Join(taskDir, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)
	
	// Verify directory exists
	if _, err := os.Stat(taskDir); os.IsNotExist(err) {
		t.Fatalf("Task directory should exist before deletion")
	}
	
	// Delete task
	err := storage.DeleteTask(taskDir)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
	
	// Verify directory is deleted
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Errorf("Task directory should be deleted")
	}
}