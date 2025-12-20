package models

import (
	"testing"
	"time"
)

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
	}{
		{TaskStatusQueued, "queued"},
		{TaskStatusProcessing, "processing"},
		{TaskStatusSucceeded, "succeeded"},
		{TaskStatusFailed, "failed"},
		{TaskStatusExpired, "expired"},
	}

	for _, test := range tests {
		if string(test.status) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.status))
		}
	}
}

func TestTaskStage(t *testing.T) {
	tests := []struct {
		stage    TaskStage
		expected string
	}{
		{TaskStageDownloadSource, "download_source"},
		{TaskStageProbe, "probe"},
		{TaskStageSlicing, "slicing"},
		{TaskStageExtractingFrames, "extracting_frames"},
		{TaskStageManifest, "manifest"},
		{TaskStagePackaging, "packaging"},
		{TaskStageUploading, "uploading"},
	}

	for _, test := range tests {
		if string(test.stage) != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, string(test.stage))
		}
	}
}

func TestVideoTaskDefaults(t *testing.T) {
	task := VideoTask{
		TaskID:           "test_task",
		Status:           TaskStatusQueued,
		Stage:            TaskStageDownloadSource,
		SegmentSec:       8,
		FrameIntervalSec: 2,
		FrameFormat:      "jpg",
		ZipFormat:        "zip",
		CreatedAt:        time.Now(),
	}

	if task.TaskID != "test_task" {
		t.Errorf("Expected task ID 'test_task', got %s", task.TaskID)
	}
	if task.Status != TaskStatusQueued {
		t.Errorf("Expected status 'queued', got %s", task.Status)
	}
	if task.SegmentSec != 8 {
		t.Errorf("Expected segment_sec 8, got %d", task.SegmentSec)
	}
	if task.FrameIntervalSec != 2 {
		t.Errorf("Expected frame_interval_sec 2, got %d", task.FrameIntervalSec)
	}
}