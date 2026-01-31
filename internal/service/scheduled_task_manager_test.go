package service

import (
	"context"
	"testing"
	"time"

	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScheduledTaskManager(t *testing.T) (*ScheduledTaskManager, *repository.SQLiteRepository) {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:")
	require.NoError(t, err)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	taskManager := NewTaskManager(
		repo,
		repo,
		nil,
		nil,
		nil,
		logger,
	)

	scheduledTaskManager := NewScheduledTaskManager(repo, repo, taskManager, logger)

	return scheduledTaskManager, repo
}

func TestScheduledTaskManager_AddTask(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "test-scheduled-task-1",
		Name:     "Test Scheduled Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images:    []string{"nginx:latest"},
			BatchSize: 10,
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	err = manager.AddTask(task)
	assert.NoError(t, err)
	assert.Contains(t, manager.cronEntries, task.ID)

	savedTask, err := repo.GetScheduledTask(context.Background(), task.ID)
	assert.NoError(t, err)
	assert.Equal(t, task.Name, savedTask.Name)
	assert.Equal(t, task.CronExpr, savedTask.CronExpr)
}

func TestScheduledTaskManager_AddTask_Disabled(t *testing.T) {
	manager, _ := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "test-scheduled-task-2",
		Name:     "Disabled Task",
		CronExpr: "0 0 * * *",
		Enabled:  false,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := manager.AddTask(task)
	assert.NoError(t, err)
	assert.NotContains(t, manager.cronEntries, task.ID)
}

func TestScheduledTaskManager_AddTask_Duplicate(t *testing.T) {
	manager, _ := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "test-scheduled-task-3",
		Name:     "Duplicate Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	// Add task first time
	err := manager.AddTask(task)
	assert.NoError(t, err)

	// Try to add same task again
	err = manager.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already scheduled")
}

func TestScheduledTaskManager_RemoveTask(t *testing.T) {
	manager, _ := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "test-scheduled-task-4",
		Name:     "Remove Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	// Add task
	err := manager.AddTask(task)
	assert.NoError(t, err)
	assert.Contains(t, manager.cronEntries, task.ID)

	// Remove task
	err = manager.RemoveTask(task.ID)
	assert.NoError(t, err)
	assert.NotContains(t, manager.cronEntries, task.ID)

	// Removing non-existent task should not error
	err = manager.RemoveTask("non-existent-task")
	assert.NoError(t, err)
}

func TestScheduledTaskManager_EnableDisableTask(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	// Create a disabled task
	task := &models.ScheduledTask{
		ID:       "test-scheduled-task-5",
		Name:     "Enable/Disable Task",
		CronExpr: "0 0 * * *",
		Enabled:  false,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	// Save to database
	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	// Enable task
	err = manager.EnableTask(task.ID)
	assert.NoError(t, err)
	assert.Contains(t, manager.cronEntries, task.ID)

	savedTask, _ := repo.GetScheduledTask(context.Background(), task.ID)
	assert.True(t, savedTask.Enabled)

	// Disable task
	err = manager.DisableTask(task.ID)
	assert.NoError(t, err)
	assert.NotContains(t, manager.cronEntries, task.ID)

	savedTask, _ = repo.GetScheduledTask(context.Background(), task.ID)
	assert.False(t, savedTask.Enabled)
}

func TestScheduledTaskManager_ListScheduledTasks(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	// Create multiple tasks
	tasks := []*models.ScheduledTask{
		{
			ID:       "task-1",
			Name:     "Task 1",
			CronExpr: "0 0 * * *",
			Enabled:  true,
			TaskConfig: models.TaskConfig{
				Images: []string{"nginx:latest"},
			},
			CreatedBy: "test-user",
		},
		{
			ID:       "task-2",
			Name:     "Task 2",
			CronExpr: "0 6 * * *",
			Enabled:  false,
			TaskConfig: models.TaskConfig{
				Images: []string{"alpine:latest"},
			},
			CreatedBy: "test-user",
		},
		{
			ID:       "task-3",
			Name:     "Task 3",
			CronExpr: "0 12 * * *",
			Enabled:  true,
			TaskConfig: models.TaskConfig{
				Images: []string{"redis:latest"},
			},
			CreatedBy: "test-user",
		},
	}

	for _, task := range tasks {
		err := repo.CreateScheduledTask(context.Background(), task)
		require.NoError(t, err)
	}

	// List all tasks
	list, total, err := manager.ListScheduledTasks(context.Background(), 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 3)

	// List with pagination
	list, total, err = manager.ListScheduledTasks(context.Background(), 0, 2)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 2)

	list, total, err = manager.ListScheduledTasks(context.Background(), 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, list, 1)
}

func TestScheduledTaskManager_GetScheduledTask(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "task-get",
		Name:     "Get Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	// Get existing task
	foundTask, err := manager.GetScheduledTask(context.Background(), "task-get")
	assert.NoError(t, err)
	assert.Equal(t, task.Name, foundTask.Name)
	assert.Equal(t, task.CronExpr, foundTask.CronExpr)

	// Get non-existent task
	_, err = manager.GetScheduledTask(context.Background(), "non-existent")
	assert.Error(t, err)
}

func TestScheduledTaskManager_DeleteScheduledTask(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "task-delete",
		Name:     "Delete Task",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	// Add to scheduler first
	err = manager.AddTask(task)
	require.NoError(t, err)

	// Remove from scheduler first
	err = manager.RemoveTask(task.ID)
	require.NoError(t, err)

	// Delete from database
	err = manager.DeleteScheduledTask(context.Background(), "task-delete")
	assert.NoError(t, err)

	// Verify task is deleted from database
	_, err = repo.GetScheduledTask(context.Background(), "task-delete")
	assert.Error(t, err)
}

func TestScheduledTaskManager_ListEnabledScheduledTasks(t *testing.T) {
	_, repo := setupScheduledTaskManager(t)

	// Create mixed enabled/disabled tasks
	tasks := []*models.ScheduledTask{
		{
			ID:       "enabled-1",
			Name:     "Enabled 1",
			CronExpr: "0 0 * * *",
			Enabled:  true,
			TaskConfig: models.TaskConfig{
				Images: []string{"nginx:latest"},
			},
			CreatedBy: "test-user",
		},
		{
			ID:       "disabled-1",
			Name:     "Disabled 1",
			CronExpr: "0 6 * * *",
			Enabled:  false,
			TaskConfig: models.TaskConfig{
				Images: []string{"alpine:latest"},
			},
			CreatedBy: "test-user",
		},
		{
			ID:       "enabled-2",
			Name:     "Enabled 2",
			CronExpr: "0 12 * * *",
			Enabled:  true,
			TaskConfig: models.TaskConfig{
				Images: []string{"redis:latest"},
			},
			CreatedBy: "test-user",
		},
	}

	for _, task := range tasks {
		err := repo.CreateScheduledTask(context.Background(), task)
		require.NoError(t, err)
	}

	// List enabled tasks
	enabledTasks, err := repo.ListEnabledScheduledTasks(context.Background())
	assert.NoError(t, err)
	assert.Len(t, enabledTasks, 2)

	for _, task := range enabledTasks {
		assert.True(t, task.Enabled)
	}
}

func TestScheduledTaskManager_InvalidCronExpression(t *testing.T) {
	manager, _ := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "invalid-cron",
		Name:     "Invalid Cron",
		CronExpr: "invalid-cron-expression",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := manager.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestScheduledTaskManager_OverlapPolicySkip(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "overlap-skip",
		Name:     "Overlap Skip",
		CronExpr: "* * * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		OverlapPolicy:  models.OverlapPolicySkip,
		TimeoutSeconds: 0,
		CreatedBy:      "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	err = manager.AddTask(task)
	require.NoError(t, err)

	// Mark task as running
	manager.executingTasks[task.ID] = true

	// Try to execute while previous is running
	_, err = manager.executeTask(task.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skipped")
}

func TestScheduledTaskManager_Start(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	// Create an enabled task
	task := &models.ScheduledTask{
		ID:       "start-test",
		Name:     "Start Test",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	assert.NoError(t, err)

	// Verify task was loaded into scheduler
	assert.Contains(t, manager.cronEntries, task.ID)

	// Stop manager
	manager.Stop()
}

func TestScheduledTaskManager_CleanupOldExecutions(t *testing.T) {
	manager, repo := setupScheduledTaskManager(t)

	// Create old execution
	oldTime := time.Now().Add(-100 * 24 * time.Hour)
	oldExecution := &models.ScheduledExecution{
		ScheduledTaskID: "task-1",
		TaskID:          "old-task",
		Status:          models.ScheduledExecutionSuccess,
		StartedAt:       oldTime,
		FinishedAt:      &oldTime,
	}

	err := repo.CreateExecution(context.Background(), oldExecution)
	require.NoError(t, err)

	// Create recent execution
	recentTime := time.Now()
	recentExecution := &models.ScheduledExecution{
		ScheduledTaskID: "task-1",
		TaskID:          "recent-task",
		Status:          models.ScheduledExecutionSuccess,
		StartedAt:       recentTime,
		FinishedAt:      &recentTime,
	}

	err = repo.CreateExecution(context.Background(), recentExecution)
	require.NoError(t, err)

	// Cleanup old executions
	deleted, err := manager.CleanupOldExecutions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only recent execution remains
	executions, _, err := repo.ListExecutions(context.Background(), "", 0, 10)
	assert.NoError(t, err)
	assert.Len(t, executions, 1)
	assert.Equal(t, "recent-task", executions[0].TaskID)
}

func TestScheduledTaskManager_UpdateScheduledTask(t *testing.T) {
	_, repo := setupScheduledTaskManager(t)

	task := &models.ScheduledTask{
		ID:       "update-test",
		Name:     "Original Name",
		CronExpr: "0 0 * * *",
		Enabled:  true,
		TaskConfig: models.TaskConfig{
			Images: []string{"nginx:latest"},
		},
		CreatedBy: "test-user",
	}

	err := repo.CreateScheduledTask(context.Background(), task)
	require.NoError(t, err)

	// Update task
	task.Name = "Updated Name"
	task.CronExpr = "0 6 * * *"
	err = repo.UpdateScheduledTask(context.Background(), task)
	assert.NoError(t, err)

	// Verify update
	updatedTask, err := repo.GetScheduledTask(context.Background(), task.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedTask.Name)
	assert.Equal(t, "0 6 * * *", updatedTask.CronExpr)
}
