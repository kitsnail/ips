package k8s

import (
	"context"
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
func (j *JobCreator) CreateJob(ctx context.Context, taskID, nodeName string, images []string) error {
	jobName := fmt.Sprintf("prewarm-%s-%s", taskID, nodeName)

	// TTL设置：Job完成后15分钟自动清理
	ttl := int32(900)
	backoffLimit := int32(0) // 镜像预热不需要多次重试，失败就记录

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
							Name:    "puller",
							Image:   j.pullerImage,
							Command: []string{"/app/apiserver"},
							Args:    []string{"pull"},
							Env: []corev1.EnvVar{
								{
									Name:  "IMAGES",
									Value: strings.Join(images, ","),
								},
								{
									Name:  "CRI_SOCKET_PATH",
									Value: j.criSocketPath,
								},
							},
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
