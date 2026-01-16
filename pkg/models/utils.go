package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateTaskID 生成任务ID
// 格式: task-YYYYMMDD-HHMMSS-随机字符
func GenerateTaskID() string {
	now := time.Now()
	timestamp := now.Format("20060102-150405")

	// 生成4字节随机数
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	return fmt.Sprintf("task-%s-%s", timestamp, randomStr)
}
