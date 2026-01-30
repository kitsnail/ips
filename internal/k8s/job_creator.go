package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JobCreator Job创建器
type JobCreator struct {
	client        *Client
	workerImage   string
	pullerImage   string
	criSocketPath string
}

// NewJobCreator 创建Job创建器
func NewJobCreator(client *Client, workerImage, pullerImage, criSocketPath string) *JobCreator {
	if workerImage == "" {
		workerImage = "registry.k8s.io/pause:3.10"
	}
	if pullerImage == "" {
		pullerImage = "registry.k8s.io/build-containers/crictl:v1.31.0"
	}
	if criSocketPath == "" {
		criSocketPath = "/run/containerd/containerd.sock"
	}

	return &JobCreator{
		client:        client,
		workerImage:   workerImage,
		pullerImage:   pullerImage,
		criSocketPath: criSocketPath,
	}
}

// CreateJob 创建Job来预热镜像
// taskID: 任务ID
// nodeName: 目标节点名称
// images: 要预热的镜像列表
// secretName: 可选的包含凭据的 Secret 名称，为空表示不需要认证
func (j *JobCreator) CreateJob(ctx context.Context, taskID, nodeName string, images []string, secretName string) error {
	jobName := fmt.Sprintf("prewarm-%s-%s", taskID, nodeName)

	// TTL设置：Job完成后15分钟自动清理
	ttl := int32(900)
	backoffLimit := int32(0) // 镜像预热不需要多次重试，失败就记录

	// 构建环境变量列表
	envVars := []corev1.EnvVar{
		{
			Name:  "IMAGES",
			Value: strings.Join(images, ","),
		},
		{
			Name:  "CRI_SOCKET_PATH",
			Value: j.criSocketPath,
		},
	}

	// 如果有 Secret，通过 secretKeyRef 引入 REGISTRY_CREDS 环境变量
	// 这样密码不会在 kubectl describe pod 中以明文显示
	if secretName != "" {
		envVars = append(envVars, corev1.EnvVar{
			Name: "REGISTRY_CREDS",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretName,
					},
					Key: "credentials",
				},
			},
		})
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: j.client.Namespace,
			Labels: map[string]string{
				"app":     "image-prewarm",
				"task-id": taskID,
				"node":    nodeName,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			BackoffLimit:            &backoffLimit,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":     "image-prewarm",
						"task-id": taskID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "puller",
							Image:           j.pullerImage,
							ImagePullPolicy: corev1.PullAlways, // 确保使用最新镜像
							Command:         []string{"/app/apiserver"},
							Args:            []string{"pull"},
							Env:             envVars,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "cri-socket",
									MountPath: j.criSocketPath,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: func(b bool) *bool { return &b }(true),
								RunAsUser:  func(i int64) *int64 { return &i }(0),
								RunAsGroup: func(i int64) *int64 { return &i }(0),
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "cri-socket",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: j.criSocketPath,
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"kubernetes.io/hostname": nodeName, // 指定节点
					},
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // 容忍所有污点，确保能调度到任何节点
						},
					},
				},
			},
		},
	}

	// 创建Job
	_, err := j.client.Clientset.BatchV1().Jobs(j.client.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create job %s: %w", jobName, err)
	}

	return nil
}

// DeleteJob 删除Job
func (j *JobCreator) DeleteJob(ctx context.Context, jobName string) error {
	deletePolicy := metav1.DeletePropagationBackground

	err := j.client.Clientset.BatchV1().Jobs(j.client.Namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	if err != nil {
		return fmt.Errorf("failed to delete job %s: %w", jobName, err)
	}

	return nil
}

// CreateCredsSecret 创建用于存储 crictl 凭据的 Opaque Secret
// taskID: 任务ID
// username: 镜像仓库用户名
// password: 镜像仓库密码
// 返回创建的 Secret 名称
func (j *JobCreator) CreateCredsSecret(ctx context.Context, taskID, username, password string) (string, error) {
	secretName := fmt.Sprintf("registry-creds-%s", taskID)

	// 凭据格式：username:password，供 crictl --creds 参数使用
	credentials := fmt.Sprintf("%s:%s", username, password)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: j.client.Namespace,
			Labels: map[string]string{
				"app":     "image-prewarm",
				"task-id": taskID,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"credentials": []byte(credentials),
		},
	}

	_, err := j.client.Clientset.CoreV1().Secrets(j.client.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create credentials secret %s: %w", secretName, err)
	}

	return secretName, nil
}

// CreateSecret 创建用于私有镜像仓库认证的Secret（dockerconfigjson 格式，已弃用）
// taskID: 任务ID
// registry: 镜像仓库地址（如 harbor.example.com）
// username: 镜像仓库用户名
// password: 镜像仓库密码
func (j *JobCreator) CreateSecret(ctx context.Context, taskID, registry, username, password string) (string, error) {
	secretName := fmt.Sprintf("image-pull-secret-%s", taskID)

	// 构建 .dockerconfigjson 格式的认证信息，使用标准JSON编码以安全处理特殊字符
	authString := fmt.Sprintf("%s:%s", username, password)
	authEncoded := base64.StdEncoding.EncodeToString([]byte(authString))

	dockerConfig := map[string]interface{}{
		"auths": map[string]interface{}{
			registry: map[string]interface{}{
				"username": username,
				"password": password,
				"auth":     authEncoded,
			},
		},
	}

	dockerConfigJSON, err := json.Marshal(dockerConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal docker config: %w", err)
	}

	// 注意：使用 Data 字段时，K8s API 会自动进行 Base64 编码存储
	// 因此这里直接传入原始 JSON 字节数据，不需要手动 Base64 编码

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: j.client.Namespace,
			Labels: map[string]string{
				"app":     "image-prewarm",
				"task-id": taskID,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			".dockerconfigjson": dockerConfigJSON, // 直接使用 JSON 字节数据
		},
	}

	_, err = j.client.Clientset.CoreV1().Secrets(j.client.Namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create secret %s: %v", secretName, err.Error())
	}

	return secretName, nil
}

// DeleteSecret 删除用于镜像仓库认证的Secret
// secretName: Secret名称
func (j *JobCreator) DeleteSecret(ctx context.Context, secretName string) error {
	deletePolicy := metav1.DeletePropagationBackground

	err := j.client.Clientset.CoreV1().Secrets(j.client.Namespace).Delete(ctx, secretName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})

	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %v", secretName, err.Error())
	}

	return nil
}

// ListJobsByTaskID 列出指定任务的所有Job
func (j *JobCreator) ListJobsByTaskID(ctx context.Context, taskID string) ([]batchv1.Job, error) {
	jobList, err := j.client.Clientset.BatchV1().Jobs(j.client.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("task-id=%s", taskID),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list jobs for task %s: %w", taskID, err)
	}

	return jobList.Items, nil
}

// GetK8sClient 获取K8s客户端（用于Watch等高级功能）
func (j *JobCreator) GetK8sClient() *Client {
	return j.client
}
