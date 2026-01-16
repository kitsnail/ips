#!/bin/bash

# 镜像预热服务API测试脚本

API_URL="${API_URL:-http://localhost:8080}"
echo "Testing API at: $API_URL"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
function test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local expected_status=$5

    echo -e "${YELLOW}Testing: $name${NC}"

    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method "$API_URL$endpoint")
    fi

    body=$(echo "$response" | head -n -1)
    status=$(echo "$response" | tail -n 1)

    if [ "$status" == "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $status)"
        echo "Response: $(echo $body | jq . 2>/dev/null || echo $body)"
    else
        echo -e "${RED}✗ FAIL${NC} (Expected HTTP $expected_status, got HTTP $status)"
        echo "Response: $(echo $body | jq . 2>/dev/null || echo $body)"
    fi
    echo ""

    # 如果是创建任务的测试，保存任务ID
    if [ "$name" == "Create Task" ] && [ "$status" == "201" ]; then
        TASK_ID=$(echo $body | jq -r .taskId)
        echo "Task ID: $TASK_ID"
    fi
}

# 1. 测试健康检查
test_endpoint "Health Check" "GET" "/healthz" "" "200"

# 2. 测试就绪检查
test_endpoint "Readiness Check" "GET" "/readyz" "" "200"

# 3. 测试创建任务
test_endpoint "Create Task" "POST" "/api/v1/tasks" '{
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 2
}' "201"

# 等待一下让任务有时间处理
if [ -n "$TASK_ID" ]; then
    echo "Waiting 2 seconds..."
    sleep 2

    # 4. 测试查询任务详情
    test_endpoint "Get Task" "GET" "/api/v1/tasks/$TASK_ID" "" "200"
fi

# 5. 测试列出所有任务
test_endpoint "List Tasks" "GET" "/api/v1/tasks" "" "200"

# 6. 测试列出指定状态的任务
test_endpoint "List Running Tasks" "GET" "/api/v1/tasks?status=running&limit=5" "" "200"

# 7. 测试查询不存在的任务
test_endpoint "Get Non-existent Task" "GET" "/api/v1/tasks/non-existent-id" "" "404"

# 8. 测试取消任务
if [ -n "$TASK_ID" ]; then
    test_endpoint "Cancel Task" "DELETE" "/api/v1/tasks/$TASK_ID" "" "200"

    # 验证任务状态
    test_endpoint "Verify Cancelled Task" "GET" "/api/v1/tasks/$TASK_ID" "" "200"
fi

# 9. 测试创建无效的任务（缺少必需字段）
test_endpoint "Create Invalid Task" "POST" "/api/v1/tasks" '{
    "batchSize": 10
}' "400"

echo -e "${GREEN}All tests completed!${NC}"
