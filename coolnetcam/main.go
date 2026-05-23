package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var staticFS embed.FS

const (
	baseURL          = "https://video.coolnet.com.ua"
	keepAliveEvery   = 8 * time.Second
	discoverInterval = 5 * time.Minute
	hlsSegmentTrim   = 3 // only serve last N segments for instant playback
)

type Camera struct {
	Name        string `json:"name"`
	ChannelID   string `json:"channelId"`
	KeepaliveID string `json:"keepaliveId"`
	StreamURL   string `json:"streamUrl"`
	Active      bool   `json:"active"`
}

type App struct {
	mu       sync.RWMutex
	cameras  []Camera
	client   *http.Client
	cookie   string
	username string
	password string
}

func main() {
	username := os.Getenv("COOLNETCAM_USER")
	password := os.Getenv("COOLNETCAM_PASS")
	listenAddr := envOrDefault("COOLNETCAM_LISTEN", ":8090")

	if username == "" || password == "" {
		log.Fatal("COOLNETCAM_USER and COOLNETCAM_PASS must be set (configure them in the add-on Configuration tab)")
	}

	jar, _ := cookiejar.New(nil)
	app := &App{
		client: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
		username: username,
		password: password,
	}

	if err := app.login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	log.Printf("Logged in as %s", username)

	if err := app.discoverCameras(); err != nil {
		log.Fatalf("Camera discovery failed: %v", err)
	}

	app.mu.RLock()
	for _, c := range app.cameras {
		log.Printf("Camera: %s | channel=%s keepalive=%s", c.Name, c.ChannelID, c.KeepaliveID)
	}
	app.mu.RUnlock()

	// Activate all streams and start keepalive loops
	app.activateAll()

	// Periodic re-discovery in case cameras change
	go func() {
		for {
			time.Sleep(discoverInterval)
			if err := app.discoverCameras(); err != nil {
				log.Printf("Re-discovery failed: %v", err)
				// Try re-login
				if err := app.login(); err != nil {
					log.Printf("Re-login failed: %v", err)
				} else {
					app.discoverCameras()
				}
			}
			app.activateAll()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.handleIndex)
	mux.HandleFunc("/api/cameras", app.handleCameras)
	mux.HandleFunc("/stream/", app.handleStream)

	log.Printf("Listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

func (a *App) login() error {
	form := url.Values{
		"login_form": {"1"},
		"username":   {a.username},
		"password":   {a.password},
	}
	resp, err := a.client.PostForm(baseURL+"/index.php", form)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	// Extract cookie for raw HTTP requests
	u, _ := url.Parse(baseURL)
	for _, c := range a.client.Jar.Cookies(u) {
		if c.Name == "yalf_user" {
			a.cookie = c.Name + "=" + c.Value
			return nil
		}
	}
	return fmt.Errorf("no yalf_user cookie received")
}

var (
	reCamEntry  = regexp.MustCompile(`livechannel=([a-z0-9]+)"[^>]*><img[^>]*title="([^"]*)"`)
	reKeepalive = regexp.MustCompile(`keepstreamalive=(\d+)`)
)

func (a *App) discoverCameras() error {
	body, err := a.fetchPage("/?module=livecams")
	if err != nil {
		return err
	}

	matches := reCamEntry.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return fmt.Errorf("no cameras found in livecams page")
	}

	var cameras []Camera
	seen := map[string]bool{}
	for _, m := range matches {
		id := m[1]
		name := strings.TrimSpace(m[2])
		if seen[id] {
			continue
		}
		seen[id] = true
		if name == "" {
			name = id
		}
		cameras = append(cameras, Camera{
			Name:      name,
			ChannelID: id,
			StreamURL: fmt.Sprintf("/stream/%s/stream.m3u8", id),
		})
	}
	a.mu.Lock()
	a.cameras = cameras
	a.mu.Unlock()

	// Discover keepalive IDs by visiting each camera page
	a.mu.RLock()
	cams := make([]Camera, len(a.cameras))
	copy(cams, a.cameras)
	a.mu.RUnlock()

	for i := range cams {
		body, err := a.fetchPage(fmt.Sprintf("/?module=livecams&livechannel=%s", cams[i].ChannelID))
		if err != nil {
			log.Printf("Failed to fetch camera page for %s: %v", cams[i].ChannelID, err)
			continue
		}
		if m := reKeepalive.FindStringSubmatch(body); m != nil {
			cams[i].KeepaliveID = m[1]
			cams[i].Active = true
		}
	}

	a.mu.Lock()
	a.cameras = cams
	a.mu.Unlock()
	return nil
}

func (a *App) activateAll() {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, cam := range a.cameras {
		if cam.KeepaliveID != "" {
			go a.keepAliveLoop(cam)
		}
	}
}

func (a *App) keepAliveLoop(cam Camera) {
	log.Printf("Keepalive started for %s (id=%s)", cam.Name, cam.KeepaliveID)
	ticker := time.NewTicker(keepAliveEvery)
	defer ticker.Stop()

	// Immediate first keepalive
	a.sendKeepalive(cam)

	for range ticker.C {
		// Check if camera still exists
		a.mu.RLock()
		found := false
		for _, c := range a.cameras {
			if c.ChannelID == cam.ChannelID && c.KeepaliveID == cam.KeepaliveID {
				found = true
				break
			}
		}
		a.mu.RUnlock()
		if !found {
			log.Printf("Keepalive stopped for %s (removed or changed)", cam.Name)
			return
		}
		a.sendKeepalive(cam)
	}
}

func (a *App) sendKeepalive(cam Camera) {
	_, err := a.fetchPage(fmt.Sprintf("/?module=livecams&keepstreamalive=%s", cam.KeepaliveID))
	if err != nil {
		log.Printf("Keepalive failed for %s: %v", cam.Name, err)
	}
}

func (a *App) fetchPage(path string) (string, error) {
	req, err := http.NewRequest("GET", baseURL+path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", a.cookie)
	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// handleStream proxies HLS requests and trims m3u8 playlists for low latency
func (a *App) handleStream(w http.ResponseWriter, r *http.Request) {
	// Path: /stream/{channelId}/{file}
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad path", 400)
		return
	}
	channelID := parts[0]
	file := parts[1]

	upstream := fmt.Sprintf("%s/howl/livestreams/%s/%s", baseURL, channelID, file)
	req, err := http.NewRequest("GET", upstream, nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		http.Error(w, "upstream: "+resp.Status, resp.StatusCode)
		return
	}

	// For .ts segments, stream directly
	if strings.HasSuffix(file, ".ts") {
		w.Header().Set("Content-Type", "video/mp2t")
		w.Header().Set("Cache-Control", "no-cache")
		io.Copy(w, resp.Body)
		return
	}

	// For .m3u8, trim to last N segments for instant start
	body, _ := io.ReadAll(resp.Body)
	playlist := string(body)

	trimmed := trimPlaylist(playlist, hlsSegmentTrim)

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(trimmed))
}

// trimPlaylist keeps only the last n segments from an HLS playlist
func trimPlaylist(playlist string, n int) string {
	lines := strings.Split(strings.TrimSpace(playlist), "\n")

	// Find all segment pairs (EXTINF + URI)
	type segment struct{ extinf, uri string }
	var header []string
	var segments []segment
	var mediaSeq int
	inHeader := true

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "#EXTINF:") {
			inHeader = false
			if i+1 < len(lines) {
				segments = append(segments, segment{line, lines[i+1]})
				i++
			}
		} else if inHeader {
			if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				fmt.Sscanf(line, "#EXT-X-MEDIA-SEQUENCE:%d", &mediaSeq)
			} else {
				header = append(header, line)
			}
		}
	}

	// Keep only last n segments
	skip := 0
	if len(segments) > n {
		skip = len(segments) - n
	}
	segments = segments[skip:]
	mediaSeq += skip

	var b strings.Builder
	for _, h := range header {
		b.WriteString(h)
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "#EXT-X-MEDIA-SEQUENCE:%d\n", mediaSeq)
	for _, s := range segments {
		b.WriteString(s.extinf)
		b.WriteString("\n")
		b.WriteString(s.uri)
		b.WriteString("\n")
	}
	return b.String()
}

func (a *App) handleCameras(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a.cameras)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, _ := staticFS.ReadFile("index.html")
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
