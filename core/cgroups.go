package core

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

const (
	cgroupsPath = "/sys/fs/cgroup"
	projectName = "simple-container"
)

func SetCgroups(pid int, groupName string, cpuQuotaUs string, memLimitBytes string) error {
	cpuPath := path.Join(cgroupsPath, "cpu", projectName, groupName)
	memPath := path.Join(cgroupsPath, "memory", projectName, groupName)

	// 创建容器的控制目录
	if err := os.MkdirAll(cpuPath, 0700); err != nil {
		return fmt.Errorf("failed to create cgroup path: %v", err)
	}
	if err := os.MkdirAll(memPath, 0700); err != nil {
		return fmt.Errorf("failed to create cgroup path: %v", err)
	}

	// 设置 CPU
	if err := os.WriteFile(path.Join(cpuPath, "cpu.cfs_quota_us"), []byte(cpuQuotaUs), 0700); err != nil {
		return fmt.Errorf("failed to write cpu quota us: %v", err)
	}
	if err := os.WriteFile(path.Join(cpuPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to write cpu tasks: %v", err)
	}
	fmt.Println("[cgroups] set:", cpuPath)

	// 设置内存
	if err := os.WriteFile(path.Join(memPath, "memory.limit_in_bytes"), []byte(memLimitBytes), 0700); err != nil {
		return fmt.Errorf("failed to write memory limit bytes: %v", err)
	}
	if err := os.WriteFile(path.Join(memPath, "tasks"), []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to write memory tasks: %v", err)
	}
	fmt.Println("[cgroups] set:", memPath)

	return nil
}

func DelCgroupsPath(groupName string) error {
	cpuPath := path.Join(cgroupsPath, "cpu", projectName, groupName)
	memPath := path.Join(cgroupsPath, "memory", projectName, groupName)

	if err := os.RemoveAll(cpuPath); err != nil {
		return fmt.Errorf("failed to delete cpu cgroup: %v", err)
	}
	fmt.Println("[cgroups] delete:", cpuPath)

	if err := os.RemoveAll(memPath); err != nil {
		return fmt.Errorf("failed to delete mem cgroup: %v", err)
	}
	fmt.Println("[cgroups] delete:", memPath)

	return nil
}
