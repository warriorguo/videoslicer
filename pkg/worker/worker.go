package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/xid"
	"github.com/warriorguo/videoslicer/internal/database"
	"github.com/warriorguo/videoslicer/pkg/ffmpeg"
	"github.com/warriorguo/videoslicer/pkg/models"
	"github.com/warriorguo/videoslicer/pkg/storage"
)

type Worker struct {
	id             string
	queue          *TaskQueue
	processor      *ffmpeg.Processor
	storage        *storage.Local
	maxConcurrent  int
	pollInterval   time.Duration
	leaseRenewal   time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

type Config struct {
	MaxConcurrent int
	PollInterval  time.Duration
	LeaseDuration time.Duration
	TaskDir       string
}

func NewWorker(db *database.DB, config Config) *Worker {
	workerID := "worker_" + xid.New().String()
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Worker{
		id:            workerID,
		queue:         NewTaskQueue(db, workerID, config.LeaseDuration),
		processor:     ffmpeg.NewProcessor(),
		storage:       storage.NewLocal(config.TaskDir),
		maxConcurrent: config.MaxConcurrent,
		pollInterval:  config.PollInterval,
		leaseRenewal:  config.LeaseDuration / 3, // Renew lease at 1/3 of duration
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (w *Worker) Start() {
	log.Printf("Worker %s starting with %d max concurrent tasks", w.id, w.maxConcurrent)
	
	// Channel to limit concurrent tasks
	semaphore := make(chan struct{}, w.maxConcurrent)
	
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("Worker %s stopping", w.id)
			return
		case <-ticker.C:
			select {
			case semaphore <- struct{}{}: // Acquire semaphore
				go func() {
					defer func() { <-semaphore }() // Release semaphore
					w.processNextTask()
				}()
			default:
				// Skip if at max concurrent limit
			}
		}
	}
}

func (w *Worker) Stop() {
	log.Printf("Worker %s stopping...", w.id)
	w.cancel()
}

func (w *Worker) processNextTask() {
	task, err := w.queue.AcquireTask()
	if err != nil {
		log.Printf("Error acquiring task: %v", err)
		return
	}
	
	if task == nil {
		// No tasks available
		return
	}
	
	log.Printf("Worker %s processing task %s", w.id, task.TaskID)
	
	// Start lease renewal goroutine
	leaseCtx, cancelLease := context.WithCancel(w.ctx)
	go w.renewLease(leaseCtx, task.TaskID)
	defer cancelLease()
	
	// Process the task
	err = w.processTask(task)
	if err != nil {
		log.Printf("Error processing task %s: %v", task.TaskID, err)
		w.queue.FailTask(task.TaskID, "PROCESSING_ERROR", err.Error())
	}
}

func (w *Worker) processTask(task *models.VideoTask) error {
	taskDir := filepath.Dir(task.SourcePath)
	
	// Stage 1: Probe video
	w.queue.UpdateTask(task.TaskID, models.TaskStatusProcessing, models.TaskStageProbe, 10)
	
	videoInfo, err := w.processor.Probe(task.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to probe video: %w", err)
	}
	
	// Stage 2: Slice video
	w.queue.UpdateTask(task.TaskID, models.TaskStatusProcessing, models.TaskStageSlicing, 30)
	
	clipsDir := filepath.Join(taskDir, "clips")
	if err := os.MkdirAll(clipsDir, 0755); err != nil {
		return fmt.Errorf("failed to create clips directory: %w", err)
	}
	
	clipFiles, err := w.processor.SliceVideo(task.SourcePath, clipsDir, task.SegmentSec)
	if err != nil {
		return fmt.Errorf("failed to slice video: %w", err)
	}
	
	// Stage 3: Extract frames
	w.queue.UpdateTask(task.TaskID, models.TaskStatusProcessing, models.TaskStageExtractingFrames, 60)
	
	framesDir := filepath.Join(taskDir, "frames")
	if err := os.MkdirAll(framesDir, 0755); err != nil {
		return fmt.Errorf("failed to create frames directory: %w", err)
	}
	
	clips := make([]models.ClipInfo, len(clipFiles))
	totalFrames := 0
	
	for i, clipFile := range clipFiles {
		clipID := fmt.Sprintf("c%03d", i)
		clipFramesDir := filepath.Join(framesDir, clipID)
		if err := os.MkdirAll(clipFramesDir, 0755); err != nil {
			return fmt.Errorf("failed to create clip frames directory: %w", err)
		}
		
		frameFiles, err := w.processor.ExtractFrames(clipFile, clipFramesDir, task.FrameIntervalSec, task.FrameFormat, task.BgColor)
		if err != nil {
			return fmt.Errorf("failed to extract frames from clip %s: %w", clipFile, err)
		}
		
		// Convert to relative paths for manifest
		relativeFrames := make([]string, len(frameFiles))
		for j, frameFile := range frameFiles {
			relativeFrames[j] = filepath.Join("frames", clipID, filepath.Base(frameFile))
		}
		
		clips[i] = models.ClipInfo{
			ClipID:   clipID,
			StartSec: float64(i) * task.SegmentSec,
			EndSec:   float64(i+1) * task.SegmentSec,
			File:     filepath.Join("clips", filepath.Base(clipFile)),
			Frames:   relativeFrames,
		}
		
		totalFrames += len(frameFiles)
	}
	
	// Stage 4: Generate manifest
	w.queue.UpdateTask(task.TaskID, models.TaskStatusProcessing, models.TaskStageManifest, 80)
	
	manifest := models.VideoManifest{
		TaskID: task.TaskID,
		Input: models.VideoInputInfo{
			Filename:    filepath.Base(task.SourcePath),
			DurationSec: videoInfo.Duration,
			SizeBytes:   task.SourceSize,
			Width:       videoInfo.Width,
			Height:      videoInfo.Height,
			FPS:         videoInfo.FPS,
			Codec:       videoInfo.Codec,
		},
		Params: models.ProcessingParams{
			SegmentSec:       task.SegmentSec,
			FrameIntervalSec: task.FrameIntervalSec,
			FrameFormat:      task.FrameFormat,
			BgColor:          task.BgColor,
			ZipFormat:        task.ZipFormat,
		},
		Clips: clips,
		Stats: models.ProcessingStats{
			ClipCount:  len(clips),
			FrameCount: totalFrames,
		},
		CreatedAt: time.Now(),
	}
	
	manifestPath := filepath.Join(taskDir, "manifest.json")
	if err := w.storage.SaveManifest(manifestPath, &manifest); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}
	
	// Stage 5: Package result
	w.queue.UpdateTask(task.TaskID, models.TaskStatusProcessing, models.TaskStagePackaging, 90)
	
	resultPath := filepath.Join(taskDir, "result."+task.ZipFormat)
	if err := w.storage.PackageResult(taskDir, resultPath, task.ZipFormat); err != nil {
		return fmt.Errorf("failed to package result: %w", err)
	}
	
	// Get result file size
	stat, err := os.Stat(resultPath)
	if err != nil {
		return fmt.Errorf("failed to stat result file: %w", err)
	}
	
	// Complete task
	if err := w.queue.CompleteTask(task.TaskID, resultPath, manifestPath, stat.Size()); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}
	
	log.Printf("Task %s completed successfully", task.TaskID)
	return nil
}

func (w *Worker) renewLease(ctx context.Context, taskID string) {
	ticker := time.NewTicker(w.leaseRenewal)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.queue.RenewLease(taskID); err != nil {
				log.Printf("Failed to renew lease for task %s: %v", taskID, err)
				return
			}
		}
	}
}