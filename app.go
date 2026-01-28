package main

import (
	"context"
	"crypto"
	_ "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/minio/selfupdate"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/mod/semver"
)

var Version = "0.0.0"

type App struct {
	ctx context.Context

	mu         sync.Mutex
	buttonText string
}

func NewApp() *App {
	return &App{buttonText: "Output"} // default
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.loadState()
}

type manifest struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

type updateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	HasUpdate bool   `json:"hasUpdate"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Error     string `json:"error,omitempty"`
}

// ------- Persistent State (ปุ่มเปลี่ยนแล้วเปิดใหม่ต้องจำ) -------

type uiState struct {
	ButtonText string `json:"buttonText"`
}

func (a *App) GetButtonText() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buttonText
}

func (a *App) SetButtonText(t string) error {
	a.mu.Lock()
	a.buttonText = t
	a.mu.Unlock()
	return a.saveState()
}

func (a *App) statePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "myapp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func (a *App) loadState() error {
	p, err := a.statePath()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var s uiState
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.ButtonText != "" {
		a.mu.Lock()
		a.buttonText = s.ButtonText
		a.mu.Unlock()
	}
	return nil
}

func (a *App) saveState() error {
	p, err := a.statePath()
	if err != nil {
		return err
	}
	a.mu.Lock()
	s := uiState{ButtonText: a.buttonText}
	a.mu.Unlock()

	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(p, b, 0o644)
}

// ------- Update Flow -------

func (a *App) CheckUpdate(manifestURL string) updateInfo {
	info := updateInfo{Current: Version}

	m, err := fetchManifest(manifestURL)
	if err != nil {
		info.Error = err.Error()
		return info
	}

	info.Latest = m.Version
	info.URL = m.URL
	info.SHA256 = m.SHA256

	// semver ต้องมี prefix v
	cur := Version
	lat := m.Version
	if cur != "" && cur[0] != 'v' {
		cur = "v" + cur
	}
	if lat != "" && lat[0] != 'v' {
		lat = "v" + lat
	}
	info.HasUpdate = semver.IsValid(cur) && semver.IsValid(lat) && semver.Compare(lat, cur) > 0
	return info
}

func fetchManifest(url string) (manifest, error) {
	resp, err := http.Get(url)
	if err != nil {
		return manifest{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return manifest{}, errors.New("manifest http status not ok")
	}

	var m manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return manifest{}, err
	}
	if m.Version == "" || m.URL == "" || m.SHA256 == "" {
		return manifest{}, errors.New("manifest missing fields")
	}
	return m, nil
}

// ผู้ใช้กดอัปเดต
type applyResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func (a *App) ApplyUpdate(url string, sha256Hex string) applyResult {
	resp, err := http.Get(url)
	if err != nil {
		return applyResult{OK: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return applyResult{OK: false, Error: err.Error()}
	}

	sum, err := hex.DecodeString(sha256Hex)
	if err != nil {
		return applyResult{OK: false, Error: "bad sha256: " + err.Error()}
	}

	// replace executable
	if err := selfupdate.Apply(
		bytesReader(body),
		selfupdate.Options{Hash: crypto.SHA256, Checksum: sum},
	); err != nil {
		// rollback if possible
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return applyResult{OK: false, Error: "update failed + rollback failed"}
		}
		return applyResult{OK: false, Error: err.Error()}
	}

	// แจ้ง frontend ว่าอัปเดตแล้ว
	wruntime.EventsEmit(a.ctx, "update:applied", nil)
	return applyResult{OK: true, Message: "อัปเดตสำเร็จ ✅ กรุณารีสตาร์ทแอป"}
}

// helper: io.Reader จาก []byte แบบไม่ import bytes ให้ยืดยาว
type br struct {
	b []byte
	i int
}

func bytesReader(b []byte) *br { return &br{b: b} }
func (r *br) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func (a *App) Quit() { wruntime.Quit(a.ctx) }

func (a *App) RestartApp() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe)
	_ = cmd.Start()
	wruntime.Quit(a.ctx)
	return nil
}

func (a *App) GetAppVersion() string {
	return Version
}
