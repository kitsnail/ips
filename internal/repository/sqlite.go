package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kitsnail/ips/pkg/models"
	_ "modernc.org/sqlite"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(dsn string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %v", err)
	}

	repo := &SQLiteRepository{db: db}
	if err := repo.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %v", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) initSchema() error {
	// 任务表
	taskSchema := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		images TEXT,
		batch_size INTEGER,
		priority INTEGER,
		max_retries INTEGER,
		retry_delay INTEGER,
		retry_strategy TEXT,
		webhook_url TEXT,
		status TEXT,
		progress TEXT,
		node_statuses TEXT,
		failed_nodes TEXT,
		error_message TEXT,
		created_at DATETIME,
		started_at DATETIME,
		finished_at DATETIME
	);`

	// 用户表
	userSchema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT,
		role TEXT,
		created_at DATETIME,
		updated_at DATETIME
	);`

	// API 令牌表
	tokenSchema := `
	CREATE TABLE IF NOT EXISTS api_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		name TEXT,
		token TEXT UNIQUE,
		created_at DATETIME,
		expires_at DATETIME,
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`

	for _, schema := range []string{taskSchema, userSchema, tokenSchema} {
		if _, err := r.db.Exec(schema); err != nil {
			return err
		}
	}
	return nil
}

// TaskRepository Implementation

func (r *SQLiteRepository) CreateTask(ctx context.Context, task *models.Task) error {
	imagesJSON, _ := json.Marshal(task.Images)
	progressJSON, _ := json.Marshal(task.Progress)
	nodeStatsJSON, _ := json.Marshal(task.NodeStatuses)
	failedNodesJSON, _ := json.Marshal(task.FailedNodes)

	query := `INSERT INTO tasks (id, images, batch_size, priority, max_retries, retry_delay, retry_strategy, 
		webhook_url, status, progress, node_statuses, failed_nodes, error_message, created_at, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, imagesJSON, task.BatchSize, task.Priority, task.MaxRetries, task.RetryDelay, task.RetryStrategy,
		task.WebhookURL, task.Status, progressJSON, nodeStatsJSON, failedNodesJSON, task.ErrorMessage,
		task.CreatedAt, task.StartedAt, task.FinishedAt)
	return err
}

func (r *SQLiteRepository) UpdateTask(ctx context.Context, task *models.Task) error {
	progressJSON, _ := json.Marshal(task.Progress)
	nodeStatsJSON, _ := json.Marshal(task.NodeStatuses)
	failedNodesJSON, _ := json.Marshal(task.FailedNodes)

	query := `UPDATE tasks SET status=?, progress=?, node_statuses=?, failed_nodes=?, error_message=?, 
		started_at=?, finished_at=? WHERE id=?`

	_, err := r.db.ExecContext(ctx, query,
		task.Status, progressJSON, nodeStatsJSON, failedNodesJSON, task.ErrorMessage,
		task.StartedAt, task.FinishedAt, task.ID)
	return err
}

func (r *SQLiteRepository) GetTask(ctx context.Context, id string) (*models.Task, error) {
	query := `SELECT id, images, batch_size, priority, max_retries, retry_delay, retry_strategy, 
		webhook_url, status, progress, node_statuses, failed_nodes, error_message, created_at, started_at, finished_at 
		FROM tasks WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	var task models.Task
	var imagesJSON, progressJSON, nodeStatsJSON, failedNodesJSON []byte

	err := row.Scan(&task.ID, &imagesJSON, &task.BatchSize, &task.Priority, &task.MaxRetries, &task.RetryDelay, &task.RetryStrategy,
		&task.WebhookURL, &task.Status, &progressJSON, &nodeStatsJSON, &failedNodesJSON, &task.ErrorMessage,
		&task.CreatedAt, &task.StartedAt, &task.FinishedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(imagesJSON, &task.Images)
	json.Unmarshal(progressJSON, &task.Progress)
	json.Unmarshal(nodeStatsJSON, &task.NodeStatuses)
	json.Unmarshal(failedNodesJSON, &task.FailedNodes)

	return &task, nil
}

func (r *SQLiteRepository) ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int, error) {
	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tasks").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, images, batch_size, priority, max_retries, retry_delay, retry_strategy, 
		webhook_url, status, progress, node_statuses, failed_nodes, error_message, created_at, started_at, finished_at 
		FROM tasks ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tasks []*models.Task
	for rows.Next() {
		var task models.Task
		var imagesJSON, progressJSON, nodeStatsJSON, failedNodesJSON []byte

		err := rows.Scan(&task.ID, &imagesJSON, &task.BatchSize, &task.Priority, &task.MaxRetries, &task.RetryDelay, &task.RetryStrategy,
			&task.WebhookURL, &task.Status, &progressJSON, &nodeStatsJSON, &failedNodesJSON, &task.ErrorMessage,
			&task.CreatedAt, &task.StartedAt, &task.FinishedAt)
		if err != nil {
			return nil, 0, err
		}

		json.Unmarshal(imagesJSON, &task.Images)
		json.Unmarshal(progressJSON, &task.Progress)
		json.Unmarshal(nodeStatsJSON, &task.NodeStatuses)
		json.Unmarshal(failedNodesJSON, &task.FailedNodes)

		tasks = append(tasks, &task)
	}
	return tasks, total, nil
}

func (r *SQLiteRepository) DeleteTask(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = ?", id)
	return err
}

// UserRepository Implementation

func (r *SQLiteRepository) CreateUser(ctx context.Context, user *models.User) error {
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	query := `INSERT INTO users (username, password, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, user.Username, user.Password, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	user.ID = id
	return nil
}

func (r *SQLiteRepository) GetUser(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?", id)
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return &user, err
}

func (r *SQLiteRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?", username)
	var user models.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil // Return nil, nil if not found
	}
	return &user, err
}

func (r *SQLiteRepository) ListUsers(ctx context.Context) ([]*models.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, username, role, created_at, updated_at FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, nil
}

func (r *SQLiteRepository) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	query := `UPDATE users SET password=?, role=?, updated_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query, user.Password, user.Role, user.UpdatedAt, user.ID)
	return err
}

func (r *SQLiteRepository) DeleteUser(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

func (r *SQLiteRepository) CreateToken(ctx context.Context, token *models.APIToken) error {
	token.CreatedAt = time.Now()
	query := `INSERT INTO api_tokens (user_id, name, token, created_at, expires_at) VALUES (?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, token.UserID, token.Name, token.Token, token.CreatedAt, token.ExpiresAt)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	token.ID = id
	return nil
}

func (r *SQLiteRepository) GetToken(ctx context.Context, tokenStr string) (*models.APIToken, error) {
	row := r.db.QueryRowContext(ctx, "SELECT id, user_id, name, token, created_at, expires_at FROM api_tokens WHERE token = ?", tokenStr)
	var token models.APIToken
	err := row.Scan(&token.ID, &token.UserID, &token.Name, &token.Token, &token.CreatedAt, &token.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	return &token, err
}

func (r *SQLiteRepository) ListTokens(ctx context.Context, userID int64) ([]*models.APIToken, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, user_id, name, token, created_at, expires_at FROM api_tokens WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*models.APIToken
	for rows.Next() {
		var token models.APIToken
		if err := rows.Scan(&token.ID, &token.UserID, &token.Name, &token.Token, &token.CreatedAt, &token.ExpiresAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}
	return tokens, nil
}

func (r *SQLiteRepository) DeleteToken(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM api_tokens WHERE id = ?", id)
	return err
}
