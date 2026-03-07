package models

import "time"

type VideoManifest struct {
	TaskID    string            `json:"task_id"`
	Input     VideoInputInfo    `json:"input"`
	Params    ProcessingParams  `json:"params"`
	Clips     []ClipInfo        `json:"clips"`
	Stats     ProcessingStats   `json:"stats"`
	CreatedAt time.Time         `json:"created_at"`
}

type VideoInputInfo struct {
	Filename    string  `json:"filename"`
	DurationSec float64 `json:"duration_sec"`
	SizeBytes   int64   `json:"size_bytes"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	FPS         float64 `json:"fps"`
	Codec       string  `json:"codec"`
}

type ProcessingParams struct {
	SegmentSec       float64 `json:"segment_sec"`
	FrameIntervalSec int    `json:"frame_interval_sec"`
	FrameFormat      string `json:"frame_format"`
	ZipFormat        string `json:"zip_format"`
}

type ClipInfo struct {
	ClipID   string    `json:"clip_id"`
	StartSec float64   `json:"start_sec"`
	EndSec   float64   `json:"end_sec"`
	File     string    `json:"file"`
	Frames   []string  `json:"frames"`
}

type ProcessingStats struct {
	ClipCount  int `json:"clip_count"`
	FrameCount int `json:"frame_count"`
}