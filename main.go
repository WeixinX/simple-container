package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"simple-container/core"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func main() {
	switch os.Args[1] {
	case "pull":
		// 模拟拉取镜像
		// tar -zxf ubuntu-base-16.04.6-amd64.tar.gz -C rootfs
		baseImage := "ubuntu-base-16.04.6-amd64.tar.gz"
		must(os.MkdirAll(core.ImagePath, 0700))
		must(exec.Command("tar", "-zxf", baseImage, "-C", core.ImagePath).Run())
		fmt.Printf("[pull] tar -zxf %s -C %s\n", baseImage, core.ImagePath)
		return

	case "run":
		fmt.Printf("[run] pid: %d, ppid: %d\n", os.Getpid(), os.Getppid())

		// fork 出自身作为子进程执行 init 分支
		initCmd, err := os.Readlink("/proc/self/exe")
		assertNoError(err)

		os.Args[1] = "init"
		cmd := exec.Command(initCmd, os.Args[1:]...)
		setCmdEnv(cmd)

		containerName := os.Args[2]

		// 设置 Namespaces 视图隔离
		core.SetNamespaceIsolation(cmd)

		// 准备容器 RootFS 文件系统
		defer func() {
			err := core.DelRootFS(containerName)
			assertNoError(err)
		}()
		must(core.GetRootFS(containerName))

		// 开启伪 tty，运行子进程，即 /proc/self/exe init ...args
		ptmx := startWithPTY(cmd)
		defer ptmx.Close()

		// 设置容器 CGroups 资源限制
		cpuQuotaUs := "50000" // 50000/100000 = 50%
		memLimitBytes := "256m"
		defer func() {
			err := core.DelCgroupsPath(containerName)
			assertNoError(err)
		}()
		must(core.SetCgroups(cmd.Process.Pid, containerName, cpuQuotaUs, memLimitBytes))

		cmd.Wait()
		return

	case "init":
		time.Sleep(100 * time.Millisecond)

		fmt.Printf("[init] pid: %d, ppid: %d\n", os.Getpid(), os.Getppid())

		containerName := os.Args[2]
		cmd := os.Args[3]

		// 设置容器 RootFS 文件系统
		must(core.SetRootFS(containerName))

		// 启动新程序，替换当前程序，PID 不变
		fmt.Println("will exec cmd:", cmd)
		must(syscall.Exec(cmd, os.Args[3:], os.Environ()))
		return

	case "test":
		cmd := exec.Command("/bin/sh")
		setCmdEnv(cmd)
		core.SetNamespaceIsolation(cmd)

		// 开启伪 tty 执行 cmd
		ptmx := startWithPTY(cmd)
		defer ptmx.Close()

		cmd.Wait()
		fmt.Println("bye")
		return

	default:
		fmt.Println("invalid cmd")
	}
}

func setCmdEnv(cmd *exec.Cmd) {
	cmd.Env = append(os.Environ(), "PS1=-[container]- # ")
}

func startWithPTY(cmd *exec.Cmd) *os.File {
	ptmx, err := pty.Start(cmd)
	assertNoError(err)
	go func() { _, _ = io.Copy(os.Stdout, ptmx) }()
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()

	// 禁止 pty 回显，如标准输入键入 ls，标准输出不会再出现 ls
	_, err = term.MakeRaw(int(ptmx.Fd()))
	assertNoError(err)
	return ptmx
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func assertNoError(err error) {
	must(err)
}
