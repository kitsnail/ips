package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client K8s客户端封装
type Client struct {
	Clientset kubernetes.Interface
	Config    *rest.Config
	Namespace string
}

// NewClient 创建K8s客户端
// 优先使用in-cluster配置，如果失败则使用kubeconfig
func NewClient(namespace string) (*Client, error) {
	if namespace == "" {
		// 尝试从Pod内读取当前namespace
		if ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
			namespace = string(ns)
		} else {
			namespace = "default"
		}
	}

	// 尝试in-cluster配置
	config, err := rest.InClusterConfig()
	if err != nil {
		// 如果in-cluster失败，尝试使用kubeconfig
		config, err = buildConfigFromKubeconfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	return &Client{
		Clientset: clientset,
		Config:    config,
		Namespace: namespace,
	}, nil
}

// buildConfigFromKubeconfig 从kubeconfig文件构建配置
func buildConfigFromKubeconfig() (*rest.Config, error) {
	// 获取kubeconfig路径
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home dir: %w", err)
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	// 检查文件是否存在
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubeconfig file not found at %s", kubeconfigPath)
	}

	// 构建配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}
