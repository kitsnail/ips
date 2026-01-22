package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/repository"
	"github.com/kitsnail/ips/pkg/models"
)

// LibraryHandler 镜像库处理器
type LibraryHandler struct {
	repo repository.LibraryRepository
}

// NewLibraryHandler 创建镜像库处理器
func NewLibraryHandler(repo repository.LibraryRepository) *LibraryHandler {
	return &LibraryHandler{repo: repo}
}

// ListImages 列出镜像库
func (h *LibraryHandler) ListImages(c *gin.Context) {
	// Parse pagination params
	limit := 10
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	images, total, err := h.repo.ListImages(c.Request.Context(), offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list library images", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// SaveImage 保存镜像到库
func (h *LibraryHandler) SaveImage(c *gin.Context) {
	var img models.LibraryImage
	if err := c.ShouldBindJSON(&img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if img.Image == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image URL is required"})
		return
	}

	if err := h.repo.SaveImage(c.Request.Context(), &img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, img)
}

// DeleteImage 从库中删除镜像
func (h *LibraryHandler) DeleteImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.repo.DeleteImage(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Image ID %d deleted", id)})
}
