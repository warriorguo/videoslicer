package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables to test defaults
	envVars := []string{
		"SERVER_PORT", "MAX_FILE_SIZE", "DB_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "WORKER_MAX_CONCURRENT",
		"WORKER_POLL_INTERVAL", "WORKER_LEASE_DURATION", "WORKER_TIMEOUT",
		"STORAGE_TASK_DIR",
	}
	
	// Save original values
	originalValues := make(map[string]string)
	for _, envVar := range envVars {
		originalValues[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	
	// Restore original values after test
	defer func() {
		for envVar, value := range originalValues {
			if value != "" {
				os.Setenv(envVar, value)
			}
		}
	}()
	
	// Set required environment variables to avoid validation errors
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with defaults: %v", err)
	}
	
	// Test defaults
	if config.Server.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", config.Server.Port)
	}
	if config.Server.MaxFileSize != 1024*1024*1024 {
		t.Errorf("Expected default max file size 1GB, got %d", config.Server.MaxFileSize)
	}
	if config.Database.Host != "localhost" {
		t.Errorf("Expected default DB host localhost, got %s", config.Database.Host)
	}
	if config.Database.Port != 5432 {
		t.Errorf("Expected default DB port 5432, got %d", config.Database.Port)
	}
	if config.Worker.MaxConcurrent != 2 {
		t.Errorf("Expected default max concurrent 2, got %d", config.Worker.MaxConcurrent)
	}
	if config.Worker.PollInterval != 5*time.Second {
		t.Errorf("Expected default poll interval 5s, got %v", config.Worker.PollInterval)
	}
	if config.Storage.TaskDir != "./tasks" {
		t.Errorf("Expected default task dir ./tasks, got %s", config.Storage.TaskDir)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	// Set custom environment variables
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("MAX_FILE_SIZE", "2147483648") // 2GB
	os.Setenv("DB_HOST", "custom-db-host")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "customuser")
	os.Setenv("DB_PASSWORD", "custompass")
	os.Setenv("DB_NAME", "customdb")
	os.Setenv("DB_SSLMODE", "require")
	os.Setenv("WORKER_MAX_CONCURRENT", "4")
	os.Setenv("WORKER_POLL_INTERVAL", "10s")
	os.Setenv("WORKER_LEASE_DURATION", "10m")
	os.Setenv("WORKER_TIMEOUT", "60m")
	os.Setenv("STORAGE_TASK_DIR", "/custom/tasks")
	
	// Clean up after test
	defer func() {
		envVars := []string{
			"SERVER_PORT", "MAX_FILE_SIZE", "DB_HOST", "DB_PORT", "DB_USER",
			"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "WORKER_MAX_CONCURRENT",
			"WORKER_POLL_INTERVAL", "WORKER_LEASE_DURATION", "WORKER_TIMEOUT",
			"STORAGE_TASK_DIR",
		}
		for _, envVar := range envVars {
			os.Unsetenv(envVar)
		}
	}()
	
	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with custom values: %v", err)
	}
	
	// Test custom values
	if config.Server.Port != "9090" {
		t.Errorf("Expected custom port 9090, got %s", config.Server.Port)
	}
	if config.Server.MaxFileSize != 2147483648 {
		t.Errorf("Expected custom max file size 2GB, got %d", config.Server.MaxFileSize)
	}
	if config.Database.Host != "custom-db-host" {
		t.Errorf("Expected custom DB host, got %s", config.Database.Host)
	}
	if config.Database.Port != 5433 {
		t.Errorf("Expected custom DB port 5433, got %d", config.Database.Port)
	}
	if config.Worker.MaxConcurrent != 4 {
		t.Errorf("Expected custom max concurrent 4, got %d", config.Worker.MaxConcurrent)
	}
	if config.Worker.PollInterval != 10*time.Second {
		t.Errorf("Expected custom poll interval 10s, got %v", config.Worker.PollInterval)
	}
	if config.Storage.TaskDir != "/custom/tasks" {
		t.Errorf("Expected custom task dir, got %s", config.Storage.TaskDir)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	testCases := []struct {
		name       string
		setupEnv   func()
		expectsErr bool
	}{
		{
			name: "Missing DB_HOST",
			setupEnv: func() {
				os.Unsetenv("DB_HOST")
				os.Setenv("REQUIRE_CONFIG_VALIDATION", "true")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
			},
			expectsErr: true,
		},
		{
			name: "Missing DB_USER",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Unsetenv("DB_USER")
				os.Setenv("REQUIRE_CONFIG_VALIDATION", "true")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
			},
			expectsErr: true,
		},
		{
			name: "Missing DB_PASSWORD",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_USER", "user")
				os.Unsetenv("DB_PASSWORD")
				os.Setenv("REQUIRE_CONFIG_VALIDATION", "true")
				os.Setenv("DB_NAME", "db")
			},
			expectsErr: true,
		},
		{
			name: "Missing DB_NAME",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Unsetenv("DB_NAME")
				os.Setenv("REQUIRE_CONFIG_VALIDATION", "true")
			},
			expectsErr: true,
		},
		{
			name: "All required present",
			setupEnv: func() {
				os.Setenv("DB_HOST", "localhost")
				os.Setenv("DB_USER", "user")
				os.Setenv("DB_PASSWORD", "pass")
				os.Setenv("DB_NAME", "db")
			},
			expectsErr: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up before each test
			os.Unsetenv("REQUIRE_CONFIG_VALIDATION")
			tc.setupEnv()
			
			_, err := Load()
			
			if tc.expectsErr && err == nil {
				t.Errorf("Expected error for test case %s, but got nil", tc.name)
			}
			if !tc.expectsErr && err != nil {
				t.Errorf("Expected no error for test case %s, but got: %v", tc.name, err)
			}
			
			// Clean up after each test
			os.Unsetenv("REQUIRE_CONFIG_VALIDATION")
		})
	}
}