package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Processor struct {
	timeout time.Duration
}

type VideoInfo struct {
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
	FPS      float64 `json:"fps"`
	Codec    string  `json:"codec"`
}

type FFProbeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

type FFProbeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	RFrameRate string `json:"r_frame_rate"`
}

type FFProbeResult struct {
	Format  FFProbeFormat   `json:"format"`
	Streams []FFProbeStream `json:"streams"`
}

func NewProcessor() *Processor {
	return &Processor{
		timeout: 30 * time.Minute, // Default timeout for FFmpeg operations
	}
}

func (p *Processor) Probe(videoPath string) (*VideoInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_format",
		"-show_streams",
		"-of", "json",
		videoPath)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}
	
	var result FFProbeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}
	
	info := &VideoInfo{}
	
	// Parse duration
	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.Duration = duration
		}
	}
	
	// Find video stream
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.Codec = stream.CodecName
			
			// Parse frame rate
			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
						if den, err := strconv.ParseFloat(parts[1], 64); err == nil && den != 0 {
							info.FPS = num / den
						}
					}
				}
			}
			break
		}
	}
	
	return info, nil
}

func (p *Processor) SliceVideo(inputPath, outputDir string, segmentSec float64) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	outputPattern := filepath.Join(outputDir, "c%03d.mp4")
	segmentTime := fmt.Sprintf("%.4f", segmentSec)

	// Try fast copy mode first
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-y",
		"-i", inputPath,
		"-c", "copy",
		"-map", "0",
		"-f", "segment",
		"-segment_time", segmentTime,
		"-reset_timestamps", "1",
		outputPattern)

	if err := cmd.Run(); err != nil {
		// Fallback to re-encoding mode
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-hide_banner", "-y",
			"-i", inputPath,
			"-c:v", "libx264",
			"-preset", "veryfast",
			"-crf", "23",
			"-c:a", "aac",
			"-b:a", "128k",
			"-f", "segment",
			"-segment_time", segmentTime,
			"-reset_timestamps", "1",
			outputPattern)
		
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("ffmpeg slice failed: %w", err)
		}
	}
	
	// Find generated clip files
	pattern := filepath.Join(outputDir, "c*.mp4")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find clip files: %w", err)
	}
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no clip files generated")
	}
	
	return matches, nil
}

func (p *Processor) ExtractFrames(clipPath, outputDir string, intervalSec float64, format string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()

	outputPattern := filepath.Join(outputDir, "f%04d."+format)

	var cmd *exec.Cmd
	if intervalSec <= 0 {
		// Extract every frame
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-hide_banner", "-y",
			"-i", clipPath,
			outputPattern)
	} else {
		fpsFilter := fmt.Sprintf("fps=1/%g", intervalSec)
		cmd = exec.CommandContext(ctx, "ffmpeg",
			"-hide_banner", "-y",
			"-i", clipPath,
			"-vf", fpsFilter,
			outputPattern)
	}

	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg frame extraction failed: %w, stderr: %s", err, stderr.String())
	}
	
	// Find generated frame files
	pattern := filepath.Join(outputDir, "f*."+format)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find frame files: %w", err)
	}
	
	return matches, nil
}

func (p *Processor) ExtractFramesFixedCount(clipPath, outputDir string, count int, format string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
	defer cancel()
	
	// First, get clip duration
	info, err := p.Probe(clipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe clip: %w", err)
	}
	
	if info.Duration <= 0 {
		return nil, fmt.Errorf("invalid clip duration")
	}
	
	outputPattern := filepath.Join(outputDir, "f%04d."+format)
	
	// Calculate fps for fixed count
	fps := float64(count) / info.Duration
	fpsFilter := fmt.Sprintf("fps=%.6f", fps)
	
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner", "-y",
		"-i", clipPath,
		"-vf", fpsFilter,
		"-frames:v", strconv.Itoa(count),
		outputPattern)
	
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg frame extraction failed: %w", err)
	}
	
	// Find generated frame files
	pattern := filepath.Join(outputDir, "f*."+format)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find frame files: %w", err)
	}
	
	return matches, nil
}

func (p *Processor) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

func (p *Processor) GetTimeout() time.Duration {
	return p.timeout
}

func IsFFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func IsFFprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}