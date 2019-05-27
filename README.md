
# 客户端

- 单文件，不同的平台根据 runtime.GOOS 和 runtime.GOARCH 来判断
- 打开时，通过 http 协议获取服务器上的文件列表
- 以较好的格式展现给用户，用户可以使用上下键来选中
- 用户选中按下回车键后，通过 http 从服务器下载相应的压缩包
- 本地建立相同的文件名，并通过 io.Copy 将从服务器下载的压缩包内容复制给本地文件
- 解压本地文件到特定的目录
- 生产 exe 快捷方式到桌面
- 打开配置文件

# 服务端

- 提供两个服务
- 提取程序列表
- 程序下载服务，并记录日志（ip，时间，程序）

# 包

- go get -u github.com/manifoldco/promptui
- go get -u github.com/chzyer/readline