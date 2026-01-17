package repository

import (
	"context"
	"testing"
	"time"

	"github.com/kitsnail/ips/pkg/models"
)

func TestMemoryRepository_Create(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	task := &models.Task{
		ID:        "test-task-1",
		Status:    models.TaskPending,
		Images:    []string{"nginx:latest"},
		BatchSize: 10,
		CreatedAt: time.Now(),
	}

	// 测试创建任务
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 验证任务已创建
	retrieved, err := repo.Get(ctx, "test-task-1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}

	if retrieved.Status != task.Status {
		t.Errorf("Expected Status %s, got %s", task.Status, retrieved.Status)
	}
}

func TestMemoryRepository_CreateDuplicate(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	task := &models.Task{
		ID:        "test-task-2",
		Status:    models.TaskPending,
		Images:    []string{"redis:7"},
		BatchSize: 5,
		CreatedAt: time.Now(),
	}

	// 第一次创建应该成功
	err := repo.Create(ctx, task)
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	// 第二次创建相同ID应该失败
	err = repo.Create(ctx, task)
	if err != ErrTaskAlreadyExists {
		t.Errorf("Expected ErrTaskAlreadyExists, got %v", err)
	}
}

func TestMemoryRepository_Get(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// 测试获取不存在的任务
	_, err := repo.Get(ctx, "non-existent")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound, got %v", err)
	}

	// 创建任务
	task := &models.Task{
		ID:        "test-task-3",
		Status:    models.TaskRunning,
		Images:    []string{"mysql:8"},
		BatchSize: 20,
		CreatedAt: time.Now(),
	}
	repo.Create(ctx, task)

	// 测试获取存在的任务
	retrieved, err := repo.Get(ctx, "test-task-3")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestMemoryRepository_Update(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// 创建任务
	task := &models.Task{
		ID:        "test-task-4",
		Status:    models.TaskPending,
		Images:    []string{"postgres:15"},
		BatchSize: 15,
		CreatedAt: time.Now(),
	}
	repo.Create(ctx, task)

	// 更新任务状态
	task.Status = models.TaskRunning
	err := repo.Update(ctx, task)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证更新成功
	retrieved, _ := repo.Get(ctx, "test-task-4")
	if retrieved.Status != models.TaskRunning {
		t.Errorf("Expected Status %s, got %s", models.TaskRunning, retrieved.Status)
	}
}

func TestMemoryRepository_Delete(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// 创建任务
	task := &models.Task{
		ID:        "test-task-5",
		Status:    models.TaskCompleted,
		Images:    []string{"alpine:latest"},
		BatchSize: 5,
		CreatedAt: time.Now(),
	}
	repo.Create(ctx, task)

	// 删除任务
	err := repo.Delete(ctx, "test-task-5")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证任务已删除
	_, err = repo.Get(ctx, "test-task-5")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound after delete, got %v", err)
	}
}

func TestMemoryRepository_List(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// 创建多个任务
	tasks := []*models.Task{
		{
			ID:        "task-1",
			Status:    models.TaskPending,
			Images:    []string{"nginx:latest"},
			BatchSize: 10,
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			ID:        "task-2",
			Status:    models.TaskRunning,
			Images:    []string{"redis:7"},
			BatchSize: 5,
			CreatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:        "task-3",
			Status:    models.TaskCompleted,
			Images:    []string{"mysql:8"},
			BatchSize: 15,
			CreatedAt: time.Now().Add(-1 * time.Hour),
		},
	}

	for _, task := range tasks {
		repo.Create(ctx, task)
	}

	// 测试列出所有任务
	allTasks, total, err := repo.List(ctx, models.TaskFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}

	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}

	// 测试按状态过滤
	status := models.TaskRunning
	filteredTasks, total, err := repo.List(ctx, models.TaskFilter{
		Status: &status,
	})
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected total 1, got %d", total)
	}

	if len(filteredTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(filteredTasks))
	}

	if filteredTasks[0].Status != models.TaskRunning {
		t.Errorf("Expected status Running, got %s", filteredTasks[0].Status)
	}
}
