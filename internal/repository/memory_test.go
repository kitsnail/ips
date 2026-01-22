package repository

import (
	"context"
	"testing"
	"time"

	"github.com/kitsnail/ips/pkg/models"
)

func TestMemoryRepository_CreateTask(t *testing.T) {
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
	err := repo.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 验证任务已创建
	retrieved, err := repo.GetTask(ctx, "test-task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
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
	err := repo.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("First CreateTask failed: %v", err)
	}

	// 第二次创建相同ID应该失败
	err = repo.CreateTask(ctx, task)
	if err != ErrTaskAlreadyExists {
		t.Errorf("Expected ErrTaskAlreadyExists, got %v", err)
	}
}

func TestMemoryRepository_GetTask(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// 测试获取不存在的任务
	_, err := repo.GetTask(ctx, "non-existent")
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
	repo.CreateTask(ctx, task)

	// 测试获取存在的任务
	retrieved, err := repo.GetTask(ctx, "test-task-3")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.ID != task.ID {
		t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestMemoryRepository_UpdateTask(t *testing.T) {
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
	repo.CreateTask(ctx, task)

	// 更新任务状态
	task.Status = models.TaskRunning
	err := repo.UpdateTask(ctx, task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// 验证更新成功
	retrieved, _ := repo.GetTask(ctx, "test-task-4")
	if retrieved.Status != models.TaskRunning {
		t.Errorf("Expected Status %s, got %s", models.TaskRunning, retrieved.Status)
	}
}

func TestMemoryRepository_DeleteTask(t *testing.T) {
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
	repo.CreateTask(ctx, task)

	// 删除任务
	err := repo.DeleteTask(ctx, "test-task-5")
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// 验证任务已删除
	_, err = repo.GetTask(ctx, "test-task-5")
	if err != ErrTaskNotFound {
		t.Errorf("Expected ErrTaskNotFound after delete, got %v", err)
	}
}

func TestMemoryRepository_ListTasks(t *testing.T) {
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
		repo.CreateTask(ctx, task)
	}

	// Test ListTasks with pagination
	allTasks, total, err := repo.ListTasks(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}
	if total != 3 {
		t.Errorf("Expected total 3, got %d", total)
	}
}
