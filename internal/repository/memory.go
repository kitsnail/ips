package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/kitsnail/ips/pkg/models"
)

// MemoryRepository 内存存储实现
type MemoryRepository struct {
	mu             sync.RWMutex
	tasks          map[string]*models.Task
	scheduledTasks map[string]*models.ScheduledTask
	libraryImages  map[int64]*models.LibraryImage
	secrets        map[int64]*models.RegistrySecret
	nextLibraryID  int64
	nextSecretID   int64
}

// NewMemoryRepository 创建内存存储
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tasks:          make(map[string]*models.Task),
		scheduledTasks: make(map[string]*models.ScheduledTask),
		libraryImages:  make(map[int64]*models.LibraryImage),
		secrets:        make(map[int64]*models.RegistrySecret),
		nextLibraryID:  1,
		nextSecretID:   1,
	}
}

// CreateTask 创建任务
func (r *MemoryRepository) CreateTask(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}

	r.tasks[task.ID] = task
	return nil
}

// GetTask 获取任务
func (r *MemoryRepository) GetTask(ctx context.Context, id string) (*models.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.tasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return task, nil
}

// ListTasks 列出任务
func (r *MemoryRepository) ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allTasks []*models.Task
	for _, task := range r.tasks {
		allTasks = append(allTasks, task)
	}

	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})

	total := len(allTasks)

	// Apply pagination
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}

	if offset >= total {
		return []*models.Task{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return allTasks[offset:end], total, nil
}

// UpdateTask 更新任务
func (r *MemoryRepository) UpdateTask(ctx context.Context, task *models.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[task.ID]; !exists {
		return ErrTaskNotFound
	}

	r.tasks[task.ID] = task
	return nil
}

// DeleteTask 删除任务
func (r *MemoryRepository) DeleteTask(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[id]; !exists {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}

// Dummy User methods to satisfy interface

func (r *MemoryRepository) CreateUser(ctx context.Context, user *models.User) error { return nil }
func (r *MemoryRepository) GetUser(ctx context.Context, id int64) (*models.User, error) {
	return nil, nil
}
func (r *MemoryRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	return nil, nil
}
func (r *MemoryRepository) ListUsers(ctx context.Context) ([]*models.User, error)         { return nil, nil }
func (r *MemoryRepository) UpdateUser(ctx context.Context, user *models.User) error       { return nil }
func (r *MemoryRepository) DeleteUser(ctx context.Context, id int64) error                { return nil }
func (r *MemoryRepository) CreateToken(ctx context.Context, token *models.APIToken) error { return nil }
func (r *MemoryRepository) GetToken(ctx context.Context, tokenStr string) (*models.APIToken, error) {
	return nil, nil
}
func (r *MemoryRepository) ListTokens(ctx context.Context, userID int64) ([]*models.APIToken, error) {
	return nil, nil
}
func (r *MemoryRepository) DeleteToken(ctx context.Context, id int64) error { return nil }

func (r *MemoryRepository) CreateSecret(ctx context.Context, secret *models.RegistrySecret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	secret.ID = r.nextSecretID
	r.nextSecretID++
	secret.CreatedAt = time.Now()
	secret.UpdatedAt = secret.CreatedAt
	r.secrets[secret.ID] = secret
	return nil
}
func (r *MemoryRepository) GetSecret(ctx context.Context, id int64) (*models.RegistrySecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	secret, exists := r.secrets[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return secret, nil
}
func (r *MemoryRepository) GetSecretByName(ctx context.Context, name string) (*models.RegistrySecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, secret := range r.secrets {
		if secret.Name == name {
			return secret, nil
		}
	}
	return nil, ErrTaskNotFound
}
func (r *MemoryRepository) ListScheduledTasks(ctx context.Context, offset, limit int) ([]*models.ScheduledTask, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allTasks []*models.ScheduledTask
	for _, task := range r.scheduledTasks {
		allTasks = append(allTasks, task)
	}

	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].CreatedAt.After(allTasks[j].CreatedAt)
	})

	total := len(allTasks)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}

	if offset >= total {
		return []*models.ScheduledTask{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return allTasks[offset:end], total, nil
}
func (r *MemoryRepository) ListSecrets(ctx context.Context, offset, limit int) ([]*models.SecretListItem, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allSecrets []*models.SecretListItem
	for _, secret := range r.secrets {
		allSecrets = append(allSecrets, &models.SecretListItem{
			ID:        secret.ID,
			Name:      secret.Name,
			Registry:  secret.Registry,
			Username:  secret.Username,
			CreatedAt: secret.CreatedAt,
			UpdatedAt: secret.UpdatedAt,
		})
	}

	sort.Slice(allSecrets, func(i, j int) bool {
		return allSecrets[i].CreatedAt.After(allSecrets[j].CreatedAt)
	})

	total := len(allSecrets)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}

	if offset >= total {
		return []*models.SecretListItem{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return allSecrets[offset:end], total, nil
}
func (r *MemoryRepository) UpdateSecret(ctx context.Context, secret *models.RegistrySecret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.secrets[secret.ID]; !exists {
		return ErrTaskNotFound
	}
	secret.UpdatedAt = time.Now()
	r.secrets[secret.ID] = secret
	return nil
}
func (r *MemoryRepository) DeleteSecret(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.secrets[id]; !exists {
		return ErrTaskNotFound
	}
	delete(r.secrets, id)
	return nil
}
func (r *MemoryRepository) GetSecretCredentials(ctx context.Context, id int64) (*models.RegistrySecret, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	secret, exists := r.secrets[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return secret, nil
}

func (r *MemoryRepository) CreateLibraryImage(ctx context.Context, image *models.LibraryImage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	image.ID = r.nextLibraryID
	r.nextLibraryID++
	image.CreatedAt = time.Now()
	r.libraryImages[image.ID] = image
	return nil
}
func (r *MemoryRepository) GetLibraryImage(ctx context.Context, id int64) (*models.LibraryImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	image, exists := r.libraryImages[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return image, nil
}
func (r *MemoryRepository) ListLibraryImages(ctx context.Context, offset, limit int) ([]*models.LibraryImage, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var allImages []*models.LibraryImage
	for _, image := range r.libraryImages {
		allImages = append(allImages, image)
	}

	sort.Slice(allImages, func(i, j int) bool {
		return allImages[i].CreatedAt.After(allImages[j].CreatedAt)
	})

	total := len(allImages)

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}

	if offset >= total {
		return []*models.LibraryImage{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return allImages[offset:end], total, nil
}
func (r *MemoryRepository) DeleteLibraryImage(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.libraryImages[id]; !exists {
		return ErrTaskNotFound
	}
	delete(r.libraryImages, id)
	return nil
}

func (r *MemoryRepository) CreateScheduledTask(ctx context.Context, task *models.ScheduledTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.scheduledTasks[task.ID]; exists {
		return ErrTaskAlreadyExists
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt
	r.scheduledTasks[task.ID] = task
	return nil
}

func (r *MemoryRepository) GetScheduledTask(ctx context.Context, id string) (*models.ScheduledTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, exists := r.scheduledTasks[id]
	if !exists {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

func (r *MemoryRepository) UpdateScheduledTask(ctx context.Context, task *models.ScheduledTask) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.scheduledTasks[task.ID]; !exists {
		return ErrTaskNotFound
	}
	task.UpdatedAt = time.Now()
	r.scheduledTasks[task.ID] = task
	return nil
}

func (r *MemoryRepository) DeleteScheduledTask(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.scheduledTasks[id]; !exists {
		return ErrTaskNotFound
	}
	delete(r.scheduledTasks, id)
	return nil
}
