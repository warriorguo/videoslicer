package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/warriorguo/videoslicer/internal/config"
	"github.com/warriorguo/videoslicer/internal/database"
	"github.com/warriorguo/videoslicer/pkg/api"
	"github.com/warriorguo/videoslicer/pkg/ffmpeg"
	"github.com/warriorguo/videoslicer/pkg/worker"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Check FFmpeg availability
	if !ffmpeg.IsFFmpegAvailable() {
		log.Fatal("FFmpeg is not available in PATH")
	}
	if !ffmpeg.IsFFprobeAvailable() {
		log.Fatal("FFprobe is not available in PATH")
	}
	
	// Create task directory
	if err := os.MkdirAll(cfg.Storage.TaskDir, 0755); err != nil {
		log.Fatalf("Failed to create task directory: %v", err)
	}
	
	// Connect to database
	db, err := database.NewDB(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run schema migrations
	if err := db.EnsureSchema(); err != nil {
		log.Fatalf("Failed to run schema migrations: %v", err)
	}

	// Create API server
	server := api.NewServer(db, api.Config{
		Port:        cfg.Server.Port,
		TaskDir:     cfg.Storage.TaskDir,
		MaxFileSize: cfg.Server.MaxFileSize,
	})
	
	// Create worker
	w := worker.NewWorker(db, worker.Config{
		MaxConcurrent: cfg.Worker.MaxConcurrent,
		PollInterval:  cfg.Worker.PollInterval,
		LeaseDuration: cfg.Worker.LeaseDuration,
		TaskDir:       cfg.Storage.TaskDir,
	})
	
	// Setup graceful shutdown
	
	var wg sync.WaitGroup
	
	// Start API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Starting API server on port %s", cfg.Server.Port)
		if err := server.Start(); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()
	
	// Start worker
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("Starting worker")
		w.Start()
	}()
	
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	log.Println("Shutdown signal received")
	
	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	
	// Stop worker
	w.Stop()
	
	// Stop server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	// Cancel context to stop remaining goroutines
	// cancel() // Removed as context is not used
	
	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		log.Println("Shutdown completed")
	case <-shutdownCtx.Done():
		log.Println("Shutdown timeout exceeded")
	}
}