package service

import (
	"context"
	"fmt"

	"github.com/kitsnail/ips/internal/k8s"
)

// NodeFilter 节点过滤服务
type NodeFilter struct {
	k8sClient *k8s.Client
}

// NewNodeFilter 创建节点过滤服务
func NewNodeFilter(k8sClient *k8s.Client) *NodeFilter {
	return &NodeFilter{
		k8sClient: k8sClient,
	}
}

// FilterNodes 根据选择器过滤节点
// 返回符合条件且就绪的节点名称列表
func (n *NodeFilter) FilterNodes(ctx context.Context, selector map[string]string) ([]string, error) {
	var nodes []string
	var err error

	// 如果有选择器，使用选择器过滤
	if len(selector) > 0 {
		nodeList, err := n.k8sClient.GetNodesBySelector(ctx, selector)
		if err != nil {
			return nil, fmt.Errorf("failed to get nodes by selector: %w", err)
		}
		// 过滤出就绪且可调度的节点
		nodes = k8s.FilterReadyNodes(nodeList)
	} else {
		// 否则获取所有节点
		nodeList, err := n.k8sClient.GetNodes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get all nodes: %w", err)
		}
		// 过滤出就绪且可调度的节点
		nodes = k8s.FilterReadyNodes(nodeList)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no ready nodes found")
	}

	return nodes, err
}
