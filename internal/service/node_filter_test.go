package service

import (
	"context"
	"testing"

	"github.com/kitsnail/ips/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNodeFilter_FilterNodes_AllNodes(t *testing.T) {
	// 创建 fake clientset
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
		Namespace: "default",
	}

	filter := NewNodeFilter(k8sClient)
	ctx := context.Background()

	// 测试获取所有节点（无选择器）
	nodes, err := filter.FilterNodes(ctx, nil)
	if err != nil {
		t.Fatalf("FilterNodes failed: %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(nodes))
	}
}

func TestNodeFilter_FilterNodes_WithSelector(t *testing.T) {
	// 创建 fake clientset
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-1",
				Labels: map[string]string{
					"workload": "compute",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
				Labels: map[string]string{
					"workload": "storage",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	k8sClient := &k8s.Client{
		Clientset: fakeClientset,
		Namespace: "default",
	}

	filter := NewNodeFilter(k8sClient)
	ctx := context.Background()

	// 测试使用选择器过滤
	nodes, err := filter.FilterNodes(ctx, map[string]string{
		"workload": "compute",
	})
	if err != nil {
		t.Fatalf("FilterNodes failed: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}

	if nodes[0] != "node-1" {
		t.Errorf("Expected node-1, got %s", nodes[0])
	}
}
