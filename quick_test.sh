#!/bin/bash

# 快速 API 测试脚本
# 使用方法: ./quick_test.sh

BASE_URL="http://localhost:8080"

echo "🚀 快速 API 测试"
echo "==================="

# 1. 健康检查
echo "1. 健康检查..."
curl -s "$BASE_URL/health"
echo -e "\n"

# 2. 创建测试文件并上传
echo "2. 创建任务..."
echo "Test video content for API testing - $(date)" > test_video.mp4

# 上传并获取 task_id
RESPONSE=$(curl -s -X POST \
  -F "file=@test_video.mp4" \
  -F "segment_sec=5" \
  -F "frame_interval_sec=2" \
  "$BASE_URL/v1/tasks")

echo "创建响应: $RESPONSE"

# 提取 task_id (简单方法)
TASK_ID=$(echo "$RESPONSE" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
echo "任务 ID: $TASK_ID"

if [ -n "$TASK_ID" ]; then
    # 3. 查询状态
    echo -e "\n3. 查询任务状态..."
    curl -s "$BASE_URL/v1/tasks/$TASK_ID"
    echo -e "\n"
    
    # 4. 尝试下载（可能还没准备好）
    echo "4. 尝试下载结果..."
    curl -s -w "HTTP状态: %{http_code}\n" \
         -o "result.zip" \
         "$BASE_URL/v1/tasks/$TASK_ID/result"
    
    if [ -f "result.zip" ]; then
        ls -la result.zip
    fi
fi

# 清理
rm -f test_video.mp4 result.zip

echo -e "\n✅ 快速测试完成！"