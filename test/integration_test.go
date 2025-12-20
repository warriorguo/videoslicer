package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/warriorguo/videoslicer/internal/config"
	"github.com/warriorguo/videoslicer/internal/database"
	"github.com/warriorguo/videoslicer/pkg/api"
	"github.com/warriorguo/videoslicer/pkg/models"
)

func TestIntegration_FullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Setup test database
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// Create temporary directory for tasks
	tempDir := t.TempDir()
	
	// Create test server
	server := api.NewServer(db, api.Config{
		Port:        "0", // Use any available port
		TaskDir:     tempDir,
		MaxFileSize: 10 * 1024 * 1024, // 10MB for testing
	})
	
	// Create test HTTP server
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()
	
	// Test 1: Create task with mock video file
	t.Run("CreateTask", func(t *testing.T) {
		taskID := createTaskTest(t, testServer.URL)
		if taskID == "" {
			t.Fatal("Failed to create task")
		}
		
		// Test 2: Check task status
		t.Run("CheckTaskStatus", func(t *testing.T) {
			checkTaskStatusTest(t, testServer.URL, taskID)
		})
		
		// Test 3: Try to download result (should fail since no actual processing)
		t.Run("DownloadResult", func(t *testing.T) {
			downloadResultTest(t, testServer.URL, taskID)
		})
	})
}

func createTaskTest(t *testing.T, baseURL string) string {
	// Create a mock video file
	mockVideoContent := []byte("mock video content - this is not a real video")
	
	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	
	// Add file part
	part, err := writer.CreateFormFile("file", "test_video.mp4")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	part.Write(mockVideoContent)
	
	// Add parameters
	writer.WriteField("segment_sec", "5")
	writer.WriteField("frame_interval_sec", "1")
	writer.WriteField("frame_format", "jpg")
	writer.WriteField("zip_format", "zip")
	
	writer.Close()
	
	// Create request
	req, err := http.NewRequest("POST", baseURL+"/v1/tasks", &buf)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	
	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var createResp struct {
		TaskID    string    `json:"task_id"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Validate response
	if createResp.TaskID == "" {
		t.Fatal("Task ID is empty")
	}
	if createResp.Status != "queued" {
		t.Errorf("Expected status 'queued', got '%s'", createResp.Status)
	}
	
	t.Logf("Created task with ID: %s", createResp.TaskID)
	return createResp.TaskID
}

func checkTaskStatusTest(t *testing.T, baseURL, taskID string) {
	// Create request
	req, err := http.NewRequest("GET", baseURL+"/v1/tasks/"+taskID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var taskResp models.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Validate response
	if taskResp.TaskID != taskID {
		t.Errorf("Expected task ID '%s', got '%s'", taskID, taskResp.TaskID)
	}
	if taskResp.Status != models.TaskStatusQueued {
		t.Logf("Task status: %s", taskResp.Status)
	}
	
	t.Logf("Task status: %s, progress: %.1f%%", taskResp.Progress.Stage, taskResp.Progress.Percent)
}

func downloadResultTest(t *testing.T, baseURL, taskID string) {
	// Create request
	req, err := http.NewRequest("GET", baseURL+"/v1/tasks/"+taskID+"/result", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	// Should return 404 since processing hasn't completed
	if resp.StatusCode != http.StatusNotFound {
		t.Logf("Download result status: %d (expected 404 for unprocessed task)", resp.StatusCode)
	}
}

func TestAPI_InvalidRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping API test in short mode")
	}
	
	// Setup test database
	db, cleanup := setupTestDB(t)
	defer cleanup()
	
	// Create temporary directory for tasks
	tempDir := t.TempDir()
	
	// Create test server
	server := api.NewServer(db, api.Config{
		Port:        "0",
		TaskDir:     tempDir,
		MaxFileSize: 1024, // Very small for testing
	})
	
	testServer := httptest.NewServer(server.Handler())
	defer testServer.Close()
	
	t.Run("NoFile", func(t *testing.T) {
		resp, err := http.Post(testServer.URL+"/v1/tasks", "application/json", bytes.NewReader([]byte("{}")))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
	
	t.Run("NonExistentTask", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/v1/tasks/nonexistent")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", resp.StatusCode)
		}
	})
	
	t.Run("HealthCheck", func(t *testing.T) {
		resp, err := http.Get(testServer.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "OK" {
			t.Errorf("Expected 'OK', got '%s'", string(body))
		}
	})
}

func setupTestDB(t *testing.T) (*database.DB, func()) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
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
		t.Fatalf("Failed to connect to database: %v", err)
	}
	
	// Create a unique table name for this test
	testTableSuffix := fmt.Sprintf("_test_%d", time.Now().UnixNano())
	
	// Create test table
	schema := `
		CREATE TABLE IF NOT EXISTS video_tasks` + testTableSuffix + ` (
			task_id VARCHAR(64) PRIMARY KEY,
			status VARCHAR(20) NOT NULL DEFAULT 'queued',
			stage VARCHAR(30) NOT NULL DEFAULT '',
			progress_percent DECIMAL(5,2) NOT NULL DEFAULT 0.0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			
			source_path TEXT NOT NULL,
			source_size BIGINT NOT NULL DEFAULT 0,
			
			result_path TEXT,
			result_size BIGINT NOT NULL DEFAULT 0,
			manifest_path TEXT,
			
			segment_sec INTEGER NOT NULL DEFAULT 8,
			frame_interval_sec INTEGER NOT NULL DEFAULT 2,
			frame_format VARCHAR(10) NOT NULL DEFAULT 'jpg',
			zip_format VARCHAR(10) NOT NULL DEFAULT 'zip',
			
			error_code VARCHAR(50),
			error_message TEXT,
			
			lease_owner VARCHAR(100),
			lease_expires_at TIMESTAMP,
			
			callback_url TEXT
		);
	`
	
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	
	cleanup := func() {
		// Drop test table
		dropSQL := "DROP TABLE IF EXISTS video_tasks" + testTableSuffix
		db.Exec(dropSQL)
		db.Close()
	}
	
	return db, cleanup
}

func TestDatabaseConnection(t *testing.T) {
	// Test database connection with current configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	db, err := database.NewDB(database.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		DBName:   cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Test simple query
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute test query: %v", err)
	}
	
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
	
	t.Log("Database connection successful")
}