# Video Slicer API 使用示例

## 前提条件

确保服务已启动：
```bash
go run cmd/server/main.go
# 或者
./bin/videoslicer
```

服务默认运行在 `http://localhost:8080`

## 1. 健康检查

```bash
curl -X GET http://localhost:8080/health
```

**预期响应：**
```
OK
```

## 2. 创建任务（上传视频）

### 基本用法

```bash
# 上传视频文件，使用默认参数
curl -X POST \
  -F "file=@/path/to/your/video.mp4" \
  http://localhost:8080/v1/tasks
```

### 完整参数

```bash
# 上传视频，自定义所有参数
curl -X POST \
  -F "file=@/path/to/your/video.mp4" \
  -F "segment_sec=10" \
  -F "frame_interval_sec=3" \
  -F "frame_format=png" \
  -F "zip_format=tar.gz" \
  http://localhost:8080/v1/tasks
```

### 使用测试文件

如果没有视频文件，可以创建一个测试文件：

```bash
# 创建一个测试文件（模拟视频）
echo "This is a test video file" > test_video.mp4

# 上传测试文件
curl -X POST \
  -F "file=@test_video.mp4" \
  -F "segment_sec=5" \
  -F "frame_interval_sec=2" \
  http://localhost:8080/v1/tasks
```

**预期响应：**
```json
{
  "task_id": "t_cmkl9abc123def456",
  "status": "queued",
  "created_at": "2025-12-20T15:30:45Z"
}
```

**参数说明：**
- `file`: 视频文件（必需）
- `segment_sec`: 切片时长（秒），默认 8
- `frame_interval_sec`: 抽帧间隔（秒），默认 2
- `frame_format`: 帧格式（jpg/png），默认 jpg
- `zip_format`: 打包格式（zip/tar.gz），默认 zip

## 3. 查询任务状态

```bash
# 替换为实际的 task_id
TASK_ID="t_cmkl9abc123def456"

curl -X GET http://localhost:8080/v1/tasks/$TASK_ID
```

**可能的响应（处理中）：**
```json
{
  "task_id": "t_cmkl9abc123def456",
  "status": "processing",
  "progress": {
    "stage": "slicing",
    "percent": 45.0
  },
  "params": {
    "segment_sec": 5,
    "frame_interval_sec": 2,
    "frame_format": "jpg",
    "zip_format": "zip"
  },
  "created_at": "2025-12-20T15:30:45Z"
}
```

**可能的响应（已完成）：**
```json
{
  "task_id": "t_cmkl9abc123def456",
  "status": "succeeded",
  "progress": {
    "stage": "completed",
    "percent": 100.0
  },
  "params": {
    "segment_sec": 5,
    "frame_interval_sec": 2,
    "frame_format": "jpg",
    "zip_format": "zip"
  },
  "result": {
    "zip_size_bytes": 1234567
  },
  "created_at": "2025-12-20T15:30:45Z"
}
```

**可能的响应（失败）：**
```json
{
  "task_id": "t_cmkl9abc123def456",
  "status": "failed",
  "progress": {
    "stage": "slicing",
    "percent": 30.0
  },
  "params": {
    "segment_sec": 5,
    "frame_interval_sec": 2,
    "frame_format": "jpg",
    "zip_format": "zip"
  },
  "error": {
    "code": "FFMPEG_ERROR",
    "message": "Failed to slice video: invalid format"
  },
  "created_at": "2025-12-20T15:30:45Z"
}
```

**状态说明：**
- `queued`: 已创建，等待处理
- `processing`: 处理中
- `succeeded`: 处理成功，可下载
- `failed`: 处理失败
- `expired`: 结果过期

**处理阶段：**
- `download_source`: 下载源文件
- `probe`: 分析视频信息
- `slicing`: 切片中
- `extracting_frames`: 抽帧中
- `manifest`: 生成清单
- `packaging`: 打包中
- `uploading`: 上传结果

## 4. 下载处理结果

```bash
# 替换为实际的 task_id
TASK_ID="t_cmkl9abc123def456"

# 下载结果文件
curl -X GET \
  -o "result_${TASK_ID}.zip" \
  http://localhost:8080/v1/tasks/$TASK_ID/result
```

**成功响应：**
- HTTP 状态码：200
- Content-Type: application/octet-stream
- 文件内容：ZIP 或 TAR.GZ 格式的压缩包

**错误响应（任务未完成）：**
```
HTTP/1.1 404 Not Found
Result not available
```

## 5. 完整的工作流程示例

```bash
#!/bin/bash

echo "=== Video Slicer API 完整流程示例 ==="

# 1. 健康检查
echo "1. 检查服务健康状态..."
curl -s http://localhost:8080/health
echo

# 2. 创建测试文件
echo "2. 创建测试视频文件..."
echo "This is a test video file for API demonstration" > test_video.mp4
ls -la test_video.mp4

# 3. 上传文件并创建任务
echo "3. 上传视频并创建任务..."
RESPONSE=$(curl -s -X POST \
  -F "file=@test_video.mp4" \
  -F "segment_sec=5" \
  -F "frame_interval_sec=2" \
  http://localhost:8080/v1/tasks)

echo "创建任务响应: $RESPONSE"

# 4. 提取 task_id
TASK_ID=$(echo $RESPONSE | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
echo "任务 ID: $TASK_ID"

if [ -z "$TASK_ID" ]; then
    echo "❌ 无法获取任务 ID，请检查服务是否正常运行"
    exit 1
fi

# 5. 轮询任务状态
echo "4. 监控任务处理进度..."
for i in {1..30}; do
    STATUS_RESPONSE=$(curl -s http://localhost:8080/v1/tasks/$TASK_ID)
    STATUS=$(echo $STATUS_RESPONSE | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    STAGE=$(echo $STATUS_RESPONSE | grep -o '"stage":"[^"]*"' | cut -d'"' -f4)
    
    echo "[$i/30] 状态: $STATUS, 阶段: $STAGE"
    
    if [ "$STATUS" = "succeeded" ]; then
        echo "✅ 任务处理成功！"
        break
    elif [ "$STATUS" = "failed" ]; then
        echo "❌ 任务处理失败"
        echo "详细信息: $STATUS_RESPONSE"
        exit 1
    fi
    
    sleep 2
done

# 6. 下载结果
if [ "$STATUS" = "succeeded" ]; then
    echo "5. 下载处理结果..."
    curl -o "result_${TASK_ID}.zip" \
         http://localhost:8080/v1/tasks/$TASK_ID/result
    
    echo "✅ 结果已下载到 result_${TASK_ID}.zip"
    ls -la result_${TASK_ID}.zip
    
    # 可选：查看压缩包内容
    echo "📁 压缩包内容:"
    unzip -l result_${TASK_ID}.zip
else
    echo "⏰ 任务仍在处理中，请稍后手动检查"
fi

# 7. 清理
rm test_video.mp4

echo "=== 流程完成 ==="
```

## 6. 错误处理示例

### 无效文件格式

```bash
echo "invalid content" > test.txt
curl -X POST -F "file=@test.txt" http://localhost:8080/v1/tasks
```

**响应：**
```
HTTP/1.1 400 Bad Request
Invalid file format
```

### 文件过大

```bash
# 创建一个大文件（假设超过限制）
dd if=/dev/zero of=large_file.mp4 bs=1M count=1100

curl -X POST -F "file=@large_file.mp4" http://localhost:8080/v1/tasks
```

**响应：**
```
HTTP/1.1 413 Request Entity Too Large
File too large
```

### 任务不存在

```bash
curl -X GET http://localhost:8080/v1/tasks/nonexistent_task_id
```

**响应：**
```json
HTTP/1.1 404 Not Found
Task not found
```

## 7. 高级用法

### 并发上传多个文件

```bash
#!/bin/bash

# 并发创建多个任务
for i in {1..5}; do
    echo "Test video $i" > test_video_$i.mp4
    (
        echo "上传 test_video_$i.mp4..."
        curl -s -X POST \
          -F "file=@test_video_$i.mp4" \
          -F "segment_sec=$((5+i))" \
          http://localhost:8080/v1/tasks
    ) &
done

wait
echo "所有文件上传完成"
```

### 监控多个任务

```bash
# 获取所有任务的状态（需要实现批量查询 API）
# 当前版本需要逐个查询
TASK_IDS=("task1" "task2" "task3")

for task_id in "${TASK_IDS[@]}"; do
    echo "任务 $task_id 状态："
    curl -s http://localhost:8080/v1/tasks/$task_id | jq '.status,.progress'
done
```

## 8. 性能测试

### 压力测试

```bash
#!/bin/bash

# 创建测试文件
echo "Performance test video" > perf_test.mp4

# 并发发送请求
echo "开始压力测试（10个并发请求）..."
for i in {1..10}; do
    (
        time curl -s -X POST \
          -F "file=@perf_test.mp4" \
          http://localhost:8080/v1/tasks > /dev/null
        echo "请求 $i 完成"
    ) &
done

wait
echo "压力测试完成"
```

## 9. 开发调试

### 详细输出

```bash
# 显示详细的 HTTP 信息
curl -v -X POST \
  -F "file=@test_video.mp4" \
  http://localhost:8080/v1/tasks

# 显示响应头
curl -I http://localhost:8080/health

# 保存响应到文件
curl -o response.json \
  http://localhost:8080/v1/tasks/your_task_id
```

### JSON 格式化

```bash
# 使用 jq 格式化 JSON 响应
curl -s http://localhost:8080/v1/tasks/your_task_id | jq '.'

# 只显示特定字段
curl -s http://localhost:8080/v1/tasks/your_task_id | jq '.status,.progress'
```

---

## 注意事项

1. **文件路径**：确保文件路径正确，使用绝对路径或相对路径
2. **文件格式**：支持的视频格式：mp4, mov, avi, mkv, webm, flv, wmv
3. **文件大小**：默认最大 1GB，可在配置中调整
4. **超时设置**：大文件处理可能需要较长时间
5. **并发限制**：Worker 数量限制了并发处理的任务数
6. **存储空间**：确保有足够的磁盘空间存储临时文件和结果

## 故障排除

如果遇到问题，可以检查：
1. 服务是否正常运行：`curl http://localhost:8080/health`
2. 文件是否存在：`ls -la your_video_file.mp4`
3. 服务日志：查看控制台输出
4. 数据库连接：确保 PostgreSQL 正常运行