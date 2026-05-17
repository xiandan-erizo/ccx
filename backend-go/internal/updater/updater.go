package updater

import (
	"context"
	"crypto"
	_ "crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	update "github.com/inconshreveable/go-update"
)

const (
	repoOwner = "BenedictKing"
	repoName  = "ccx"
	apiBase   = "https://api.github.com"
)

var ErrAlreadyUpdating = errors.New("already updating")

type UpdateStatus struct {
	CurrentVersion       string `json:"current_version"`
	LatestVersion        string `json:"latest_version"`
	HasUpdate            bool   `json:"has_update"`
	CanUpdate            bool   `json:"can_update"`
	IsDocker             bool   `json:"is_docker"`
	UpdateDisabledReason string `json:"update_disabled_reason"`
	ReleaseNotes         string `json:"release_notes"`
	ReleaseURL           string `json:"release_url"`
	IsUpdating           bool   `json:"is_updating"`
	CheckedAt            string `json:"checked_at"`
}

type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Body       string        `json:"body"`
	HTMLURL    string        `json:"html_url"`
	Prerelease bool          `json:"prerelease"`
	Draft      bool          `json:"draft"`
	Assets     []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Updater struct {
	currentVersion string
	shutdownFunc   func()
	httpClient     *http.Client
	mu             sync.Mutex
	isUpdating     bool
	lastStatus     *UpdateStatus
	isDocker       bool
}

func New(version string, shutdownFunc func()) *Updater {
	return &Updater{
		currentVersion: version,
		shutdownFunc:   shutdownFunc,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		isDocker: detectDocker(),
	}
}

func detectDocker() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}

func (u *Updater) IsDocker() bool {
	return u.isDocker
}

func (u *Updater) StartUpdate() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.isUpdating {
		return false
	}
	u.isUpdating = true
	return true
}

func (u *Updater) IsUpdating() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.isUpdating
}

func (u *Updater) GetLastStatus() *UpdateStatus {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.lastStatus != nil {
		cp := *u.lastStatus
		cp.IsUpdating = u.isUpdating
		return &cp
	}
	return nil
}

func (u *Updater) CheckUpdate(ctx context.Context) (*UpdateStatus, error) {
	status := &UpdateStatus{
		CurrentVersion: u.currentVersion,
		IsDocker:       u.isDocker,
		IsUpdating:     u.IsUpdating(),
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}

	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		status.UpdateDisabledReason = fmt.Sprintf("检查更新失败: %v", err)
		u.mu.Lock()
		u.lastStatus = status
		u.mu.Unlock()
		return status, err
	}

	status.LatestVersion = release.TagName
	status.ReleaseNotes = release.Body
	status.ReleaseURL = release.HTMLURL

	if !isUpgradeableVersion(u.currentVersion) {
		status.UpdateDisabledReason = "当前版本不是可升级的正式版本"
	} else if compareVersions(release.TagName, u.currentVersion) > 0 {
		status.HasUpdate = true
	}

	status.CanUpdate = status.HasUpdate
	if status.CanUpdate {
		if u.isDocker {
			status.CanUpdate = false
			status.UpdateDisabledReason = "Docker 环境不支持内置升级，请使用 Watchtower 或拉取新镜像"
		} else if reason := u.checkWritable(); reason != "" {
			status.CanUpdate = false
			status.UpdateDisabledReason = reason
		}
	}

	u.mu.Lock()
	u.lastStatus = status
	u.mu.Unlock()

	return status, nil
}

func (u *Updater) Apply(ctx context.Context) error {
	if !u.StartUpdate() {
		return ErrAlreadyUpdating
	}
	return u.ApplyStarted(ctx)
}

func (u *Updater) ApplyStarted(ctx context.Context) error {
	defer func() {
		u.mu.Lock()
		u.isUpdating = false
		u.mu.Unlock()
	}()

	if u.isDocker {
		return fmt.Errorf("docker environment does not support in-place update")
	}

	status := u.GetLastStatus()
	if status == nil || !status.HasUpdate {
		return fmt.Errorf("no update available")
	}

	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("fetch release: %w", err)
	}
	if release.TagName != status.LatestVersion {
		return fmt.Errorf("latest release changed from %s to %s, please check update again", status.LatestVersion, release.TagName)
	}

	assetName := buildAssetName()
	var assetURL, checksumURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == assetName+".sha256" {
			checksumURL = a.BrowserDownloadURL
		}
	}
	if assetURL == "" {
		return fmt.Errorf("asset %s not found in release %s", assetName, release.TagName)
	}
	if checksumURL == "" {
		return fmt.Errorf("checksum asset %s.sha256 not found in release %s", assetName, release.TagName)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	checksum, err := u.fetchChecksum(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}

	log.Printf("[Updater-Download] 开始下载: %s", assetURL)
	body, err := u.downloadStream(ctx, assetURL)
	if err != nil {
		return fmt.Errorf("download asset: %w", err)
	}
	defer body.Close()

	bakPath := exePath + ".bak"
	if err := update.Apply(body, update.Options{
		TargetPath:  exePath,
		TargetMode:  0755,
		Checksum:    checksum,
		Hash:        crypto.SHA256,
		OldSavePath: bakPath,
	}); err != nil {
		if rbErr := update.RollbackError(err); rbErr != nil {
			return fmt.Errorf("update failed: %w, rollback also failed: %v", err, rbErr)
		}
		return fmt.Errorf("update failed: %w", err)
	}

	log.Printf("[Updater-Backup] 已备份当前二进制: %s", bakPath)
	log.Printf("[Updater-Success] 升级完成: %s -> %s，即将重启", u.currentVersion, release.TagName)

	go func() {
		time.Sleep(1 * time.Second)
		u.shutdownFunc()
	}()

	return nil
}
func (u *Updater) fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", apiBase, repoOwner, repoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func (u *Updater) fetchChecksum(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checksum file returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return nil, fmt.Errorf("checksum file is empty")
	}

	checksum, err := hex.DecodeString(fields[0])
	if err != nil {
		return nil, fmt.Errorf("invalid checksum format: %w", err)
	}
	return checksum, nil
}

func (u *Updater) downloadStream(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download returned %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (u *Updater) checkWritable() string {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("无法获取可执行文件路径: %v", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Sprintf("无法解析可执行文件路径: %v", err)
	}

	dir := filepath.Dir(exePath)
	testFile := filepath.Join(dir, ".ccx-update-test")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Sprintf("可执行文件目录无写入权限: %v", err)
	}
	f.Close()
	os.Remove(testFile)
	return ""
}

func buildAssetName() string {
	name := fmt.Sprintf("ccx-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func compareVersions(a, b string) int {
	a = normalizeVersion(a)
	b = normalizeVersion(b)

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		var av, bv int
		if i < len(aParts) {
			fmt.Sscanf(aParts[i], "%d", &av)
		}
		if i < len(bParts) {
			fmt.Sscanf(bParts[i], "%d", &bv)
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

func isUpgradeableVersion(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" || v == "unknown" || strings.Contains(v, "dev") {
		return false
	}
	return true
}

func normalizeVersion(v string) string {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}
	return v
}
