# CF IP 优选工具 - 网页版

基于原始 APP 核心算法，一键获取最快的 Cloudflare IP。

## ✨ 功能特点

- 🌓 白天/夜晚双主题，自动记忆
- ⚡ 高并发测速，快速定位最优 IP
- 📋 历史记录保存最近10条，点击 IP 可复制
- 🛑 扫描过程可随时停止
- 🔄 一键更新最新 IP 列表
- 🎯 支持 IPv4/IPv6、TLS/HTTP、自定义带宽

## 🚀 使用方法

### 方式一：直接运行（需要 Go 环境）

```bash
go run main.go better.go scan.go
###方式二：编译成可执行文件
bash
go build -o cfip-optimizer.exe main.go better.go scan.go
然后双击 cfip-optimizer.exe
📸 效果图
https://github.com/wt20230521/CF-IP-Optimizer/raw/main/screenshot.png

📄 许可证
MIT License

🔗 仓库地址
https://github.com/wt20230521/CF-IP-Optimizer
