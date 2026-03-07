package models

import (
	"time"
)

type TaskStatus string

const (
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusProcessing TaskStatus = "processing"
	TaskStatusSucceeded  TaskStatus = "succeeded"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusExpired    TaskStatus = "expired"
)

type TaskStage string

const (
	TaskStageDownloadSource   TaskStage = "download_source"
	TaskStageProbe           TaskStage = "probe"
	TaskStageSlicing         TaskStage = "slicing"
	TaskStageExtractingFrames TaskStage = "extracting_frames"
	TaskStageManifest        TaskStage = "manifest"
	TaskStagePackaging       TaskStage = "packaging"
	TaskStageUploading       TaskStage = "uploading"
)

type VideoTask struct {
	TaskID      string    `db:"task_id" json:"task_id"`
	Status      TaskStatus `db:"status" json:"status"`
	Stage       TaskStage `db:"stage" json:"stage"`
	ProgressPercent float64 `db:"progress_percent" json:"progress_percent"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	
	// Source file info
	SourcePath string `db:"source_path" json:"source_path"`
	SourceSize int64  `db:"source_size" json:"source_size"`
	
	// Result info
	ResultPath string `db:"result_path" json:"result_path"`
	ResultSize int64  `db:"result_size" json:"result_size"`
	ManifestPath string `db:"manifest_path" json:"manifest_path"`
	
	// Processing parameters
	SegmentSec       float64 `db:"segment_sec" json:"segment_sec"`
	FrameIntervalSec int    `db:"frame_interval_sec" json:"frame_interval_sec"`
	FrameFormat     string `db:"frame_format" json:"frame_format"`
	ZipFormat       string `db:"zip_format" json:"zip_format"`
	
	// Error info
	ErrorCode    string `db:"error_code" json:"error_code"`
	ErrorMessage string `db:"error_message" json:"error_message"`
	
	// Worker lease
	LeaseOwner    string     `db:"lease_owner" json:"lease_owner"`
	LeaseExpiresAt *time.Time `db:"lease_expires_at" json:"lease_expires_at"`
	
	// Optional callback
	CallbackURL string `db:"callback_url" json:"callback_url"`
}

type TaskProgress struct {
	Stage   TaskStage `json:"stage"`
	Percent float64   `json:"percent"`
}

type TaskParams struct {
	SegmentSec       float64 `json:"segment_sec"`
	FrameIntervalSec int    `json:"frame_interval_sec"`
	FrameFormat     string `json:"frame_format"`
	ZipFormat       string `json:"zip_format"`
}

type TaskResult struct {
	ZipURL      string `json:"zip_url,omitempty"`
	ZipSizeBytes int64  `json:"zip_size_bytes"`
	ManifestURL string `json:"manifest_url,omitempty"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TaskResponse struct {
	TaskID    string        `json:"task_id"`
	Status    TaskStatus    `json:"status"`
	Progress  TaskProgress  `json:"progress"`
	Params    TaskParams    `json:"params"`
	Result    *TaskResult   `json:"result,omitempty"`
	Error     *TaskError    `json:"error,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}