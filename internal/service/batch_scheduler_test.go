package service

import (
	"testing"
)

func TestBatchScheduler_SplitBatches(t *testing.T) {
	scheduler := &BatchScheduler{}

	tests := []struct {
		name      string
		nodes     []string
		batchSize int
		expected  int // 期望的批次数
	}{
		{
			name:      "正常分批",
			nodes:     []string{"node-1", "node-2", "node-3", "node-4", "node-5"},
			batchSize: 2,
			expected:  3, // [2, 2, 1]
		},
		{
			name:      "批次大小等于节点数",
			nodes:     []string{"node-1", "node-2", "node-3"},
			batchSize: 3,
			expected:  1,
		},
		{
			name:      "批次大小大于节点数",
			nodes:     []string{"node-1", "node-2"},
			batchSize: 10,
			expected:  1,
		},
		{
			name:      "批次大小为1",
			nodes:     []string{"node-1", "node-2", "node-3"},
			batchSize: 1,
			expected:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := scheduler.splitBatches(tt.nodes, tt.batchSize)

			if len(batches) != tt.expected {
				t.Errorf("Expected %d batches, got %d", tt.expected, len(batches))
			}

			// 验证所有节点都被包含
			totalNodes := 0
			for _, batch := range batches {
				totalNodes += len(batch)
			}

			if totalNodes != len(tt.nodes) {
				t.Errorf("Expected total %d nodes, got %d", len(tt.nodes), totalNodes)
			}
		})
	}
}

func TestBatchScheduler_CalculateBatches(t *testing.T) {
	scheduler := &BatchScheduler{}

	tests := []struct {
		name       string
		totalNodes int
		batchSize  int
		expected   int
		expectErr  bool
	}{
		{
			name:       "正常计算",
			totalNodes: 100,
			batchSize:  10,
			expected:   10,
			expectErr:  false,
		},
		{
			name:       "有余数",
			totalNodes: 105,
			batchSize:  10,
			expected:   11,
			expectErr:  false,
		},
		{
			name:       "批次大小为0",
			totalNodes: 100,
			batchSize:  0,
			expected:   0,
			expectErr:  true,
		},
		{
			name:       "节点数为0",
			totalNodes: 0,
			batchSize:  10,
			expected:   0,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scheduler.CalculateBatches(tt.totalNodes, tt.batchSize)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d batches, got %d", tt.expected, result)
				}
			}
		})
	}
}
