package storage

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/warriorguo/videoslicer/pkg/models"
)

type Local struct {
	baseDir string
}

func NewLocal(baseDir string) *Local {
	return &Local{
		baseDir: baseDir,
	}
}

func (l *Local) SaveManifest(manifestPath string, manifest *models.VideoManifest) error {
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}
	
	return nil
}

func (l *Local) PackageResult(taskDir, resultPath, format string) error {
	switch format {
	case "zip":
		return l.createZip(taskDir, resultPath)
	case "tar.gz":
		return l.createTarGz(taskDir, resultPath)
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

func (l *Local) createZip(taskDir, resultPath string) error {
	zipFile, err := os.Create(resultPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()
	
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()
	
	taskName := filepath.Base(taskDir)
	
	err = filepath.Walk(taskDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the result file itself and source file
		if filePath == resultPath || strings.HasPrefix(filepath.Base(filePath), "source_") {
			return nil
		}
		
		// Skip directories, but continue walking
		if info.IsDir() {
			return nil
		}
		
		// Create relative path from task directory
		relPath, err := filepath.Rel(taskDir, filePath)
		if err != nil {
			return err
		}
		
		// Add task name prefix to maintain structure
		zipPath := filepath.Join(taskName, relPath)
		
		// Create zip entry
		zipEntry, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry: %w", err)
		}
		
		// Copy file content
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		
		_, err = io.Copy(zipEntry, file)
		if err != nil {
			return fmt.Errorf("failed to copy file to zip: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create zip archive: %w", err)
	}
	
	return nil
}

func (l *Local) createTarGz(taskDir, resultPath string) error {
	tarFile, err := os.Create(resultPath)
	if err != nil {
		return fmt.Errorf("failed to create tar.gz file: %w", err)
	}
	defer tarFile.Close()
	
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()
	
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	taskName := filepath.Base(taskDir)
	
	err = filepath.Walk(taskDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip the result file itself and source file
		if filePath == resultPath || strings.HasPrefix(filepath.Base(filePath), "source_") {
			return nil
		}
		
		// Skip directories, but continue walking
		if info.IsDir() {
			return nil
		}
		
		// Create relative path from task directory
		relPath, err := filepath.Rel(taskDir, filePath)
		if err != nil {
			return err
		}
		
		// Add task name prefix to maintain structure
		tarPath := filepath.Join(taskName, relPath)
		
		// Create tar header
		header := &tar.Header{
			Name:    tarPath,
			Mode:    int64(info.Mode()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}
		
		// Copy file content
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return fmt.Errorf("failed to copy file to tar: %w", err)
		}
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to create tar.gz archive: %w", err)
	}
	
	return nil
}

func (l *Local) DeleteTask(taskDir string) error {
	return os.RemoveAll(taskDir)
}

func (l *Local) GetTaskPath(taskID string) string {
	return filepath.Join(l.baseDir, taskID)
}

func (l *Local) EnsureTaskDir(taskID string) (string, error) {
	taskPath := l.GetTaskPath(taskID)
	if err := os.MkdirAll(taskPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create task directory: %w", err)
	}
	return taskPath, nil
}