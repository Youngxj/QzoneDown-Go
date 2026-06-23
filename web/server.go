package web

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"
)

//go:embed static
var staticFiles embed.FS

func Start() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/config", corsMiddleware(handleConfig))
	mux.HandleFunc("/api/albums", corsMiddleware(handleAlbums))
	mux.HandleFunc("/api/download", corsMiddleware(handleDownload))
	mux.HandleFunc("/api/download/progress", corsMiddleware(handleDownloadProgress))
	mux.HandleFunc("/api/login/start", corsMiddleware(handleLoginStart))
	mux.HandleFunc("/api/login/qrcode", corsMiddleware(handleLoginQrcode))
	mux.HandleFunc("/api/login/check", corsMiddleware(handleLoginCheck))

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	addr := fmt.Sprintf(":%s", port)
	fmt.Printf("  _____   ____                      _____                      \n / __  \\ |  _ \\                    |  __ \\                     \n| |  | | | |_) | ___   ___   ___  | |  | | _____      ___ __  \n| |  | | |  _ < / _ \\ / _ \\ / _ \\ | |  | |/ _ \\ \\ /\\ / / '_ \\ \n| |__| | | |_) | (_) | (_) |  __/ | |__| | (_) \\ V  V /| | | |\n \\___\\_/ |____/ \\___/ \\___/ \\___| |_____/ \\___/ \\_/\\_/ |_| |_|\n                                                               \n")
	fmt.Printf("Web UI started at http://localhost%s\n", addr)

	// Auto-open browser after short delay
	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser("http://localhost" + addr)
	}()

	fmt.Printf("Press Ctrl+C to stop\n")
	log.Fatal(http.ListenAndServe(addr, mux))
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		// Fallback for older Windows
		if runtime.GOOS == "windows" {
			exec.Command("explorer", url).Start()
		}
		fmt.Printf("Failed to open browser: %s\n", err)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":"%s"}`, msg)
}
