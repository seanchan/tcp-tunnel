# TCP Tunnel
<span style="font-weight: bold; color: blue;">[English](./README.md) </span> | [中文] <!-- 切换链接 -->

## 简介

这是一个 TCP 隧道工具，支持服务器和客户端的构建和使用。

## 安装

请从 [Releases](https://github.com/yourusername/yourrepo/releases) 页面下载适合你平台的服务器和客户端。

## 使用

### 服务器

1. 下载并解压服务器文件。
2. 在终端中运行以下命令启动服务器：

   ```bash
   ./tcp-tunnel-server
   ```

3. 服务器启动后，记下显示的地址和端口。

### 客户端

1. 下载并解压客户端文件。
2. 在终端中运行以下命令连接到服务器：

   ```bash
   ./tcp-tunnel-client -s <服务器地址> -p <服务器端口>
   ```

3. 客户端连接成功后，你可以开始使用 TCP 隧道。

## 贡献

欢迎提交问题和拉取请求！