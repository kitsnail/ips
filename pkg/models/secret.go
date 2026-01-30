package models

import "time"

// RegistrySecret represents a private registry credential
type RegistrySecret struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`     // Human-readable name
	Registry  string    `json:"registry"` // Registry address (e.g., harbor.example.com)
	Username  string    `json:"username"` // Registry username
	Password  string    `json:"-"`        // Password (never returned in JSON)
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateSecretRequest request to create a registry secret
type CreateSecretRequest struct {
	Name     string `json:"name" binding:"required"`
	Registry string `json:"registry" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpdateSecretRequest request to update a registry secret
type UpdateSecretRequest struct {
	Name     string `json:"name" binding:"required"`
	Registry string `json:"registry" binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password"` // Optional when updating
}

// SecretListItem represents a secret in list view (without password)
type SecretListItem struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Registry  string    `json:"registry"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
