package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
)

type SecretHandler struct {
	secretRepo repository.SecretRegistryRepository
}

func NewSecretHandler(secretRepo repository.SecretRegistryRepository) *SecretHandler {
	return &SecretHandler{
		secretRepo: secretRepo,
	}
}

func (h *SecretHandler) ListSecrets(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * pageSize

	secrets, total, err := h.secretRepo.ListSecrets(c.Request.Context(), offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"secrets":  secrets,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *SecretHandler) CreateSecret(c *gin.Context) {
	var req models.CreateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret := &models.RegistrySecret{
		Name:     req.Name,
		Registry: req.Registry,
		Username: req.Username,
		Password: req.Password,
	}

	if err := h.secretRepo.CreateSecret(c.Request.Context(), secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, secret)
}

func (h *SecretHandler) GetSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID"})
		return
	}

	secret, err := h.secretRepo.GetSecret(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, secret)
}

func (h *SecretHandler) UpdateSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID"})
		return
	}

	var req models.UpdateSecretRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secret, err := h.secretRepo.GetSecretCredentials(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrTaskNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Secret not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	secret.Name = req.Name
	secret.Registry = req.Registry
	secret.Username = req.Username
	if req.Password != "" {
		secret.Password = req.Password
	}

	if err := h.secretRepo.UpdateSecret(c.Request.Context(), secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, secret)
}

func (h *SecretHandler) DeleteSecret(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid secret ID"})
		return
	}

	if err := h.secretRepo.DeleteSecret(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
