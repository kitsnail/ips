package service

import (
	"container/heap"
	"sync"

	"github.com/kitsnail/ips/pkg/models"
)

// TaskItem 任务队列项
type TaskItem struct {
	Task  *models.Task
	index int // heap 中的索引
}

// PriorityQueue 任务优先级队列
// 实现 heap.Interface
type PriorityQueue struct {
	items []*TaskItem
	mu    sync.RWMutex
}

// NewPriorityQueue 创建优先级队列
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*TaskItem, 0),
	}
	heap.Init(pq)
	return pq
}

// Len 返回队列长度
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// Less 比较两个任务的优先级
// 优先级高的排在前面；如果优先级相同，按创建时间排序（先进先出）
func (pq *PriorityQueue) Less(i, j int) bool {
	// 优先级高的排前面
	if pq.items[i].Task.Priority != pq.items[j].Task.Priority {
		return pq.items[i].Task.Priority > pq.items[j].Task.Priority
	}
	// 优先级相同时，按创建时间排序（早创建的排前面）
	return pq.items[i].Task.CreatedAt.Before(pq.items[j].Task.CreatedAt)
}

// Swap 交换两个元素
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

// Push 添加元素到队列
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*TaskItem)
	item.index = n
	pq.items = append(pq.items, item)
}

// Pop 从队列中取出最高优先级的元素
func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.index = -1 // 标记为已移除
	pq.items = old[0 : n-1]
	return item
}

// Enqueue 线程安全地入队
func (pq *PriorityQueue) Enqueue(task *models.Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &TaskItem{
		Task: task,
	}
	heap.Push(pq, item)
}

// Dequeue 线程安全地出队
func (pq *PriorityQueue) Dequeue() *models.Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.Len() == 0 {
		return nil
	}

	item := heap.Pop(pq).(*TaskItem)
	return item.Task
}

// Peek 查看队首元素但不移除
func (pq *PriorityQueue) Peek() *models.Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.Len() == 0 {
		return nil
	}

	return pq.items[0].Task
}

// IsEmpty 检查队列是否为空
func (pq *PriorityQueue) IsEmpty() bool {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.Len() == 0
}

// Size 返回队列大小
func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.Len()
}
