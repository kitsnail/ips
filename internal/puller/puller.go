package puller

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Run 运行拉取逻辑
func Run(images []string, criSocketPath string) {
	results := make(map[string]int)

	// 读取凭据（格式：username:password）
	registryCreds := os.Getenv("REGISTRY_CREDS")

	fmt.Printf("Starting pre-warm for %d images using socket %s\n", len(images), criSocketPath)
	if registryCreds != "" {
		fmt.Println("Using registry credentials for authentication")
	}

	for _, img := range images {
		fmt.Printf("Pulling %s...\n", img)

		// 构造 crictl 命令
		// 注意：我们需要通过环境变量或参数指定 socket
		var cmd *exec.Cmd
		if registryCreds != "" {
			// 使用 --creds 参数进行认证
			cmd = exec.Command("crictl", "--image-endpoint", "unix://"+criSocketPath, "pull", "--creds", registryCreds, img)
		} else {
			cmd = exec.Command("crictl", "--image-endpoint", "unix://"+criSocketPath, "pull", img)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to pull %s: %v\nOutput: %s\n", img, err, string(output))
			results[img] = 0
		} else {
			fmt.Printf("Successfully pulled %s\n", img)
			results[img] = 1
		}
	}

	// 写入 termination log
	data, _ := json.Marshal(results)
	err := os.WriteFile("/dev/termination-log", data, 0644)
	if err != nil {
		fmt.Printf("Failed to write termination log: %v\n", err)
		// 如果无法写入，至少打印出来
		fmt.Printf("FINAL_RESULT: %s\n", string(data))
	}

	// 如果有失败的，以非零状态退出？
	// 其实没必要，因为我们已经把结果写到了 termination log，
	// 让 Job 始终 Succeeded 可能更方便处理（由 StatusTracker 判断）。
	// 但通常如果有失败，退出码非零更符合 K8s 习惯。
	// 这里我们选择让 Job 成功，因为拉取逻辑已经执行完毕。
}
