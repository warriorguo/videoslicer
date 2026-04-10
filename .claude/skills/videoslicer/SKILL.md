---
name: videoslicer
description: Interact with the VideoSlicer API to upload videos, slice them into clips, extract frames, and download results. Use when the user wants to process videos, check task status, or download processed results.
allowed-tools: Bash Read
argument-hint: [upload|status|download|help] [args...]
---

# VideoSlicer Skill

This skill interacts with the VideoSlicer API for video processing tasks: slicing videos into clips, extracting frames, and packaging results.

## Activation

When activated, first determine the API base URL:
- Default: `https://videoslicer.local.playquota.com`
- If the user provides a different URL, use that instead.

Store as `BASE_URL` for all API calls.

## API Reference

### Health Check
```
GET /health
```
Returns "OK" if the service is running.

### Create Task (Upload Video)
```
POST /v1/tasks
Content-Type: multipart/form-data
```

**Form fields:**
- `file` (required): Video file (.mp4, .mov, .avi, .mkv, .webm, .flv, .wmv)
- `segment_sec` (optional, default: 8): Duration of each video clip in seconds
- `frame_interval_sec` (optional, default: 2): Frame extraction interval in seconds
- `frame_format` (optional, default: "jpg"): Frame image format ("jpg" or "png")
- `bg_color` (optional, default: "black"): Background color for frame extraction, composited behind transparent videos (e.g. WebM with alpha). Accepts any ffmpeg color name or hex value (e.g. "black", "white", "0x1a1a1a")
- `zip_format` (optional, default: "zip"): Archive format ("zip" or "tar.gz")

**Response (201):**
```json
{
  "task_id": "t_cmkl9abc123def456",
  "status": "queued",
  "created_at": "2025-12-20T15:30:45Z"
}
```

**Example:**
```bash
curl -X POST ${BASE_URL}/v1/tasks \
  -F "file=@video.mp4" \
  -F "segment_sec=5" \
  -F "frame_interval_sec=1" \
  -F "frame_format=jpg" \
  -F "bg_color=black"
```

### Get Task Status
```
GET /v1/tasks/{task_id}
```

**Response (200):**
```json
{
  "task_id": "t_xxx",
  "status": "processing",
  "progress": {
    "stage": "slicing",
    "percent": 45.0
  },
  "params": {
    "segment_sec": 5,
    "frame_interval_sec": 2,
    "frame_format": "jpg",
    "bg_color": "black",
    "zip_format": "zip"
  },
  "result": {
    "zip_size_bytes": 1234567
  },
  "error": {
    "code": "FFMPEG_ERROR",
    "message": "..."
  },
  "created_at": "2025-12-20T15:30:45Z"
}
```

**Status values:** `queued` -> `processing` -> `succeeded` / `failed` / `expired`

**Processing stages:** `download_source` -> `probe` -> `slicing` -> `extracting_frames` -> `manifest` -> `packaging` -> `uploading` -> `completed`

### Download Result
```
GET /v1/tasks/{task_id}/result
```
Returns the archive file (zip or tar.gz) when task status is `succeeded`.

**Example:**
```bash
curl -o result.zip ${BASE_URL}/v1/tasks/{task_id}/result
```

**Archive structure:**
```
task_t_xxx/
├── manifest.json
├── clips/
│   ├── c000.mp4
│   ├── c001.mp4
│   └── ...
└── frames/
    ├── c000/
    │   ├── f0001.jpg
    │   └── f0002.jpg
    └── c001/
        └── ...
```

## Common Workflows

### 1. Upload and Process a Video

```bash
# Upload
curl -s -X POST ${BASE_URL}/v1/tasks \
  -F "file=@/path/to/video.mp4" \
  -F "segment_sec=10" \
  -F "frame_interval_sec=2" | python3 -m json.tool

# Poll status (repeat until succeeded or failed)
curl -s ${BASE_URL}/v1/tasks/{task_id} | python3 -m json.tool

# Download result
curl -o result.zip ${BASE_URL}/v1/tasks/{task_id}/result
```

### 2. Check Service Health

```bash
curl ${BASE_URL}/health
```

## Tips

1. **Poll task status** every 5-10 seconds when waiting for processing to complete.
2. **Large videos** take longer — monitor the `progress.percent` field.
3. **Smaller `segment_sec`** creates more clips; larger values create fewer, longer clips.
4. **Frame interval** controls density — `frame_interval_sec=1` extracts one frame per second per clip.
5. Task must be `succeeded` before downloading results.
6. Max upload file size is 1GB by default.
