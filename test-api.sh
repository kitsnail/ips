#!/bin/bash

# 镜像预热服务API测试脚本

API_URL="${API_URL:-http://192.168.3.106:8080}"
echo "Testing API at: $API_URL"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试计数
PASSED=0
FAILED=0

# 测试函数
function test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local expected_status=$5

    echo -e "${YELLOW}Testing: $name${NC}"

    # 创建临时文件存储响应
    local temp_file=$(mktemp)

    if [ -n "$data" ]; then
        http_code=$(curl -s -w "%{http_code}" -o "$temp_file" -X $method "$API_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        http_code=$(curl -s -w "%{http_code}" -o "$temp_file" -X $method "$API_URL$endpoint")
    fi

    body=$(cat "$temp_file")
    rm -f "$temp_file"

    if [ "$http_code" == "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAIL${NC} (Expected HTTP $expected_status, got HTTP $http_code)"
        FAILED=$((FAILED + 1))
    fi

    # 格式化并显示响应
    if [ -n "$body" ]; then
        echo -e "${BLUE}Response:${NC}"
        if command -v jq &> /dev/null; then
            echo "$body" | jq . 2>/dev/null || echo "$body"
        else
            echo "$body"
        fi
    else
        echo -e "${BLUE}Response:${NC} (empty)"
    fi
    echo ""

    # 如果是创建任务的测试，保存任务ID
    if [ "$name" == "Create Task" ] && [ "$http_code" == "201" ]; then
        if command -v jq &> /dev/null; then
            TASK_ID=$(echo "$body" | jq -r '.taskId // .task_id // empty' 2>/dev/null)
        else
            # 简单的 JSON 提取（不依赖 jq）
            TASK_ID=$(echo "$body" | grep -o '"taskId":"[^"]*"' | cut -d'"' -f4)
            if [ -z "$TASK_ID" ]; then
                TASK_ID=$(echo "$body" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
            fi
        fi
        if [ -n "$TASK_ID" ]; then
            echo -e "${GREEN}Task ID: $TASK_ID${NC}"
            echo ""
        fi
    fi
}

# 1. 测试健康检查
test_endpoint "Health Check" "GET" "/health" "" "200"

# 2. 测试就绪检查
test_endpoint "Readiness Check" "GET" "/readyz" "" "200"

# 3. 测试创建任务
test_endpoint "Create Task" "POST" "/api/v1/tasks" '{
    "images": ["nginx:latest", "redis:7"],
    "batchSize": 2
}' "201"

# 等待一下让任务有时间处理
if [ -n "$TASK_ID" ]; then
    echo -e "${YELLOW}Waiting 2 seconds for task processing...${NC}"
    sleep 2
    echo ""

    # 4. 测试查询任务详情
    test_endpoint "Get Task" "GET" "/api/v1/tasks/$TASK_ID" "" "200"
fi

# 5. 测试列出所有任务
test_endpoint "List All Tasks" "GET" "/api/v1/tasks" "" "200"

# 6. 测试列出指定状态的任务
test_endpoint "List Running Tasks" "GET" "/api/v1/tasks?status=running&limit=5" "" "200"

# 7. 测试查询不存在的任务
test_endpoint "Get Non-existent Task" "GET" "/api/v1/tasks/non-existent-id" "" "404"

# 8. 测试取消任务
if [ -n "$TASK_ID" ]; then
    # 先检查任务当前状态
    temp_file=$(mktemp)
    http_code=$(curl -s -w "%{http_code}" -o "$temp_file" -X GET "$API_URL/api/v1/tasks/$TASK_ID")
    task_status=$(cat "$temp_file" | jq -r '.status' 2>/dev/null)
    rm -f "$temp_file"

    if [ "$task_status" == "running" ] || [ "$task_status" == "pending" ]; then
        # 任务仍在运行，尝试取消
        test_endpoint "Cancel Task" "DELETE" "/api/v1/tasks/$TASK_ID" "" "200"

        # 验证任务已被取消
        test_endpoint "Verify Cancelled Task" "GET" "/api/v1/tasks/$TASK_ID" "" "200"
    else
        # 任务已完成，测试取消已完成的任务（应该失败）
        echo -e "${YELLOW}Testing: Cancel Completed Task (should fail)${NC}"
        temp_file=$(mktemp)
        http_code=$(curl -s -w "%{http_code}" -o "$temp_file" -X DELETE "$API_URL/api/v1/tasks/$TASK_ID")
        body=$(cat "$temp_file")
        rm -f "$temp_file"

        # 取消已完成的任务应该返回 409 (Conflict) 或 404
        if [ "$http_code" == "409" ] || [ "$http_code" == "404" ]; then
            echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code - Expected failure)"
            PASSED=$((PASSED + 1))
        else
            echo -e "${RED}✗ FAIL${NC} (Expected HTTP 409 or 404, got HTTP $http_code)"
            FAILED=$((FAILED + 1))
        fi

        if [ -n "$body" ]; then
            echo -e "${BLUE}Response:${NC}"
            if command -v jq &> /dev/null; then
                echo "$body" | jq . 2>/dev/null || echo "$body"
            else
                echo "$body"
            fi
        fi
        echo ""

        # 创建新任务用于测试取消
        echo -e "${YELLOW}Creating new task for cancel test...${NC}"
        temp_file=$(mktemp)
        http_code=$(curl -s -w "%{http_code}" -o "$temp_file" -X POST "$API_URL/api/v1/tasks" \
            -H "Content-Type: application/json" \
            -d '{"images": ["alpine:latest"], "batchSize": 1}')
        body=$(cat "$temp_file")
        rm -f "$temp_file"

        if [ "$http_code" == "201" ]; then
            NEW_TASK_ID=$(echo "$body" | jq -r '.taskId' 2>/dev/null)
            echo -e "${GREEN}New Task ID: $NEW_TASK_ID${NC}"
            echo ""

            # 立即尝试取消（在任务执行之前）
            test_endpoint "Cancel New Task" "DELETE" "/api/v1/tasks/$NEW_TASK_ID" "" "200"
        fi
    fi
fi

# 9. 测试创建无效的任务（缺少必需字段）
test_endpoint "Create Invalid Task" "POST" "/api/v1/tasks" '{
    "batchSize": 10
}' "400"

# 10. 测试不支持的 HTTP 方法
test_endpoint "Invalid HTTP Method" "PATCH" "/api/v1/tasks" "" "404"

# 显示测试总结
echo "========================================"
echo -e "${BLUE}Test Summary:${NC}"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"
echo -e "Total: $((PASSED + FAILED))"
echo "========================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
