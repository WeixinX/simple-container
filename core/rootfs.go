package core

import (
	"fmt"
	"os"
	"syscall"
)

const (
	mntPath        = "runtime/mnt"
	workLayerPath  = "runtime/work"
	writeLayerPath = "runtime/write"
	ImagePath      = "rootfs"
	mntOldPath     = ".old"
)

func rootfs(containerName string) string {
	return fmt.Sprintf("%s/%s", mntPath, containerName)
}

func workLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", workLayerPath, containerName)
}

func writeLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", writeLayerPath, containerName)
}

func mntOldLayer(containerName string) string {
	return fmt.Sprintf("%s/%s", rootfs(containerName), mntOldPath)
}

func GetRootFS(containerName string) error {
	// 新建容器 rootfs 挂载目录
	if err := os.MkdirAll(rootfs(containerName), 0700); err != nil {
		return fmt.Errorf("failed to mkdir mntlayer: %v", err)
	}
	// 新建容器工作层目录
	if err := os.MkdirAll(workLayer(containerName), 0700); err != nil {
		return fmt.Errorf("failed to mkdir work layer: %v", err)
	}
	// 新建容器写层目录
	if err := os.MkdirAll(writeLayer(containerName), 0700); err != nil {
		return fmt.Errorf("failed to mkdir write layer: %v", err)
	}

	// 使用 overlay 联合文件系统，将
	// - 容器镜像作为 lowerdir，
	// - 容器写层目录作为 upperdir，
	// - 容器工作目录作为 workdir
	// 将上述目录依次堆叠形成完整的容器 rootfs

	// mount -t overlay overlay -o \
	// lowerdir=rootfs,\
	// upperdir=runtime/write/<container-name>,\
	// workdir=runtime/work/<container-name> \
	// runtime/mnt/<container-name>
	dir := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
		ImagePath, writeLayer(containerName), workLayer(containerName))
	if err := syscall.Mount("overlay", rootfs(containerName), "overlay", 0, dir); err != nil {
		return fmt.Errorf("faield to mount overlay: %s", err)
	}

	fmt.Println("[rootfs] get:", rootfs(containerName))
	return nil
}

func SetRootFS(containerName string) error {
	// 挂载文件系统 rootfs
	// 在没有重新挂载时，新的 MNT Namespace 下看到的目录和主机的是一样的
	// 需重新挂载的目录有二：
	// - 1. MNT Namespace 的根目录
	// - 2. 进程自身的根目录

	// chroot 替换方式
	// 只改变了进程自身的根目录，并没有修改 MNT Namespace 的根目录
	// syscall.Chroot(mntLayer(containerName))
	// syscall.Chdir("/")

	// pivot_root 替换方式
	// 声明 MNT Namespace 从根目录开始以及其子目录（MS_REC）都是私有模式（MS_PRIVATE）挂载
	// 宿主机是 ubuntu，默认的 init 进程由 systemd 进程代替，而 systemd 进程的默认为共享挂载方式，
	// 共享模式挂载会让所有命名空间都能看到所有的挂载目录，后续会导致 pivot_root 调用失败，
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to reclare rootfs private: %v", err)
	}
	// 挂载堆叠好的 rootfs
	if err := syscall.Mount(rootfs(containerName), rootfs(containerName), "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to mount rootfs in new mnt space: %v", err)
	}
	if err := os.MkdirAll(mntOldLayer(containerName), 0700); err != nil {
		return fmt.Errorf("failed to mkdir mnt old layer: %v", err)
	}
	// 将原来的根目录放入 old 中
	// 切换根目录为堆叠好的 rootfs，即新的根目录
	if err := syscall.PivotRoot(rootfs(containerName), mntOldLayer(containerName)); err != nil {
		return fmt.Errorf("failed to pivot root: %v", err)
	}
	// 将当前进程工作目录切换至根目录
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir /: %v", err)
	}

	// 挂载 proc 目录，为了使用 top 命令
	if err := syscall.Mount("proc", "/proc", "proc",
		syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, ""); err != nil {
		panic("failed to mount proc: " + err.Error())
	}

	// 卸载 old 目录
	old := "/" + mntOldPath
	if err := syscall.Unmount(old, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to umount %s: %v", old, err)
	}
	if err := os.RemoveAll(old); err != nil {
		return fmt.Errorf("failed to remove %s: %v", old, err)
	}

	fmt.Println("[rootfs] set:", rootfs(containerName))
	return nil
}

func DelRootFS(containerName string) error {
	if err := syscall.Unmount(rootfs(containerName), 0); err != nil {
		return fmt.Errorf("failed to umount %s: %v", rootfs(containerName), err)
	}

	if err := os.RemoveAll(rootfs(containerName)); err != nil {
		return fmt.Errorf("failed to remove rootfs: %s, %v", rootfs(containerName), err)
	}
	if err := os.RemoveAll(workLayer(containerName)); err != nil {
		return fmt.Errorf("failed to remove work layer: %s, %v", workLayer(containerName), err)
	}
	if err := os.RemoveAll(writeLayer(containerName)); err != nil {
		return fmt.Errorf("failed to remove write layer: %s, %v", writeLayer(containerName), err)
	}

	fmt.Println("[rootfs] delete: ", rootfs(containerName))
	return nil
}
