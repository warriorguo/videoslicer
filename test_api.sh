#!/bin/bash

# Video Slicer API 测试脚本
# 使用方法: ./test_api.sh [服务地址]

set -e

# 配置
BASE_URL="${1:-http://localhost:8080}"
TEST_FILE="test_video.mp4"

echo "🎬 Video Slicer API 测试脚本"
echo "服务地址: $BASE_URL"
echo "==============================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 辅助函数
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

# 检查 jq 是否安装
if ! command -v jq &> /dev/null; then
    log_warning "jq 未安装，JSON 响应将不会格式化"
    JQ_AVAILABLE=false
else
    JQ_AVAILABLE=true
fi

# 1. 健康检查
test_health() {
    log_info "1. 执行健康检查..."
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$BASE_URL/health" 2>/dev/null || echo "HTTPSTATUS:000")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS\:.*//g')
    
    if [ "$http_code" = "200" ] && [ "$body" = "OK" ]; then
        log_success "服务健康检查通过"
    else
        log_error "服务健康检查失败 (HTTP: $http_code, Body: $body)"
        log_error "请确保服务正在运行在 $BASE_URL"
        exit 1
    fi
}

# 2. 创建测试文件
create_test_file() {
    log_info "2. 创建测试视频文件..."
    
    # 创建一个更大的测试文件，模拟真实视频
    cat > "$TEST_FILE" << EOF
# Mock Video File for API Testing
# This file simulates a video for testing purposes
# File size: $(date +%s) bytes
# Created: $(date)
# 
# In a real scenario, this would be an actual video file
# such as MP4, MOV, AVI, etc.
#
# Test data:
$(for i in {1..100}; do echo "Frame $i data: $(openssl rand -hex 32)"; done)
EOF
    
    file_size=$(stat -f%z "$TEST_FILE" 2>/dev/null || stat -c%s "$TEST_FILE" 2>/dev/null || echo "unknown")
    log_success "测试文件已创建: $TEST_FILE (大小: $file_size 字节)"
}

# 3. 上传视频并创建任务
create_task() {
    log_info "3. 上传视频并创建任务..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -F "file=@$TEST_FILE" \
        -F "segment_sec=5" \
        -F "frame_interval_sec=2" \
        -F "frame_format=jpg" \
        -F "zip_format=zip" \
        "$BASE_URL/v1/tasks" 2>/dev/null || echo "HTTPSTATUS:000")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS\:.*//g')
    
    if [ "$http_code" = "200" ]; then
        log_success "任务创建成功"
        
        if [ "$JQ_AVAILABLE" = true ]; then
            echo "$body" | jq '.'
            TASK_ID=$(echo "$body" | jq -r '.task_id')
        else
            echo "$body"
            TASK_ID=$(echo "$body" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
        fi
        
        if [ -z "$TASK_ID" ] || [ "$TASK_ID" = "null" ]; then
            log_error "无法提取任务 ID"
            exit 1
        fi
        
        log_success "任务 ID: $TASK_ID"
    else
        log_error "任务创建失败 (HTTP: $http_code)"
        echo "$body"
        exit 1
    fi
}

# 4. 监控任务状态
monitor_task() {
    log_info "4. 监控任务处理进度..."
    
    max_attempts=60  # 最多等待 2 分钟
    attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            "$BASE_URL/v1/tasks/$TASK_ID" 2>/dev/null || echo "HTTPSTATUS:000")
        
        http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
        body=$(echo "$response" | sed -e 's/HTTPSTATUS\:.*//g')
        
        if [ "$http_code" = "200" ]; then
            if [ "$JQ_AVAILABLE" = true ]; then
                status=$(echo "$body" | jq -r '.status')
                stage=$(echo "$body" | jq -r '.progress.stage')
                percent=$(echo "$body" | jq -r '.progress.percent')
            else
                status=$(echo "$body" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
                stage=$(echo "$body" | grep -o '"stage":"[^"]*"' | cut -d'"' -f4)
                percent=$(echo "$body" | grep -o '"percent":[0-9.]*' | cut -d':' -f2)
            fi
            
            echo "[$attempt/$max_attempts] 状态: $status | 阶段: $stage | 进度: $percent%"
            
            case "$status" in
                "succeeded")
                    log_success "任务处理成功！"
                    return 0
                    ;;
                "failed")
                    log_error "任务处理失败"
                    if [ "$JQ_AVAILABLE" = true ]; then
                        echo "$body" | jq '.error'
                    else
                        echo "$body"
                    fi
                    return 1
                    ;;
                "queued"|"processing")
                    # 继续监控
                    ;;
                *)
                    log_warning "未知状态: $status"
                    ;;
            esac
        else
            log_error "查询任务状态失败 (HTTP: $http_code)"
            return 1
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_warning "任务监控超时，请手动检查任务状态"
    return 1
}

# 5. 下载处理结果
download_result() {
    log_info "5. 尝试下载处理结果..."
    
    result_file="result_${TASK_ID}.zip"
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -o "$result_file" \
        "$BASE_URL/v1/tasks/$TASK_ID/result" 2>/dev/null || echo "HTTPSTATUS:000")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" = "200" ]; then
        if [ -f "$result_file" ]; then
            file_size=$(stat -f%z "$result_file" 2>/dev/null || stat -c%s "$result_file" 2>/dev/null || echo "unknown")
            log_success "结果已下载: $result_file (大小: $file_size 字节)"
            
            # 尝试查看压缩包内容
            if command -v unzip &> /dev/null; then
                log_info "压缩包内容预览:"
                unzip -l "$result_file" | head -20
            fi
        else
            log_error "结果文件未找到"
        fi
    elif [ "$http_code" = "404" ]; then
        log_warning "结果尚未准备就绪 (任务可能仍在处理中)"
    else
        log_error "下载结果失败 (HTTP: $http_code)"
    fi
}

# 6. 测试错误情况
test_error_cases() {
    log_info "6. 测试错误情况..."
    
    # 测试不存在的任务
    log_info "   测试查询不存在的任务..."
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        "$BASE_URL/v1/tasks/nonexistent_task" 2>/dev/null || echo "HTTPSTATUS:000")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    if [ "$http_code" = "404" ]; then
        log_success "   正确返回 404 (任务不存在)"
    else
        log_warning "   期望 404，但收到 $http_code"
    fi
    
    # 测试无文件上传
    log_info "   测试无文件上传..."
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{}' \
        "$BASE_URL/v1/tasks" 2>/dev/null || echo "HTTPSTATUS:000")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    if [ "$http_code" = "400" ]; then
        log_success "   正确返回 400 (无文件)"
    else
        log_warning "   期望 400，但收到 $http_code"
    fi
}

# 7. 性能测试
performance_test() {
    log_info "7. 简单性能测试..."
    
    # 创建一个稍大的测试文件
    perf_test_file="perf_test.mp4"
    dd if=/dev/zero of="$perf_test_file" bs=1024 count=1024 2>/dev/null
    
    start_time=$(date +%s)
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -F "file=@$perf_test_file" \
        "$BASE_URL/v1/tasks" 2>/dev/null || echo "HTTPSTATUS:000")
    
    end_time=$(date +%s)
    duration=$((end_time - start_time))
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" = "200" ]; then
        log_success "   性能测试完成，耗时: ${duration}秒"
    else
        log_warning "   性能测试失败 (HTTP: $http_code)"
    fi
    
    rm -f "$perf_test_file"
}

# 清理函数
cleanup() {
    log_info "清理测试文件..."
    rm -f "$TEST_FILE" result_*.zip
    log_success "清理完成"
}

# 主流程
main() {
    echo "开始时间: $(date)"
    echo
    
    # 设置陷阱，确保清理
    trap cleanup EXIT
    
    test_health
    echo
    
    create_test_file
    echo
    
    create_task
    echo
    
    if monitor_task; then
        echo
        download_result
    fi
    
    echo
    test_error_cases
    echo
    
    performance_test
    echo
    
    log_success "所有测试完成！"
    echo "结束时间: $(date)"
}

# 显示使用帮助
show_help() {
    echo "用法: $0 [OPTIONS] [服务地址]"
    echo
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -q, --quiet    静默模式（减少输出）"
    echo
    echo "示例:"
    echo "  $0                                    # 使用默认地址 http://localhost:8080"
    echo "  $0 http://localhost:9090              # 使用自定义地址"
    echo "  $0 https://your-server.com/api       # 使用远程服务器"
    echo
    echo "注意事项:"
    echo "  - 确保服务已启动并可访问"
    echo "  - 需要 curl 命令"
    echo "  - 建议安装 jq 以获得更好的 JSON 显示效果"
}

# 解析命令行参数
case "$1" in
    -h|--help)
        show_help
        exit 0
        ;;
    -q|--quiet)
        exec > /dev/null 2>&1
        shift
        ;;
esac

# 运行主流程
main