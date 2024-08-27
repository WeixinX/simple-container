构建

```bash
chmod +x build.sh
```

启动容器 base1 和 base2

```bash
# 容器 1
./sc run base1 /bin/sh

# 容器 2
./sc run base1 /bin/sh

# 可以增/删/改文件、目录的方式测试容器的隔离性
```

查看并测试 CPU 限制

```bash
# 容器
# 占满分给自己的 CPU 时间
while true; do i=i+1; done

# 宿主机
# 1. 查看限制
cd /sys/fs/cgroup/cpu/simple-container/<container-name>
# cpu.cfs_quota_us/cpu.cfs_period_us 是所能使用的 CPU 时间百分比
cat cpu.cfs_quota_us
cat cpu.cfs_period_us
# tasks 中记录了被限制的 pid
cat tasks

# 2. 查看容器占用情况
# 得到 init-pid
$ pstree -ap <run-pid>
sc,32949 run base1 /bin/sh
  ├─sh,<init-pid>
  ...
# 查看 cpu 使用情况
$ top -p <init-pid>
```

查看并测试内存限制

> 因为 OOM 会导致容器退出，退出时会删除相应 CGroups 目录，\
> 因此无法测试，除非注释掉删除 CGroups 源码

```bash
# 容器
# 分配超过限制的内存大小
mkdir -p /tmp/memory
mount -t tmpfs -o size=512M tmpfs /tmp/memory
dd if=/dev/zero of=/tmp/memory/block

# 宿主机
# 1. 查看限制
cd /sys/fs/cgroup/memory/simple-container/<container-name>
cat memory.limit_in_bytes
cat tasks

# 2. 查看容器 OOM 情况
cat memory.oom_control
```
