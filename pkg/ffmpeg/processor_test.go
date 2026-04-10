package ffmpeg

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Fatal("Expected processor to be created, got nil")
	}
	if processor.timeout != 30*time.Minute {
		t.Errorf("Expected default timeout 30m, got %v", processor.timeout)
	}
}

func TestSetTimeout(t *testing.T) {
	processor := NewProcessor()
	newTimeout := 10 * time.Minute
	processor.SetTimeout(newTimeout)
	
	if processor.timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, processor.timeout)
	}
}

func TestIsFFmpegAvailable(t *testing.T) {
	available := IsFFmpegAvailable()
	t.Logf("FFmpeg available: %v", available)
	
	// This test will pass regardless of FFmpeg availability
	// but logs the result for debugging
}

func TestIsFFprobeAvailable(t *testing.T) {
	available := IsFFprobeAvailable()
	t.Logf("FFprobe available: %v", available)
	
	// This test will pass regardless of FFprobe availability
	// but logs the result for debugging
}

// Note: The following tests require FFmpeg/FFprobe to be installed
// They will be skipped if the tools are not available

func TestProbe_NonExistentFile(t *testing.T) {
	if !IsFFprobeAvailable() {
		t.Skip("FFprobe not available, skipping probe test")
	}
	
	processor := NewProcessor()
	_, err := processor.Probe("/nonexistent/video.mp4")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestSliceVideo_NonExistentFile(t *testing.T) {
	if !IsFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping slice test")
	}
	
	processor := NewProcessor()
	
	// Create a temporary directory for output
	tempDir := t.TempDir()
	
	_, err := processor.SliceVideo("/nonexistent/video.mp4", tempDir, 8)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestExtractFrames_NonExistentFile(t *testing.T) {
	if !IsFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping frame extraction test")
	}
	
	processor := NewProcessor()
	
	// Create a temporary directory for output
	tempDir := t.TempDir()
	
	_, err := processor.ExtractFrames("/nonexistent/video.mp4", tempDir, 2.0, "jpg", "black")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestExtractFramesFixedCount_NonExistentFile(t *testing.T) {
	if !IsFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping frame extraction test")
	}
	
	processor := NewProcessor()
	
	// Create a temporary directory for output
	tempDir := t.TempDir()
	
	_, err := processor.ExtractFramesFixedCount("/nonexistent/video.mp4", tempDir, 3, "jpg", "black")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// Helper function to create a test video file (would need actual video data)
func createTestVideoFile(t *testing.T, dir string) string {
	// This is a placeholder - in real tests you'd create or copy a test video file
	testVideoPath := filepath.Join(dir, "test_video.mp4")
	
	// For testing purposes, just create an empty file
	// Real tests would need actual video content
	file, err := os.Create(testVideoPath)
	if err != nil {
		t.Fatalf("Failed to create test video file: %v", err)
	}
	file.Close()
	
	return testVideoPath
}