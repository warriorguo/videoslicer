# Video Slicer & Frame Extract Service

一个基于 Go 的 HTTP 服务，用于视频切片和抽帧处理。支持异步任务处理，可将长视频切成多个片段并抽取关键帧。

## 功能特性

- **视频切片**：按时间段将视频切分为多个短片段
- **帧抽取**：对每个片段抽取低成本关键帧
- **异步处理**：上传后立即返回 task_id，后台异步处理
- **多种格式**：支持多种视频格式输入和帧图片格式输出
- **打包下载**：处理完成后提供 ZIP/TAR.GZ 格式的打包下载

## 架构设计

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   HTTP API      │    │   Task Queue │    │   FFmpeg        │
│   (Upload)      │───▶│   (PostgreSQL)│───▶│   (切片/抽帧)    │
└─────────────────┘    └──────────────┘    └─────────────────┘
        │                       │                     │
        ▼                       ▼                     ▼
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Status API    │    │   Worker     │    │   Storage       │
│   (查询/下载)    │    │   (处理器)   │    │   (本地文件)     │
└─────────────────┘    └──────────────┘    └─────────────────┘
```

## 快速开始

### 依赖要求

- Go 1.21+
- PostgreSQL 13+
- FFmpeg 4.0+
- FFprobe 4.0+

### 安装依赖

#### macOS
```bash
brew install ffmpeg postgresql
```

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install ffmpeg postgresql postgresql-contrib
```

#### CentOS/RHEL
```bash
sudo yum install epel-release
sudo yum install ffmpeg postgresql-server
```

### 运行服务

1. **克隆项目**
```bash
git clone https://github.com/warriorguo/videoslicer
cd videoslicer
```

2. **安装 Go 依赖**
```bash
make deps
```

3. **设置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，配置数据库连接等参数
```

4. **初始化数据库**
```bash
# 启动 PostgreSQL 服务
sudo systemctl start postgresql  # Linux
brew services start postgresql   # macOS

# 方法1：使用提供的 SQL 脚本（推荐）
# 以 PostgreSQL 超级用户身份运行：
psql -U postgres -f setup_database.sql

# 方法2：手动创建（如果上述方法不可用）
psql -U postgres -c "CREATE USER videoslicer WITH PASSWORD 'password';"
psql -U postgres -c "CREATE DATABASE videoslicer OWNER videoslicer;"
psql -U videoslicer -d videoslicer -f internal/database/schema.sql

# 方法3：使用 Makefile（需要先创建用户和数据库）
make setup-db
```

5. **检查依赖**
```bash
make check-deps
```

6. **启动服务**
```bash
make run
```

服务将在 `http://localhost:8080` 启动。

## API 接口

### 1. 创建任务（上传视频）

```http
POST /v1/tasks
Content-Type: multipart/form-data
```

**参数：**
- `file`: 视频文件（必需）
- `segment_sec`: 切片时长，默认 8 秒
- `frame_interval_sec`: 抽帧间隔，默认 2 秒/帧
- `frame_format`: 帧格式，默认 "jpg"
- `zip_format`: 打包格式，默认 "zip"

**响应：**
```json
{
  "task_id": "t_01JFK...X",
  "status": "queued",
  "created_at": "2025-12-19T15:20:00+08:00"
}
```

### 2. 查询任务状态

```http
GET /v1/tasks/{task_id}
```

**响应：**
```json
{
  "task_id": "t_xxx",
  "status": "processing",
  "progress": {
    "stage": "slicing",
    "percent": 45.0
  },
  "params": {
    "segment_sec": 8,
    "frame_interval_sec": 2,
    "frame_format": "jpg",
    "zip_format": "zip"
  },
  "result": {
    "zip_size_bytes": 12345678
  },
  "created_at": "2025-12-19T15:20:00+08:00"
}
```

**状态说明：**
- `queued`: 已创建，等待处理
- `processing`: 处理中
- `succeeded`: 处理成功，可下载
- `failed`: 处理失败
- `expired`: 结果过期

### 3. 下载结果

```http
GET /v1/tasks/{task_id}/result
```

返回 ZIP/TAR.GZ 格式的压缩文件。

## 输出结构

处理完成的压缩包内部结构：

```
task_t_xxx/
├── manifest.json          # 元数据文件
├── clips/                 # 切片视频目录
│   ├── c000.mp4
│   ├── c001.mp4
│   └── c002.mp4
└── frames/                # 抽帧图片目录
    ├── c000/
    │   ├── f0001.jpg
    │   └── f0002.jpg
    ├── c001/
    │   ├── f0001.jpg
    │   └── f0002.jpg
    └── c002/
        ├── f0001.jpg
        └── f0002.jpg
```

## 配置说明

所有配置都通过环境变量设置：

| 变量名 | 描述 | 默认值 |
|--------|------|--------|
| `SERVER_PORT` | HTTP 服务端口 | `8080` |
| `MAX_FILE_SIZE` | 最大文件大小（字节） | `1073741824` (1GB) |
| `DB_HOST` | 数据库主机 | `localhost` |
| `DB_PORT` | 数据库端口 | `5432` |
| `DB_USER` | 数据库用户名 | `videoslicer` |
| `DB_PASSWORD` | 数据库密码 | `password` |
| `DB_NAME` | 数据库名称 | `videoslicer` |
| `WORKER_MAX_CONCURRENT` | 最大并发任务数 | `2` |
| `WORKER_POLL_INTERVAL` | 任务轮询间隔 | `5s` |
| `STORAGE_TASK_DIR` | 任务存储目录 | `./tasks` |

## Docker 部署

使用 Docker Compose 快速部署：

```bash
# 构建镜像
make docker-build

# 启动服务
make docker-run
```

## 开发

### 热重载开发

```bash
# 安装 air 工具
make dev-deps

# 启动热重载
make dev
```

### 代码检查

```bash
# 格式化代码
make fmt

# 代码检查
make lint

# 运行测试
make test
```

## 使用示例

### 快速测试

```bash
# 运行快速测试脚本
./quick_test.sh

# 运行完整测试
./test_api.sh

# 查看详细 API 示例
cat API_EXAMPLES.md
```

### 基本 curl 命令

#### 1. 健康检查
```bash
curl http://localhost:8080/health
```

#### 2. 上传视频
```bash
curl -X POST \
  -F "file=@video.mp4" \
  -F "segment_sec=10" \
  -F "frame_interval_sec=3" \
  http://localhost:8080/v1/tasks
```

#### 3. 查询任务状态
```bash
curl http://localhost:8080/v1/tasks/t_01JFK...X
```

#### 4. 下载结果
```bash
curl -o result.zip http://localhost:8080/v1/tasks/t_01JFK...X/result
```

### 完整工作流程
```bash
#!/bin/bash

# 1. 健康检查
curl http://localhost:8080/health

# 2. 创建测试文件
echo "Test video content" > test.mp4

# 3. 上传并获取任务ID
RESPONSE=$(curl -s -X POST -F "file=@test.mp4" http://localhost:8080/v1/tasks)
TASK_ID=$(echo "$RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)

# 4. 监控任务状态
while true; do
    STATUS=$(curl -s http://localhost:8080/v1/tasks/$TASK_ID | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    echo "状态: $STATUS"
    [ "$STATUS" = "succeeded" ] && break
    [ "$STATUS" = "failed" ] && exit 1
    sleep 2
done

# 5. 下载结果
curl -o result_$TASK_ID.zip http://localhost:8080/v1/tasks/$TASK_ID/result

# 6. 查看结果
unzip -l result_$TASK_ID.zip
```

## 故障排除

### 常见问题

1. **FFmpeg 不可用**
   - 确保 FFmpeg 和 FFprobe 已安装并在 PATH 中
   - 运行 `make check-deps` 检查依赖

2. **数据库连接失败**
   - 检查 PostgreSQL 是否运行
   - 验证数据库连接参数
   - 确保数据库已创建并初始化

3. **文件上传失败**
   - 检查文件大小是否超过限制
   - 确保文件格式受支持

4. **任务处理失败**
   - 检查磁盘空间是否充足
   - 查看日志了解具体错误信息

## 性能优化

- 调整 `WORKER_MAX_CONCURRENT` 参数控制并发处理数量
- 根据服务器性能配置 FFmpeg 参数
- 使用 SSD 存储提高 I/O 性能
- 定期清理过期的任务文件

## License

MIT License