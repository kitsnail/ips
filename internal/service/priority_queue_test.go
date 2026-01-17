package service

import (
	"testing"
	"time"

	"github.com/kitsnail/ips/pkg/models"
)

func TestPriorityQueue_Enqueue_Dequeue(t *testing.T) {
	pq := NewPriorityQueue()

	// 创建不同优先级的任务
	task1 := &models.Task{ID: "task-1", Priority: 5, CreatedAt: time.Now()}
	task2 := &models.Task{ID: "task-2", Priority: 8, CreatedAt: time.Now()}
	task3 := &models.Task{ID: "task-3", Priority: 3, CreatedAt: time.Now()}

	// 入队
	pq.Enqueue(task1)
	pq.Enqueue(task2)
	pq.Enqueue(task3)

	// 验证队列大小
	if pq.Size() != 3 {
		t.Errorf("Expected queue size 3, got %d", pq.Size())
	}

	// 出队应该按优先级顺序：task2(8) -> task1(5) -> task3(3)
	dequeued1 := pq.Dequeue()
	if dequeued1.ID != "task-2" {
		t.Errorf("Expected task-2 (priority 8), got %s", dequeued1.ID)
	}

	dequeued2 := pq.Dequeue()
	if dequeued2.ID != "task-1" {
		t.Errorf("Expected task-1 (priority 5), got %s", dequeued2.ID)
	}

	dequeued3 := pq.Dequeue()
	if dequeued3.ID != "task-3" {
		t.Errorf("Expected task-3 (priority 3), got %s", dequeued3.ID)
	}

	// 队列应该为空
	if !pq.IsEmpty() {
		t.Error("Expected queue to be empty")
	}
}

func TestPriorityQueue_SamePriority_FIFO(t *testing.T) {
	pq := NewPriorityQueue()

	// 创建相同优先级的任务
	now := time.Now()
	task1 := &models.Task{ID: "task-1", Priority: 5, CreatedAt: now}
	task2 := &models.Task{ID: "task-2", Priority: 5, CreatedAt: now.Add(1 * time.Second)}
	task3 := &models.Task{ID: "task-3", Priority: 5, CreatedAt: now.Add(2 * time.Second)}

	pq.Enqueue(task1)
	pq.Enqueue(task2)
	pq.Enqueue(task3)

	// 相同优先级应该按FIFO顺序
	dequeued1 := pq.Dequeue()
	if dequeued1.ID != "task-1" {
		t.Errorf("Expected task-1 (earliest), got %s", dequeued1.ID)
	}

	dequeued2 := pq.Dequeue()
	if dequeued2.ID != "task-2" {
		t.Errorf("Expected task-2, got %s", dequeued2.ID)
	}

	dequeued3 := pq.Dequeue()
	if dequeued3.ID != "task-3" {
		t.Errorf("Expected task-3, got %s", dequeued3.ID)
	}
}

func TestPriorityQueue_Peek(t *testing.T) {
	pq := NewPriorityQueue()

	task1 := &models.Task{ID: "task-1", Priority: 5, CreatedAt: time.Now()}
	task2 := &models.Task{ID: "task-2", Priority: 8, CreatedAt: time.Now()}

	pq.Enqueue(task1)
	pq.Enqueue(task2)

	// Peek 应该返回最高优先级但不移除
	peeked := pq.Peek()
	if peeked.ID != "task-2" {
		t.Errorf("Expected task-2 (priority 8), got %s", peeked.ID)
	}

	// 队列大小不变
	if pq.Size() != 2 {
		t.Errorf("Expected queue size 2 after peek, got %d", pq.Size())
	}
}

func TestPriorityQueue_DequeueEmpty(t *testing.T) {
	pq := NewPriorityQueue()

	// 从空队列出队应该返回 nil
	dequeued := pq.Dequeue()
	if dequeued != nil {
		t.Error("Expected nil when dequeuing from empty queue")
	}
}

func TestPriorityQueue_IsEmpty(t *testing.T) {
	pq := NewPriorityQueue()

	if !pq.IsEmpty() {
		t.Error("New queue should be empty")
	}

	task := &models.Task{ID: "task-1", Priority: 5, CreatedAt: time.Now()}
	pq.Enqueue(task)

	if pq.IsEmpty() {
		t.Error("Queue should not be empty after enqueue")
	}

	pq.Dequeue()

	if !pq.IsEmpty() {
		t.Error("Queue should be empty after dequeue")
	}
}
