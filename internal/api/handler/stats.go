package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kitsnail/ips/internal/k8s"
)

// StatsHandler 统计数据处理器
type StatsHandler struct {
	k8sClient *k8s.Client
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(k8sClient *k8s.Client) *StatsHandler {
	return &StatsHandler{
		k8sClient: k8sClient,
	}
}

// GetStats 获取统计信息（包括节点覆盖）
func (h *StatsHandler) GetStats(c *gin.Context) {
	// 获取所有节点
	allNodes, err := h.k8sClient.GetNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get nodes",
		})
		return
	}

	// 获取就绪节点
	readyNodes := k8s.FilterReadyNodes(allNodes)

	c.JSON(http.StatusOK, gin.H{
		"nodes": gin.H{
			"total":    len(allNodes),
			"ready":    len(readyNodes),
			"coverage": len(readyNodes),
		},
	})
}
