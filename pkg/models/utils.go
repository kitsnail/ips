package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateTaskID 生成任务ID
// 格式: [prefix-]YYYYMMDD-HHMMSS-随机字符
// prefix 为空时默认为 "task"
func GenerateTaskID(prefix string) string {
	now := time.Now()
	timestamp := now.Format("20060102-150405")

	// 生成4字节随机数
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	randomStr := hex.EncodeToString(randomBytes)

	if prefix == "" {
		prefix = "task"
	}

	return fmt.Sprintf("%s-%s-%s", prefix, timestamp, randomStr)
}
