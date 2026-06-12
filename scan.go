package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ----------------------- 包级全局变量 -----------------------

var (
	dataDir         string
	randomMu        sync.Mutex
	randomGenerator = rand.New(rand.NewSource(time.Now().UnixNano()))
	locationMap     map[string]location
	locationMu      sync.RWMutex
	speedTestDomain string
	speedTestFile   string
	progress        string
	progressMu      sync.Mutex
	cancelCtx       context.Context
	cancelCancel    context.CancelFunc
	cancelMu        sync.Mutex
)
func scanCtx() context.Context {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	if cancelCtx != nil {
		return cancelCtx
	}
	return context.Background()
}

type location struct {
	Iata   string  `json:"iata"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Cca2   string  `json:"cca2"`
	Region string  `json:"region"`
	City   string  `json:"city"`
}

// ----------------------- 工具函数 -----------------------

func dataPath(name string) string {
	if dataDir == "" {
		return name
	}
	return filepath.Join(dataDir, name)
}

var downloadClient = &http.Client{Timeout: 8 * time.Second}

func timeNow() time.Time {
	return time.Now()
}

func timeSince(t time.Time) time.Duration {
	return time.Since(t)
}

func getURLContent(targetURL string) (string, error) {
	req, _ := http.NewRequestWithContext(scanCtx(), "GET", targetURL, nil)
	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func getFileContent(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func saveToFile(filename, content string) error {
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return os.WriteFile(filename, []byte(content), 0644)
}

func removeFile(path string) {
	os.Remove(path)
}

func parseIPList(content string) []string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var ipList []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			ipList = append(ipList, line)
		}
	}
	return ipList
}

func nextRandomIntn(n int) int {
	randomMu.Lock()
	defer randomMu.Unlock()
	return randomGenerator.Intn(n)
}

func getRandomIPv4s(ipList []string) []string {
	var randomIPs []string
	for _, subnet := range ipList {
		subnet = strings.TrimSpace(subnet)
		if subnet == "" {
			continue
		}
		if idx := strings.Index(subnet, "/"); idx >= 0 {
			subnet = subnet[:idx]
		}
		octets := strings.Split(subnet, ".")
		if len(octets) == 4 {
			octets[3] = fmt.Sprintf("%d", nextRandomIntn(256))
			randomIPs = append(randomIPs, strings.Join(octets, "."))
		}
	}
	return randomIPs
}

func getRandomIPv6s(ipList []string) []string {
	var randomIPs []string
	for _, subnet := range ipList {
		subnet = strings.TrimSpace(subnet)
		if subnet == "" {
			continue
		}
		if idx := strings.Index(subnet, "/"); idx >= 0 {
			subnet = subnet[:idx]
		}
		// 展开 :: 压缩，确保有 8 段
		if strings.Contains(subnet, "::") {
			parts := strings.Split(subnet, "::")
			left := strings.Split(parts[0], ":")
			var right []string
			if len(parts) > 1 && parts[1] != "" {
				right = strings.Split(parts[1], ":")
			}
			missing := 8 - len(left) - len(right)
			sections := left
			for range missing {
				sections = append(sections, "0")
			}
			sections = append(sections, right...)
			subnet = strings.Join(sections, ":")
		}
		sections := strings.Split(subnet, ":")
		if len(sections) >= 3 {
			sections = sections[:3]
			for i := 3; i < 8; i++ {
				sections = append(sections, fmt.Sprintf("%x", nextRandomIntn(65536)))
			}
			randomIPs = append(randomIPs, strings.Join(sections, ":"))
		}
	}
	return randomIPs
}

// randomSample 从列表中随机抽取 n 个元素
func randomSample(list []string, n int) []string {
	shuffled := make([]string, len(list))
	copy(shuffled, list)
	randomMu.Lock()
	randomGenerator.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	randomMu.Unlock()
	if n > len(shuffled) {
		n = len(shuffled)
	}
	return shuffled[:n]
}

// ----------------------- 数据下载 -----------------------

// downloadAllData 确保所有数据文件存在，缺失则自动下载
func downloadAllData() {
	urlFilename := dataPath("url.txt")
	if _, err := os.Stat(urlFilename); os.IsNotExist(err) {
		if isCancelled() {
			return
		}
		setProgress("正在下载测速 URL...")
		content, err := getURLContent("https://www.baipiao.eu.org/cloudflare/url")
		if err != nil {
			setProgress("下载测速 URL 失败: " + err.Error())
			return
		}
		if err := saveToFile(urlFilename, content); err != nil {
			setProgress("保存测速 URL 失败: " + err.Error())
			return
		}
	}

	if isCancelled() {
		return
	}

	content, err := getFileContent(urlFilename)
	if err != nil {
		setProgress("读取测速 URL 失败: " + err.Error())
		return
	}
	content = strings.TrimSpace(content)
	parts := strings.SplitN(content, "/", 2)
	if len(parts) == 2 {
		speedTestDomain = parts[0]
		speedTestFile = parts[1]
	}

	for _, item := range []struct{ file, url string }{
		{"ips-v4.txt", "https://www.baipiao.eu.org/cloudflare/ips-v4"},
		{"ips-v6.txt", "https://www.baipiao.eu.org/cloudflare/ips-v6"},
	} {
		if isCancelled() {
			return
		}
		fp := dataPath(item.file)
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			setProgress("正在下载 IP 列表: " + item.file)
			c, err := getURLContent(item.url)
			if err != nil {
				setProgress("下载 IP 列表失败: " + err.Error())
				return
			}
			if err := saveToFile(fp, c); err != nil {
				setProgress("保存 IP 列表失败: " + err.Error())
				return
			}
		}
	}

	if isCancelled() {
		return
	}
	fp := dataPath("locations.json")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		setProgress("正在下载位置信息...")
		req, _ := http.NewRequestWithContext(scanCtx(), "GET", "https://www.baipiao.eu.org/cloudflare/locations", nil)
		resp, err := downloadClient.Do(req)
		if err != nil {
			setProgress("获取位置信息失败: " + err.Error())
			return
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
		resp.Body.Close()
		if err != nil {
			setProgress("读取响应内容失败: " + err.Error())
			return
		}
		if err := saveToFile(fp, string(body)); err != nil {
			setProgress("保存位置信息失败: " + err.Error())
			return
		}
	}
}

// initLocations 初始化数据中心位置信息
func initLocations() {
	downloadAllData()

	if isCancelled() {
		return
	}
	fp := dataPath("locations.json")
	body, err := os.ReadFile(fp)
	if err != nil {
		setProgress("读取位置文件失败: " + err.Error())
		return
	}

	var locations []location
	if err := json.Unmarshal(body, &locations); err != nil {
		setProgress("解析位置信息 JSON 失败: " + err.Error())
		return
	}

	loadedMap := make(map[string]location)
	for _, loc := range locations {
		loadedMap[loc.Iata] = loc
	}

	locationMu.Lock()
	locationMap = loadedMap
	locationMu.Unlock()

	setProgress(fmt.Sprintf("已加载 %d 个数据中心位置信息", len(loadedMap)))
}

// ----------------------- RTT 测试 -----------------------

// RTTResult RTT 测试结果
type RTTResult struct {
	IP        string
	LatencyMs int
}

// testRTT 测试单个 IP 的 RTT（TCP 连接 + 验证 CF-RAY）
func testRTT(ip string, useTLS bool) int {
	port := 80
	if useTLS {
		port = 443
	}

	var totalMs int
	for range 3 {
		start := time.Now()
		var d = net.Dialer{Timeout: 1 * time.Second}
		conn, err := d.DialContext(scanCtx(), "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
		if err != nil {
			return 0
		}
		tcpDuration := time.Since(start)

		conn.SetDeadline(start.Add(1 * time.Second))

		var rwc net.Conn = conn
		if useTLS {
			tlsConn := tls.Client(conn, &tls.Config{ServerName: "cloudflare.com", InsecureSkipVerify: true})
			if err := tlsConn.Handshake(); err != nil {
				conn.Close()
				return 0
			}
			rwc = tlsConn
		}

		reqStr := "GET / HTTP/1.1\r\nHost: cloudflare.com\r\nUser-Agent: Mozilla/5.0\r\nConnection: close\r\n\r\n"
		_, err = rwc.Write([]byte(reqStr))
		if err != nil {
			rwc.Close()
			return 0
		}

		reader := bufio.NewReader(rwc)
		resp, err := http.ReadResponse(reader, nil)
		rwc.Close()
		if err != nil {
			return 0
		}
		resp.Body.Close()

		if resp.Header.Get("CF-RAY") == "" {
			return 0
		}

		totalMs += int(tcpDuration.Milliseconds())
	}

	return totalMs / 3
}

// runRTTTest 运行 RTT 测试（并发，带进度显示）
func runRTTTest(ipList []string, taskNum int, useTLS bool) []RTTResult {
	if len(ipList) < taskNum {
		taskNum = len(ipList)
	}

	var wg sync.WaitGroup
	resultChan := make(chan RTTResult, len(ipList))
	thread := make(chan struct{}, taskNum)
	var count int
	var mu sync.Mutex
	total := len(ipList)

	for _, ip := range ipList {
		if isCancelled() {
			break
		}
		wg.Add(1)
		thread <- struct{}{}
		go func(ip string) {
			defer func() {
				<-thread
				wg.Done()
				mu.Lock()
				count++
				current := count
				mu.Unlock()
				if current%10 == 0 || current == total {
					setProgress(fmt.Sprintf("RTT 测试进度: %d/%d", current, total))
				}
			}()

			if isCancelled() {
				return
			}
			avgMs := testRTT(ip, useTLS)
			if avgMs > 0 {
				resultChan <- RTTResult{IP: ip, LatencyMs: avgMs}
			}
		}(ip)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []RTTResult
	for r := range resultChan {
		results = append(results, r)
	}

	if isCancelled() {
		return nil
	}
	// 按最小延迟排序，最多保留前 10 个进入速度测试
	sort.Slice(results, func(i, j int) bool {
		return results[i].LatencyMs < results[j].LatencyMs
	})

	if len(results) > 10 {
		setProgress(fmt.Sprintf("RTT 测试完成，%d/%d 个 IP 有效，保留延迟最低的 10 个", len(results), total))
		results = results[:10]
	} else {
		setProgress(fmt.Sprintf("RTT 测试完成，%d/%d 个 IP 有效", len(results), total))
	}
	return results
}

// ----------------------- 速度测试 -----------------------

// runSpeedTestSimple 简单速度测试，返回 (峰值速度 kB/s, TCP延迟ms, 三字码头)
func runSpeedTestSimple(ip string, port int, useTLS bool) (int, int, string) {
	var tcpMs int
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			start := time.Now()
			conn, err := (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
			if err == nil {
				tcpMs = int(time.Since(start).Milliseconds())
			}
			return conn, err
		},
	}
	if useTLS {
		transport.TLSClientConfig = &tls.Config{ServerName: speedTestDomain}
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	testURL := fmt.Sprintf("%s://%s/%s", scheme, speedTestDomain, speedTestFile)

	req, _ := http.NewRequestWithContext(scanCtx(), "GET", testURL, nil)
		resp, err := client.Do(req)
	if err != nil {
		return 0, 0, ""
	}
	defer resp.Body.Close()

	cfRay := resp.Header.Get("CF-RAY")
	dataCenter := extractDataCenter(cfRay)

	buf := make([]byte, 32*1024)
	var totalBytes int64
	var windowBytes int64
	windowStart := time.Now()
	maxSpeed := 0
	for {

		n, err := resp.Body.Read(buf)
		totalBytes += int64(n)
		windowBytes += int64(n)
		if err != nil {
			break
		}

		elapsed := time.Since(windowStart).Seconds()
		if elapsed >= 1.0 {
			speedKB := int(float64(windowBytes) / 1024 / elapsed)
			if speedKB > maxSpeed {
				maxSpeed = speedKB
			}
			windowBytes = 0
			windowStart = time.Now()
		}
	}

	return maxSpeed, tcpMs, dataCenter
}

// extractDataCenter 从 CF-RAY 头提取三字码头
func extractDataCenter(cfRay string) string {
	if cfRay == "" {
		return ""
	}
	parts := strings.Split(cfRay, "-")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

// lookupDataCenter 查找数据中心名称
func lookupDataCenter(colo string) string {
	locationMu.RLock()
	loc := locationMap[colo]
	locationMu.RUnlock()

	if loc.City != "" {
		return loc.City
	}
	return colo
}

// ----------------------- 核心测试逻辑 -----------------------

// cloudflareTest 核心测试逻辑（从原 cloudflareTest 迁移，去掉 stdin 交互）
func cloudflareTest(ipType int, useTLS bool, taskNum int, speed int) (string, int, int, string) {
	initLocations()
	if isCancelled() {
		return "", 0, 0, ""
	}
	filename := dataPath("ips-v4.txt")
	if ipType == 6 {
		filename = dataPath("ips-v6.txt")
	}
	content, err := getFileContent(filename)
	if err != nil {
		setProgress("读取 IP 列表失败: " + err.Error())
		return "", 0, 0, ""
	}
	ipList := parseIPList(content)
	setProgress(fmt.Sprintf("正在从 %d 个子网中随机生成 IP...", len(ipList)))

	sampleSize := 100
	if len(ipList) < sampleSize {
		sampleSize = len(ipList)
	}

	for {
		var rttResults []RTTResult
		if isCancelled() {
			setProgress("扫描已取消")
			return "", 0, 0, ""
		}
		for {
			sampled := randomSample(ipList, sampleSize)

			var testIPs []string
			if ipType == 6 {
				testIPs = getRandomIPv6s(sampled)
			} else {
				testIPs = getRandomIPv4s(sampled)
			}

			setProgress(fmt.Sprintf("已生成 %d 个测试 IP，开始 RTT 测试...", len(testIPs)))

			rttResults = runRTTTest(testIPs, taskNum, useTLS)
			if isCancelled() {
				break
			}
			if len(rttResults) > 0 {
				break
			}
			setProgress("当前所有 IP 都存在 RTT 丢包，继续新的 RTT 测试...")
		}

		// 速度测试：依次测试，满足带宽目标立即返回
		for _, r := range rttResults {
			if isCancelled() {
				setProgress("扫描已取消")
				return "", 0, 0, ""
			}

			setProgress(fmt.Sprintf("正在测速 %s (延迟 %dms)", r.IP, r.LatencyMs))
			speedPort := 80
			if useTLS {
				speedPort = 443
			}
			maxSpeed, tcpMs, dc := runSpeedTestSimple(r.IP, speedPort, useTLS)
			dcName := dc
			if dc != "" {
				dcName = lookupDataCenter(dc)
			}
			setProgress(fmt.Sprintf("%s 峰值速度 %d kB/s, 数据中心 %s", r.IP, maxSpeed, dcName))

			// 带宽达标 → 立即返回（原版逻辑）
			if maxSpeed >= speed {
				setProgress(fmt.Sprintf("找到优选 IP: %s, 速度 %d kB/s, 延迟 %dms", r.IP, maxSpeed, tcpMs))
				return r.IP, maxSpeed, tcpMs, dcName
			}
		}

		setProgress("当前所有 IP 都未达到期望带宽，重新开始新一轮测试...")
	}
}
