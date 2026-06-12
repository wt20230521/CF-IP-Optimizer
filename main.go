package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strconv"
    "sync"
    "time"
)

var (
    history   []map[string]interface{}
    historyMu sync.Mutex
)

func main() {
    SetCacheDir(".")
    
    go func() {
        log.Println("正在初始化数据...")
        initLocations()
        log.Println("数据初始化完成")
    }()
    
    http.HandleFunc("/", serveWebPage)
    http.HandleFunc("/api/scan", scanHandler)
    http.HandleFunc("/api/history", historyHandler)
    http.HandleFunc("/api/delete", deleteHandler)
    http.HandleFunc("/api/update-ips", updateIPsHandler)
    http.HandleFunc("/api/cancel", cancelHandler)

    log.Println("========================================")
    log.Println("CF IP 优选工具 - 网页版")
    log.Println("========================================")
    log.Println("服务器已启动，请访问：http://localhost:8080")
    log.Println("========================================")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveWebPage(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write([]byte(htmlTemplate))
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    
    v4 := true
    useTLS := true
    bandwidth := 50
    
    if r.URL.Query().Get("ipType") == "6" {
        v4 = false
    }
    if r.URL.Query().Get("useTLS") == "false" {
        useTLS = false
    }
    if s, err := strconv.Atoi(r.URL.Query().Get("bandwidth")); err == nil && s > 0 {
        bandwidth = s
    }
    
    log.Printf("开始扫描: IPv4=%v, TLS=%v, 目标带宽=%d Mbps", v4, useTLS, bandwidth)
    
    startTime := time.Now()
    resultJSON := GetIPs(v4, useTLS, bandwidth)
    
    var result ScanResult
    json.Unmarshal([]byte(resultJSON), &result)
    
    duration := int(time.Since(startTime).Seconds())
    if result.Elapsed == 0 {
        result.Elapsed = duration
    }
    
    response := map[string]interface{}{
        "success":         result.IP != "",
        "message":         result.Error,
        "ip":              result.IP,
        "actualBandwidth": result.RealBandwidth,
        "targetBandwidth": result.Bandwidth,
        "peakSpeed":       result.MaxSpeed,
        "latencyMs":       result.LatencyMs,
        "dataCenter":      result.DataCenter,
        "duration":        result.Elapsed,
    }
    
    if response["message"] == "" {
        response["message"] = "扫描完成"
    }
    
    if result.IP != "" {
        historyMu.Lock()
        record := map[string]interface{}{
            "id":               fmt.Sprintf("%d", time.Now().UnixNano()),
            "timestamp":        time.Now().Format("2006-01-02 15:04:05"),
            "ip":               result.IP,
            "targetBandwidth":  result.Bandwidth,
            "actualBandwidth":  result.RealBandwidth,
            "peakSpeed":        result.MaxSpeed,
            "latencyMs":        result.LatencyMs,
            "dataCenter":       result.DataCenter,
            "duration":         result.Elapsed,
        }
        history = append([]map[string]interface{}{record}, history...)
        if len(history) > 10 {
            history = history[:10]
        }
        historyMu.Unlock()
    }
    
    json.NewEncoder(w).Encode(response)
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    historyMu.Lock()
    defer historyMu.Unlock()
    if history == nil {
        history = []map[string]interface{}{}
    }
    json.NewEncoder(w).Encode(history)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    id := r.URL.Query().Get("id")
    historyMu.Lock()
    defer historyMu.Unlock()
    for i, record := range history {
        if record["id"] == id {
            history = append(history[:i], history[i+1:]...)
            break
        }
    }
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func updateIPsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    UpdateData()
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": true,
        "message": "IP 列表更新完成",
    })
}

func cancelHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    CancelScan()
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CF优选IP</title>
    <style>
        *{margin:0;padding:0;box-sizing:border-box}
        :root{--bg-card:#fff;--text-primary:#1f2937;--text-secondary:#6b7280;--border-color:#e5e7eb;--option-bg:#f3f4f6;--active-bg:linear-gradient(135deg,#667eea,#764ba2);--shadow:0 2px 8px rgba(0,0,0,0.06);--result-bg:#f3f4f6}
        [data-theme="dark"]{--bg-card:#16213e;--text-primary:#e2e8f0;--text-secondary:#94a3b8;--border-color:#334155;--option-bg:#0f3460;--shadow:0 2px 8px rgba(0,0,0,0.3);--result-bg:#0f3460}
        body{font-family:system-ui,sans-serif;background:linear-gradient(135deg,#667eea,#764ba2);min-height:100vh;padding:20px;transition:.3s}
        body.dark-mode{background:linear-gradient(135deg,#1a1a2e,#16213e)}
        .container{max-width:600px;margin:0 auto}
        .theme-toggle{position:fixed;top:20px;right:20px;width:48px;height:48px;border-radius:50%;background:rgba(255,255,255,0.2);backdrop-filter:blur(10px);display:flex;align-items:center;justify-content:center;cursor:pointer;font-size:24px;z-index:100}
        .card{background:var(--bg-card);border-radius:24px;padding:20px;margin-bottom:16px;box-shadow:var(--shadow)}
        .card-title{font-size:16px;font-weight:600;color:var(--text-primary);margin-bottom:16px;display:flex;justify-content:space-between;align-items:center}
        .update-btn{background:rgba(102,126,234,0.1);border:none;color:#667eea;padding:6px 12px;border-radius:20px;cursor:pointer}
        .config-row{display:flex;gap:16px;margin-bottom:20px;flex-wrap:wrap}
        .config-item{flex:1;min-width:120px}
        .config-label{font-size:13px;color:var(--text-secondary);margin-bottom:8px}
        .config-options{display:flex;gap:12px;background:var(--option-bg);padding:4px;border-radius:14px}
        .config-option{flex:1;text-align:center;padding:10px;border-radius:12px;cursor:pointer;color:var(--text-secondary)}
        .config-option.active{background:var(--active-bg);color:#fff}
        .config-input{width:100%;padding:12px;border:1px solid var(--border-color);border-radius:14px;background:var(--bg-card);color:var(--text-primary)}
        .button-group{display:flex;gap:12px;margin-top:8px}
        .scan-btn{flex:3;padding:14px;background:var(--active-bg);color:#fff;border:none;border-radius:16px;font-weight:600;cursor:pointer}
        .stop-btn{flex:1;padding:14px;background:#ef4444;color:#fff;border:none;border-radius:16px;font-weight:600;cursor:pointer}
        .scan-btn:disabled,.stop-btn:disabled{opacity:0.6;cursor:not-allowed}
        .scanning{text-align:center;padding:30px;color:#667eea}
        .spinner{display:inline-block;width:40px;height:40px;border:3px solid #e5e7eb;border-top-color:#667eea;border-radius:50%;animation:spin .8s linear infinite}
        @keyframes spin{to{transform:rotate(360deg)}}
        .result-ip{font-size:20px;font-weight:700;color:var(--text-primary);margin-bottom:12px;cursor:pointer}
        .result-stats{display:grid;grid-template-columns:repeat(2,1fr);gap:12px;margin-bottom:16px}
        .stat-item{background:var(--result-bg);border-radius:14px;padding:12px;text-align:center}
        .stat-label{font-size:11px;color:var(--text-secondary)}
        .stat-value{font-size:16px;font-weight:600;color:var(--text-primary)}
        .stat-value.speed{color:#10b981}
        .stat-value.latency{color:#f59e0b}
        .history-item{border-bottom:1px solid var(--border-color);padding:12px 0}
        .history-header{display:flex;justify-content:space-between;margin-bottom:8px}
        .history-time{font-size:12px;color:var(--text-secondary)}
        .history-delete{color:#ef4444;font-size:12px;cursor:pointer;padding:4px 8px;border-radius:8px;background:rgba(239,68,68,0.1)}
        .history-ip{font-size:15px;font-weight:600;color:#667eea;cursor:pointer;margin-bottom:6px}
        .history-detail{font-size:12px;color:var(--text-secondary)}
        .empty-history{text-align:center;padding:30px;color:var(--text-secondary)}
        .toast{position:fixed;bottom:30px;left:50%;transform:translateX(-50%);background:#1f2937;color:#fff;padding:10px 20px;border-radius:30px;opacity:0;transition:.3s;pointer-events:none}
        .toast.show{opacity:1}
    </style>
</head>
<body class="light-mode">
    <div class="theme-toggle" onclick="toggleTheme()">🌙</div>
    <div class="container">
        <div class="card">
            <div class="card-title">
                <span>⚙️ 扫描配置</span>
                <button class="update-btn" onclick="updateIPs()">📥 更新数据</button>
            </div>
            <div class="config-row">
                <div class="config-item">
                    <div class="config-label">IP 协议</div>
                    <div class="config-options" id="ipTypeOptions">
                        <div class="config-option active" data-value="4">IPv4</div>
                        <div class="config-option" data-value="6">IPv6</div>
                    </div>
                </div>
                <div class="config-item">
                    <div class="config-label">连接验证</div>
                    <div class="config-options" id="tlsOptions">
                        <div class="config-option active" data-value="true">TLS</div>
                        <div class="config-option" data-value="false">HTTP</div>
                    </div>
                </div>
            </div>
            <div class="config-row">
                <div class="config-item">
                    <div class="config-label">期望带宽 (Mbps)</div>
                    <input type="number" id="targetSpeed" class="config-input" value="50">
                </div>
            </div>
            <div class="button-group">
                <button class="scan-btn" id="scanBtn" onclick="startScan()">🚀 开始扫描</button>
                <button class="stop-btn" id="stopBtn" onclick="stopScan()" disabled>⏹️ 停止</button>
            </div>
        </div>
        <div class="card" id="resultCard" style="display:none">
            <div class="card-title">📡 扫描结果</div>
            <div id="resultContent"></div>
        </div>
        <div class="card">
            <div class="card-title">📋 历史记录</div>
            <div id="historyList"><div class="empty-history">暂无历史记录</div></div>
        </div>
    </div>
    <div id="toast" class="toast">已复制</div>
    <script>
        let currentIpType=4,currentUseTLS=true,currentScanController=null;
        function toggleTheme(){
            const b=document.body;
            if(b.classList.contains('light-mode')){
                b.classList.remove('light-mode');b.classList.add('dark-mode');
                localStorage.setItem('theme','dark');
            }else{
                b.classList.remove('dark-mode');b.classList.add('light-mode');
                localStorage.setItem('theme','light');
            }
        }
        if(localStorage.getItem('theme')==='dark'){
            document.body.classList.remove('light-mode');
            document.body.classList.add('dark-mode');
        }
        document.querySelectorAll('#ipTypeOptions .config-option').forEach(o=>{
            o.onclick=function(){
                document.querySelectorAll('#ipTypeOptions .config-option').forEach(x=>x.classList.remove('active'));
                this.classList.add('active');
                currentIpType=parseInt(this.dataset.value);
            }
        });
        document.querySelectorAll('#tlsOptions .config-option').forEach(o=>{
            o.onclick=function(){
                document.querySelectorAll('#tlsOptions .config-option').forEach(x=>x.classList.remove('active'));
                this.classList.add('active');
                currentUseTLS=this.dataset.value==='true';
            }
        });
        async function stopScan(){
            if(currentScanController)currentScanController.abort();
            await fetch('/api/cancel');
            showToast('已停止');
            document.getElementById('scanBtn').disabled=false;
            document.getElementById('scanBtn').innerHTML='🚀 开始扫描';
            document.getElementById('stopBtn').disabled=true;
        }
        async function updateIPs(){
            const btn=document.getElementById('updateDataBtn');
            const txt=btn.innerHTML;
            btn.innerHTML='⏳ 更新中...';
            btn.disabled=true;
            try{
                const resp=await fetch('/api/update-ips');
                const data=await resp.json();
                showToast(data.message||'更新成功');
            }catch(e){showToast('更新失败');}
            finally{btn.innerHTML=txt;btn.disabled=false;}
        }
        async function loadHistory(){
            try{
                const resp=await fetch('/api/history');
                const h=await resp.json();
                const div=document.getElementById('historyList');
                if(!h.length){div.innerHTML='<div class="empty-history">暂无历史记录</div>';return;}
                let html='';
                for(const r of h){
                    html+='<div class="history-item"><div class="history-header"><span class="history-time">'+r.timestamp+'</span><span class="history-delete" onclick="deleteRecord(\''+r.id+'\')">删除</span></div>';
                    html+='<div class="history-ip" onclick="copyToClipboard(\''+r.ip+'\')">'+r.ip+'</div>';
                    html+='<div class="history-detail">实测 '+r.actualBandwidth+' Mbps / 目标 '+r.targetBandwidth+' Mbps<br>峰值 '+r.peakSpeed+' kB/s / 延迟 '+r.latencyMs+' ms<br>数据中心 '+r.dataCenter+' / 用时 '+r.duration+' 秒</div></div>';
                }
                div.innerHTML=html;
            }catch(e){}
        }
        async function deleteRecord(id){await fetch('/api/delete?id='+id);loadHistory();}
        function copyToClipboard(t){navigator.clipboard.writeText(t);showToast('已复制: '+t);}
        function showToast(m){const t=document.getElementById('toast');t.innerText=m;t.classList.add('show');setTimeout(()=>t.classList.remove('show'),2000);}
        async function startScan(){
            const btn=document.getElementById('scanBtn'),stopBtn=document.getElementById('stopBtn');
            const resultDiv=document.getElementById('resultContent'),target=document.getElementById('targetSpeed').value;
            btn.disabled=true;btn.innerHTML='⏳ 扫描中...';stopBtn.disabled=false;
            document.getElementById('resultCard').style.display='block';
            resultDiv.innerHTML='<div class="scanning"><div class="spinner"></div><div>正在扫描...</div></div>';
            currentScanController=new AbortController();
            try{
                const resp=await fetch('/api/scan?ipType='+currentIpType+'&useTLS='+currentUseTLS+'&bandwidth='+target,{signal:currentScanController.signal});
                const d=await resp.json();
                if(d.success&&d.ip){
                    resultDiv.innerHTML='<div class="result-ip" onclick="copyToClipboard(\''+d.ip+'\')">'+d.ip+' 📋</div>'+
                        '<div class="result-stats">'+
                        '<div class="stat-item"><div class="stat-label">实测带宽</div><div class="stat-value speed">'+d.actualBandwidth+' Mbps</div></div>'+
                        '<div class="stat-item"><div class="stat-label">目标带宽</div><div class="stat-value">'+d.targetBandwidth+' Mbps</div></div>'+
                        '<div class="stat-item"><div class="stat-label">峰值速度</div><div class="stat-value">'+d.peakSpeed+' kB/s</div></div>'+
                        '<div class="stat-item"><div class="stat-label">延迟</div><div class="stat-value latency">'+d.latencyMs+' ms</div></div>'+
                        '</div><div class="stat-item"><div class="stat-label">数据中心</div><div class="stat-value">'+(d.dataCenter||'Unknown')+' / '+d.duration+'秒</div></div>';
                }else{
                    resultDiv.innerHTML='<div style="text-align:center;padding:20px;color:#ef4444;">❌ '+(d.message||'未找到IP')+'</div>';
                }
                loadHistory();
            }catch(e){
                if(e.name==='AbortError')resultDiv.innerHTML='<div style="text-align:center;padding:20px;color:#f59e0b;">⏹️ 已停止</div>';
                else resultDiv.innerHTML='<div style="text-align:center;padding:20px;color:#ef4444;">❌ 请求失败</div>';
            }finally{
                btn.disabled=false;btn.innerHTML='🚀 开始扫描';
                stopBtn.disabled=true;currentScanController=null;
            }
        }
        loadHistory();
    </script>
</body>
</html>`