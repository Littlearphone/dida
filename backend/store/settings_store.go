package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"dida/models"
)

// 默认设置
var defaultSettings = models.Settings{
	NovelPath:    getDefaultNovelPath(),
	AIModel:      "deepseek-chat",
	Endpoint:     "https://api.deepseek.com",
	AutoSave:     true,
	AutoSaveMs:   3000,
	DefaultFontSize: 16,
	DefaultLineSpacing: 1.8,
}

// SettingsStore 管理应用设置的持久化
type SettingsStore struct {
	mu       sync.RWMutex
	settings models.Settings
	filePath string // 设置文件的存储路径
}

// NewSettingsStore 创建设置存储器
// 在用户目录下创建 .dida 文件夹存放配置文件
func NewSettingsStore() (*SettingsStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir, err = os.UserHomeDir()
		if err != nil {
			configDir = "."
		}
	}
	didaDir := filepath.Join(configDir, ".dida")
	if err := os.MkdirAll(didaDir, 0755); err != nil {
		return nil, err
	}

	s := &SettingsStore{
		settings: defaultSettings,
		filePath: filepath.Join(didaDir, "settings.json"),
	}
	if err := s.load(); err != nil {
		// 加载失败则写入默认设置
		s.settings = defaultSettings
		s.save()
	}
	return s, nil
}

// 获取默认小说保存路径
func getDefaultNovelPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	path := filepath.Join(home, "Documents", "DidaNovels")
	os.MkdirAll(path, 0755)
	return path
}

// Get 返回当前设置的副本
func (s *SettingsStore) Get() models.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// Update 原子性地更新设置
func (s *SettingsStore) Update(settings models.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	return s.save()
}

// save 将设置写入JSON文件
func (s *SettingsStore) save() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}

// load 从JSON文件读取设置
func (s *SettingsStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.settings)
}
