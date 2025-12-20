package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/warriorguo/videoslicer/pkg/ffmpeg"
	"github.com/warriorguo/videoslicer/pkg/models"
	"github.com/warriorguo/videoslicer/pkg/storage"
)

func TestFFmpegBasicFunctionality(t *testing.T) {
	// Test if FFmpeg tools are available
	if !ffmpeg.IsFFmpegAvailable() {
		t.Skip("FFmpeg not available, skipping functional test")
	}
	if !ffmpeg.IsFFprobeAvailable() {
		t.Skip("FFprobe not available, skipping functional test")
	}
	
	processor := ffmpeg.NewProcessor()
	
	// Test processor creation
	if processor == nil {
		t.Fatal("Failed to create FFmpeg processor")
	}
	
	t.Log("FFmpeg processor created successfully")
	t.Logf("Default timeout: %v", processor.GetTimeout())
}

func TestStorageOperations(t *testing.T) {
	tempDir := t.TempDir()
	storage := storage.NewLocal(tempDir)
	
	// Test task directory creation
	taskID := "test_task_123"
	taskPath, err := storage.EnsureTaskDir(taskID)
	if err != nil {
		t.Fatalf("Failed to create task directory: %v", err)
	}
	
	// Verify directory exists
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		t.Errorf("Task directory was not created: %s", taskPath)
	}
	
	// Test manifest creation
	manifest := createTestManifest(taskID)
	manifestPath := filepath.Join(taskPath, "manifest.json")
	
	err = storage.SaveManifest(manifestPath, manifest)
	if err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}
	
	// Verify manifest file exists
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("Manifest file was not created: %s", manifestPath)
	}
	
	// Test directory structure
	expectedPaths := []string{
		filepath.Join(taskPath, "clips"),
		filepath.Join(taskPath, "frames"),
		filepath.Join(taskPath, "frames", "c000"),
		filepath.Join(taskPath, "frames", "c001"),
	}
	
	for _, path := range expectedPaths {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Errorf("Failed to create directory %s: %v", path, err)
		}
	}
	
	// Create some test files
	testFiles := []string{
		filepath.Join(taskPath, "clips", "c000.mp4"),
		filepath.Join(taskPath, "clips", "c001.mp4"),
		filepath.Join(taskPath, "frames", "c000", "f0001.jpg"),
		filepath.Join(taskPath, "frames", "c000", "f0002.jpg"),
		filepath.Join(taskPath, "frames", "c001", "f0001.jpg"),
		filepath.Join(taskPath, "frames", "c001", "f0002.jpg"),
	}
	
	for _, file := range testFiles {
		if err := os.WriteFile(file, []byte("test content"), 0644); err != nil {
			t.Errorf("Failed to create test file %s: %v", file, err)
		}
	}
	
	// Test packaging
	resultPath := filepath.Join(taskPath, "result.zip")
	err = storage.PackageResult(taskPath, resultPath, "zip")
	if err != nil {
		t.Fatalf("Failed to package result: %v", err)
	}
	
	// Verify result file exists and has content
	stat, err := os.Stat(resultPath)
	if err != nil {
		t.Errorf("Result file was not created: %s", resultPath)
	} else if stat.Size() == 0 {
		t.Errorf("Result file is empty")
	}
	
	t.Logf("Successfully created result package: %s (size: %d bytes)", resultPath, stat.Size())
	
	// Test tar.gz packaging
	resultTarPath := filepath.Join(taskPath, "result.tar.gz")
	err = storage.PackageResult(taskPath, resultTarPath, "tar.gz")
	if err != nil {
		t.Fatalf("Failed to package result as tar.gz: %v", err)
	}
	
	// Verify tar.gz file exists
	stat, err = os.Stat(resultTarPath)
	if err != nil {
		t.Errorf("Tar.gz file was not created: %s", resultTarPath)
	} else if stat.Size() == 0 {
		t.Errorf("Tar.gz file is empty")
	}
	
	t.Logf("Successfully created tar.gz package: %s (size: %d bytes)", resultTarPath, stat.Size())
}

func TestWorkflowSimulation(t *testing.T) {
	// Simulate a complete workflow without actual video processing
	tempDir := t.TempDir()
	storage := storage.NewLocal(tempDir)
	taskID := "workflow_test_456"
	
	// Step 1: Create task directory
	taskPath, err := storage.EnsureTaskDir(taskID)
	if err != nil {
		t.Fatalf("Step 1 failed - create task directory: %v", err)
	}
	t.Logf("Step 1: Created task directory: %s", taskPath)
	
	// Step 2: Simulate video upload (create source file)
	sourcePath := filepath.Join(taskPath, "source_test_video.mp4")
	sourceContent := make([]byte, 1024*1024) // 1MB mock video
	for i := range sourceContent {
		sourceContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(sourcePath, sourceContent, 0644); err != nil {
		t.Fatalf("Step 2 failed - create source file: %v", err)
	}
	t.Logf("Step 2: Created source file: %s", sourcePath)
	
	// Step 3: Simulate video analysis
	videoInfo := &ffmpeg.VideoInfo{
		Duration: 30.0,
		Width:    1920,
		Height:   1080,
		FPS:      30.0,
		Codec:    "h264",
	}
	t.Logf("Step 3: Analyzed video: %dx%d, %.1fs, %.1ffps", videoInfo.Width, videoInfo.Height, videoInfo.Duration, videoInfo.FPS)
	
	// Step 4: Simulate video slicing (create clip directories and files)
	clipsDir := filepath.Join(taskPath, "clips")
	if err := os.MkdirAll(clipsDir, 0755); err != nil {
		t.Fatalf("Step 4 failed - create clips directory: %v", err)
	}
	
	segmentSec := 8
	numClips := int(videoInfo.Duration) / segmentSec
	if int(videoInfo.Duration)%segmentSec > 0 {
		numClips++
	}
	
	var clips []models.ClipInfo
	for i := 0; i < numClips; i++ {
		clipID := fmt.Sprintf("c%03d", i)
		clipFile := filepath.Join(clipsDir, clipID+".mp4")
		
		// Create mock clip file
		clipContent := sourceContent[i*1000 : (i+1)*1000]
		if err := os.WriteFile(clipFile, clipContent, 0644); err != nil {
			t.Errorf("Failed to create clip file %s: %v", clipFile, err)
			continue
		}
		
		clips = append(clips, models.ClipInfo{
			ClipID:   clipID,
			StartSec: float64(i * segmentSec),
			EndSec:   float64((i + 1) * segmentSec),
			File:     fmt.Sprintf("clips/%s.mp4", clipID),
			Frames:   []string{}, // Will be filled in next step
		})
	}
	t.Logf("Step 4: Created %d video clips", len(clips))
	
	// Step 5: Simulate frame extraction
	framesDir := filepath.Join(taskPath, "frames")
	totalFrames := 0
	
	for i := range clips {
		clipFramesDir := filepath.Join(framesDir, clips[i].ClipID)
		if err := os.MkdirAll(clipFramesDir, 0755); err != nil {
			t.Errorf("Failed to create frames directory for clip %s: %v", clips[i].ClipID, err)
			continue
		}
		
		// Simulate extracting frames every 2 seconds
		frameIntervalSec := 2
		numFrames := segmentSec / frameIntervalSec
		
		var frameFiles []string
		for j := 0; j < numFrames; j++ {
			frameFile := fmt.Sprintf("f%04d.jpg", j+1)
			framePath := filepath.Join(clipFramesDir, frameFile)
			
			// Create mock frame file (simple image data)
			frameContent := []byte(fmt.Sprintf("mock-frame-data-clip-%s-frame-%d", clips[i].ClipID, j))
			if err := os.WriteFile(framePath, frameContent, 0644); err != nil {
				t.Errorf("Failed to create frame file %s: %v", framePath, err)
				continue
			}
			
			frameFiles = append(frameFiles, fmt.Sprintf("frames/%s/%s", clips[i].ClipID, frameFile))
			totalFrames++
		}
		
		clips[i].Frames = frameFiles
	}
	t.Logf("Step 5: Extracted %d frames from %d clips", totalFrames, len(clips))
	
	// Step 6: Create manifest
	manifest := &models.VideoManifest{
		TaskID: taskID,
		Input: models.VideoInputInfo{
			Filename:    "test_video.mp4",
			DurationSec: videoInfo.Duration,
			SizeBytes:   int64(len(sourceContent)),
			Width:       videoInfo.Width,
			Height:      videoInfo.Height,
			FPS:         videoInfo.FPS,
			Codec:       videoInfo.Codec,
		},
		Params: models.ProcessingParams{
			SegmentSec:       segmentSec,
			FrameIntervalSec: 2,
			FrameFormat:      "jpg",
			ZipFormat:        "zip",
		},
		Clips: clips,
		Stats: models.ProcessingStats{
			ClipCount:  len(clips),
			FrameCount: totalFrames,
		},
		CreatedAt: time.Now(),
	}
	
	manifestPath := filepath.Join(taskPath, "manifest.json")
	if err := storage.SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Step 6 failed - save manifest: %v", err)
	}
	t.Logf("Step 6: Created manifest with %d clips and %d frames", len(clips), totalFrames)
	
	// Step 7: Package results
	resultPath := filepath.Join(taskPath, "result.zip")
	if err := storage.PackageResult(taskPath, resultPath, "zip"); err != nil {
		t.Fatalf("Step 7 failed - package results: %v", err)
	}
	
	stat, err := os.Stat(resultPath)
	if err != nil {
		t.Fatalf("Result file was not created: %v", err)
	}
	
	t.Logf("Step 7: Packaged results into %s (size: %d bytes)", resultPath, stat.Size())
	t.Logf("Workflow simulation completed successfully!")
}

func createTestManifest(taskID string) *models.VideoManifest {
	return &models.VideoManifest{
		TaskID: taskID,
		Input: models.VideoInputInfo{
			Filename:    "test.mp4",
			DurationSec: 16.0,
			SizeBytes:   1000000,
			Width:       1280,
			Height:      720,
			FPS:         25.0,
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
				Frames:   []string{"frames/c000/f0001.jpg", "frames/c000/f0002.jpg", "frames/c000/f0003.jpg", "frames/c000/f0004.jpg"},
			},
			{
				ClipID:   "c001",
				StartSec: 8,
				EndSec:   16,
				File:     "clips/c001.mp4",
				Frames:   []string{"frames/c001/f0001.jpg", "frames/c001/f0002.jpg", "frames/c001/f0003.jpg", "frames/c001/f0004.jpg"},
			},
		},
		Stats: models.ProcessingStats{
			ClipCount:  2,
			FrameCount: 8,
		},
		CreatedAt: time.Now(),
	}
}