package worker

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/warriorguo/videoslicer/internal/database"
	"github.com/warriorguo/videoslicer/pkg/models"
)

type TaskQueue struct {
	db         *database.DB
	workerID   string
	leaseDuration time.Duration
}

func NewTaskQueue(db *database.DB, workerID string, leaseDuration time.Duration) *TaskQueue {
	return &TaskQueue{
		db:            db,
		workerID:      workerID,
		leaseDuration: leaseDuration,
	}
}

func (tq *TaskQueue) AcquireTask() (*models.VideoTask, error) {
	tx, err := tq.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE video_tasks 
		SET status = 'processing', 
			stage = 'download_source',
			lease_owner = $1, 
			lease_expires_at = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE task_id = (
			SELECT task_id FROM video_tasks 
			WHERE status = 'queued' 
			   OR (status = 'processing' AND lease_expires_at < CURRENT_TIMESTAMP)
			ORDER BY created_at 
			FOR UPDATE SKIP LOCKED 
			LIMIT 1
		)
		RETURNING task_id, status, stage, progress_percent, created_at, updated_at,
				  source_path, source_size, result_path, result_size, manifest_path,
				  segment_sec, frame_interval_sec, frame_format, bg_color, zip_format,
				  error_code, error_message, lease_owner, lease_expires_at, callback_url
	`

	leaseExpiry := time.Now().Add(tq.leaseDuration)
	
	task := &models.VideoTask{}
	var resultPath, manifestPath, errorCode, errorMessage, callbackURL sql.NullString
	var leaseExpiresAt sql.NullTime
	
	err = tx.QueryRow(query, tq.workerID, leaseExpiry).Scan(
		&task.TaskID, &task.Status, &task.Stage, &task.ProgressPercent,
		&task.CreatedAt, &task.UpdatedAt, &task.SourcePath, &task.SourceSize,
		&resultPath, &task.ResultSize, &manifestPath,
		&task.SegmentSec, &task.FrameIntervalSec, &task.FrameFormat, &task.BgColor, &task.ZipFormat,
		&errorCode, &errorMessage, &task.LeaseOwner, &leaseExpiresAt, &callbackURL)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No tasks available
		}
		return nil, fmt.Errorf("failed to acquire task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Set optional fields
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
	if leaseExpiresAt.Valid {
		task.LeaseExpiresAt = &leaseExpiresAt.Time
	}
	if callbackURL.Valid {
		task.CallbackURL = callbackURL.String
	}

	return task, nil
}

func (tq *TaskQueue) UpdateTask(taskID string, status models.TaskStatus, stage models.TaskStage, progress float64) error {
	query := `
		UPDATE video_tasks 
		SET status = $2, stage = $3, progress_percent = $4, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = $1 AND lease_owner = $5
	`
	
	result, err := tq.db.Exec(query, taskID, status, stage, progress, tq.workerID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("task not found or lease expired")
	}
	
	return nil
}

func (tq *TaskQueue) CompleteTask(taskID, resultPath, manifestPath string, resultSize int64) error {
	query := `
		UPDATE video_tasks 
		SET status = 'succeeded', 
			stage = 'completed',
			progress_percent = 100.0,
			result_path = $2,
			manifest_path = $3,
			result_size = $4,
			updated_at = CURRENT_TIMESTAMP,
			lease_owner = NULL,
			lease_expires_at = NULL
		WHERE task_id = $1 AND lease_owner = $5
	`
	
	result, err := tq.db.Exec(query, taskID, resultPath, manifestPath, resultSize, tq.workerID)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("task not found or lease expired")
	}
	
	return nil
}

func (tq *TaskQueue) FailTask(taskID, errorCode, errorMessage string) error {
	query := `
		UPDATE video_tasks 
		SET status = 'failed', 
			stage = 'failed',
			error_code = $2,
			error_message = $3,
			updated_at = CURRENT_TIMESTAMP,
			lease_owner = NULL,
			lease_expires_at = NULL
		WHERE task_id = $1 AND lease_owner = $4
	`
	
	result, err := tq.db.Exec(query, taskID, errorCode, errorMessage, tq.workerID)
	if err != nil {
		return fmt.Errorf("failed to fail task: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("task not found or lease expired")
	}
	
	return nil
}

func (tq *TaskQueue) RenewLease(taskID string) error {
	leaseExpiry := time.Now().Add(tq.leaseDuration)
	
	query := `
		UPDATE video_tasks 
		SET lease_expires_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE task_id = $1 AND lease_owner = $3
	`
	
	result, err := tq.db.Exec(query, taskID, leaseExpiry, tq.workerID)
	if err != nil {
		return fmt.Errorf("failed to renew lease: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("task not found or lease expired")
	}
	
	return nil
}