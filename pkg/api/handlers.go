package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"github.com/warriorguo/videoslicer/internal/database"
	"github.com/warriorguo/videoslicer/pkg/models"
)

type Handler struct {
	db       *database.DB
	taskDir  string
	maxSize  int64
}

func NewHandler(db *database.DB, taskDir string, maxSize int64) *Handler {
	return &Handler{
		db:      db,
		taskDir: taskDir,
		maxSize: maxSize,
	}
}

type CreateTaskRequest struct {
	SegmentSec       float64 `json:"segment_sec"`
	FrameIntervalSec int    `json:"frame_interval_sec"`
	FrameFormat      string `json:"frame_format"`
	ZipFormat        string `json:"zip_format"`
}

type CreateTaskResponse struct {
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.maxSize); err != nil {
		http.Error(w, "File too large", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !isValidVideoFile(header.Filename) {
		http.Error(w, "Invalid file format", http.StatusBadRequest)
		return
	}

	segmentSec := getFloatParam(r, "segment_sec", 8)
	frameIntervalSec := getIntParam(r, "frame_interval_sec", 2)
	frameFormat := getStringParam(r, "frame_format", "jpg")
	zipFormat := getStringParam(r, "zip_format", "zip")

	log.Printf("CreateTask params: segment_sec=%v, frame_interval_sec=%v, frame_format=%v, zip_format=%v",
		segmentSec, frameIntervalSec, frameFormat, zipFormat)

	taskID := "t_" + xid.New().String()
	
	taskPath := filepath.Join(h.taskDir, taskID)
	if err := os.MkdirAll(taskPath, 0755); err != nil {
		http.Error(w, "Failed to create task directory", http.StatusInternalServerError)
		return
	}

	sourcePath := filepath.Join(taskPath, "source_"+header.Filename)
	dst, err := os.Create(sourcePath)
	if err != nil {
		http.Error(w, "Failed to create source file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	task := &models.VideoTask{
		TaskID:           taskID,
		Status:          models.TaskStatusQueued,
		Stage:           models.TaskStageDownloadSource,
		ProgressPercent: 0.0,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		SourcePath:      sourcePath,
		SourceSize:      size,
		SegmentSec:      segmentSec,
		FrameIntervalSec: frameIntervalSec,
		FrameFormat:     frameFormat,
		ZipFormat:       zipFormat,
	}

	if err := h.insertTask(task); err != nil {
		http.Error(w, "Failed to create task", http.StatusInternalServerError)
		return
	}

	response := CreateTaskResponse{
		TaskID:    taskID,
		Status:    string(models.TaskStatusQueued),
		CreatedAt: task.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]

	task, err := h.getTask(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	response := models.TaskResponse{
		TaskID: task.TaskID,
		Status: task.Status,
		Progress: models.TaskProgress{
			Stage:   task.Stage,
			Percent: task.ProgressPercent,
		},
		Params: models.TaskParams{
			SegmentSec:       task.SegmentSec,
			FrameIntervalSec: task.FrameIntervalSec,
			FrameFormat:      task.FrameFormat,
			ZipFormat:        task.ZipFormat,
		},
		CreatedAt: task.CreatedAt,
	}

	if task.Status == models.TaskStatusSucceeded && task.ResultPath != "" {
		response.Result = &models.TaskResult{
			ZipSizeBytes: task.ResultSize,
		}
	}

	if task.Status == models.TaskStatusFailed {
		response.Error = &models.TaskError{
			Code:    task.ErrorCode,
			Message: task.ErrorMessage,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) DownloadResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]

	task, err := h.getTask(taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get task", http.StatusInternalServerError)
		return
	}

	if task.Status != models.TaskStatusSucceeded || task.ResultPath == "" {
		http.Error(w, "Result not available", http.StatusNotFound)
		return
	}

	file, err := os.Open(task.ResultPath)
	if err != nil {
		http.Error(w, "Result file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to get file info", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%s.%s", taskID, task.ZipFormat)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))

	io.Copy(w, file)
}

func (h *Handler) insertTask(task *models.VideoTask) error {
	query := `
		INSERT INTO video_tasks (
			task_id, status, stage, progress_percent, created_at, updated_at,
			source_path, source_size, segment_sec, frame_interval_sec, 
			frame_format, zip_format
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := h.db.Exec(query,
		task.TaskID, task.Status, task.Stage, task.ProgressPercent,
		task.CreatedAt, task.UpdatedAt, task.SourcePath, task.SourceSize,
		task.SegmentSec, task.FrameIntervalSec, task.FrameFormat, task.ZipFormat)
	return err
}

func (h *Handler) getTask(taskID string) (*models.VideoTask, error) {
	query := `
		SELECT task_id, status, stage, progress_percent, created_at, updated_at,
			   source_path, source_size, result_path, result_size, manifest_path,
			   segment_sec, frame_interval_sec, frame_format, zip_format,
			   error_code, error_message, lease_owner, lease_expires_at, callback_url
		FROM video_tasks WHERE task_id = $1
	`
	
	task := &models.VideoTask{}
	var resultPath, manifestPath, errorCode, errorMessage, leaseOwner, callbackURL sql.NullString
	var leaseExpiresAt sql.NullTime
	
	err := h.db.QueryRow(query, taskID).Scan(
		&task.TaskID, &task.Status, &task.Stage, &task.ProgressPercent,
		&task.CreatedAt, &task.UpdatedAt, &task.SourcePath, &task.SourceSize,
		&resultPath, &task.ResultSize, &manifestPath,
		&task.SegmentSec, &task.FrameIntervalSec, &task.FrameFormat, &task.ZipFormat,
		&errorCode, &errorMessage, &leaseOwner, &leaseExpiresAt, &callbackURL)
	
	if err != nil {
		return nil, err
	}
	
	if resultPath.Valid {
		task.ResultPath = resultPath.String
	}
	if manifestPath.Valid {
		task.ManifestPath = manifestPath.String
	}
	if errorCode.Valid {
		task.ErrorCode = errorCode.String
	}
	if errorMessage.Valid {
		task.ErrorMessage = errorMessage.String
	}
	if leaseOwner.Valid {
		task.LeaseOwner = leaseOwner.String
	}
	if leaseExpiresAt.Valid {
		task.LeaseExpiresAt = &leaseExpiresAt.Time
	}
	if callbackURL.Valid {
		task.CallbackURL = callbackURL.String
	}
	
	return task, nil
}

func isValidVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv"}
	
	for _, validExt := range validExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

func getIntParam(r *http.Request, key string, defaultVal int) int {
	if val := r.FormValue(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getFloatParam(r *http.Request, key string, defaultVal float64) float64 {
	if val := r.FormValue(key); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getStringParam(r *http.Request, key, defaultVal string) string {
	if val := r.FormValue(key); val != "" {
		return val
	}
	return defaultVal
}