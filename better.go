package main

import (
	"context"
	"encoding/json"
	"fmt"
)

// ScanResult 扫描结果
type ScanResult struct {
	IP            string `json:"ip"`
	Bandwidth     int    `json:"bandwidth"`     // 期望带宽 Mbps
	RealBandwidth int    `json:"realBandwidth"` // 实测带宽 Mbps
	MaxSpeed      int    `json:"maxSpeed"`      // 峰值速度 kB/s
	LatencyMs     int    `json:"latencyMs"`
	DataCenter    string `json:"dataCenter"`
	Elapsed       int    `json:"elapsed"` // 总计用时 秒
	Error         string `json:"error"`
}

// SetCacheDir 设置缓存目录（Android 应用数据目录）
// gomobile 必须显式设置，Android 下通常设为 Context.getFilesDir()
func SetCacheDir(dir string) {
	dataDir = dir
}

// GetProgress 返回当前进度描述，供 Android 端轮询
func GetProgress() string {
	progressMu.Lock()
	defer progressMu.Unlock()
	return progress
}

func setProgress(s string) {
	progressMu.Lock()
	progress = s
	progressMu.Unlock()
}

// CancelScan 取消正在进行的扫描（立即中断所有网络操作）
func CancelScan() {
	cancelMu.Lock()
	if cancelCancel != nil {
		cancelCancel()
	}
	cancelMu.Unlock()
	setProgress("用户已取消扫描")
}

func isCancelled() bool {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	if cancelCtx == nil {
		return false
	}
	select {
	case <-cancelCtx.Done():
		return true
	default:
		return false
	}
}

func resetCancel() {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	cancelCtx, cancelCancel = context.WithCancel(context.Background())
}

// GetIPs 运行 Cloudflare IP 优选，返回结果 JSON
// v4: true=IPv4 false=IPv6
// useTLS: 是否启用 TLS 握手
// bandwidth: 期望带宽（Mbps），设为 0 则使用默认 1 Mbps
// 阻塞调用，需在后台线程执行
func GetIPs(v4 bool, useTLS bool, bandwidth int) string {
	setProgress("正在初始化...")
	resetCancel()

	ipType := 4
	if !v4 {
		ipType = 6
	}

	if bandwidth <= 0 {
		bandwidth = 1
	}

	// 转为 kB/s
	speedTarget := bandwidth * 128

	startTime := timeNow()

	ip, maxSpeed, avgMs, dataCenter := cloudflareTest(ipType, useTLS, 50, speedTarget)

	realBandwidth := maxSpeed / 128
	elapsed := int(timeSince(startTime).Seconds())

	result := ScanResult{
		IP:            ip,
		Bandwidth:     bandwidth,
		RealBandwidth: realBandwidth,
		MaxSpeed:      maxSpeed,
		LatencyMs:     avgMs,
		DataCenter:    dataCenter,
		Elapsed:       elapsed,
	}

	if ip == "" {
		result.Error = fmt.Sprintf("未找到符合 %d Mbps 带宽目标的 IP（用时 %d 秒）", bandwidth, elapsed)
	}

	setProgress(fmt.Sprintf("扫描完成，用时 %d 秒", elapsed))

	b, _ := json.Marshal(result)
	return string(b)
}

// UpdateData 重新下载所有数据文件（清空缓存后重新下载）
func UpdateData() {
	setProgress("正在更新数据...")
	for _, f := range []string{"locations.json", "ips-v4.txt", "ips-v6.txt", "url.txt"} {
		removeFile(dataPath(f))
	}
	initLocations()
	setProgress("数据更新完成")
}

// ClearCache 清除缓存的数据文件
func ClearCache() {
	setProgress("正在清除缓存...")
	for _, f := range []string{"locations.json", "ips-v4.txt", "ips-v6.txt", "url.txt"} {
		removeFile(dataPath(f))
	}
	setProgress("缓存已清除")
}
