package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// GetNodes 获取所有节点
func (c *Client) GetNodes(ctx context.Context) ([]corev1.Node, error) {
	nodeList, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return nodeList.Items, nil
}

// GetNodesBySelector 根据标签选择器获取节点
func (c *Client) GetNodesBySelector(ctx context.Context, selector map[string]string) ([]corev1.Node, error) {
	// 构建标签选择器
	labelSelector := labels.SelectorFromSet(selector)

	nodeList, err := c.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes by selector: %w", err)
	}

	return nodeList.Items, nil
}

// IsNodeReady 检查节点是否就绪
func IsNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsNodeSchedulable 检查节点是否可调度
func IsNodeSchedulable(node *corev1.Node) bool {
	// 节点未被标记为不可调度
	return !node.Spec.Unschedulable
}

// FilterReadyNodes 过滤出就绪且可调度的节点
func FilterReadyNodes(nodes []corev1.Node) []string {
	var readyNodes []string

	for _, node := range nodes {
		if IsNodeReady(&node) && IsNodeSchedulable(&node) {
			readyNodes = append(readyNodes, node.Name)
		}
	}

	return readyNodes
}
