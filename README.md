# ⚡ CF IP 优选工具

> 一键获取最快的 Cloudflare IP，优雅的 Web 界面，高效稳定

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%7CmacOS%7CLinux-lightgrey)]()

---

## 📖 简介

基于 CloudflareSpeedTest 核心算法开发的 IP 优选工具，提供美观的 Web 界面。支持高并发测速，可快速筛选出当前网络环境下延迟最低、速度最快的 Cloudflare IP。

**适用场景**：加速 Cloudflare 代理服务、优化 CDN 访问体验、自建优选 DNS

---

## ✨ 功能特性

| 功能 | 说明 |
|------|------|
| 🌓 双主题 | 白天/夜晚模式，自动记忆偏好 |
| ⚡ 高并发 | 50 并发 RTT 测试，快速定位 |
| 📋 历史记录 | 保存最近 10 条结果，点击 IP 复制 |
| 🛑 实时停止 | 扫描过程可随时中断 |
| 🔄 在线更新 | 一键获取最新 Cloudflare IP 段 |
| 🎯 灵活配置 | IPv4/IPv6、TLS/HTTP、自定义带宽 |

---

## 🖥️ 界面预览

### 扫描配置区域

| 扫描配置 | | | 更新数据 |
|---------|---|---|---------|
| IP协议 | IPv4 / IPv6 | 连接验证 | TLS / HTTP |
| 期望带宽 | 50 Mbps (可调) | | |

### 按钮区域

| 开始扫描 | 停止 |
|---------|-----|

### 扫描结果示例

| IP地址 | 实测带宽 | 目标带宽 | 峰值速度 | 延迟 |
|--------|---------|---------|---------|------|
| 104.17.216.169 | 99 Mbps | 50 Mbps | 12750 kB/s | 55 ms |

---

## 🚀 快速开始

### 🔹 方式一：直接运行（需要 Go 环境）

```bash
git clone https://github.com/wt20230521/CF-IP-Optimizer.git
cd CF-IP-Optimizer
go run main.go better.go scan.go
然后浏览器访问：http://localhost:8080

🔹 方式二：编译成独立 EXE（无需 Go 环境）
bash
go build -o CF优选IP.exe main.go better.go scan.go
双击 CF优选IP.exe 即可运行

🔹 方式三：下载预编译版本
前往 Releases 页面下载

📖 使用说明
🔹 第一步：更新数据
首次使用请点击「📥 更新数据」按钮获取最新 IP 列表

🔹 第二步：配置参数
参数	选项	说明
IP 协议	IPv4 / IPv6	选择要测试的 IP 类型
连接验证	TLS / HTTP	加密连接方式
期望带宽	10-1000 Mbps	达到此带宽即停止扫描
🔹 第三步：开始扫描
点击「🚀 开始扫描」按钮，等待 8-15 秒

🔹 第四步：查看结果
点击 IP 地址自动复制到剪贴板

历史记录自动保存最近 10 条

可随时点击「⏹️ 停止」中断扫描

📊 结果解读
指标	单位	说明
实测带宽	Mbps	实际测得的网络带宽
目标带宽	Mbps	你设置的期望带宽
峰值速度	kB/s	测速期间最高瞬时速度
延迟	ms	TCP 连接延迟
数据中心	-	IP 所在物理位置
用时	秒	完成扫描所需时间
📁 项目结构
text
CF-IP-Optimizer/
├── main.go          # Web 服务主程序
├── better.go        # 核心 API 接口
├── scan.go          # 测速算法引擎
├── README.md        # 项目文档
├── LICENSE          # MIT 许可证
└── screenshot.png   # 界面截图（可选）
🛠️ 技术栈
组件	技术
后端	Go 1.25+
前端	原生 HTML/CSS/JS
测速核心	原始 APP 算法
并发模型	Goroutine + Channel
❓ 常见问题
<details> <summary><b>Q: 测速结果全部为 0？</b></summary>
请先点击「📥 更新数据」获取最新 IP 列表，然后重试。

</details><details> <summary><b>Q: 扫描时间太长？</b></summary>
可适当降低期望带宽值（如 30 Mbps），或增加并发数。

</details><details> <summary><b>Q: 支持 IPv6 吗？</b></summary>
支持，在配置中选择 IPv6 即可。

</details><details> <summary><b>Q: 如何让局域网内其他人访问？</b></summary>
修改 main.go 中端口为 0.0.0.0:8080，重启程序即可。

</details><details> <summary><b>Q: 提示端口被占用？</b></summary>
关闭其他占用 8080 端口的程序，或修改 main.go 中的端口号。

</details>
📄 许可证
MIT License © 2026

🙏 致谢
原始 APP 源码作者

Cloudflare 全球网络

🔗 相关链接
GitHub 仓库：https://github.com/wt20230521/CF-IP-Optimizer

问题反馈：https://github.com/wt20230521/CF-IP-Optimizer/issues

<p align="center"> ⚡ 如果觉得好用，请给个 Star ⭐ </p> ```
