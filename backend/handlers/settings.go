package handlers

import (
	"encoding/json"
	"net/http"

	"dida/store"
)

// SettingsHandler 设置相关的HTTP请求处理器
type SettingsHandler struct {
	settingsStore *store.SettingsStore
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(ss *store.SettingsStore) *SettingsHandler {
	return &SettingsHandler{settingsStore: ss}
}

// HandleGet 获取当前设置
// GET /api/settings
func (h *SettingsHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	// 返回时隐藏 API Key（只返回是否有值）
	resp := map[string]interface{}{
		"novelPath":         settings.NovelPath,
		"aiConfigured":      settings.AIConfigured,
		"aiModel":           settings.AIModel,
		"endpoint":          settings.Endpoint,
		"hasApiKey":         settings.APIKey != "",
		"autoSave":          settings.AutoSave,
		"autoSaveMs":        settings.AutoSaveMs,
		"defaultFontSize":   settings.DefaultFontSize,
		"defaultLineSpacing": settings.DefaultLineSpacing,
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleUpdate 更新设置
// PUT /api/settings
func (h *SettingsHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	current := h.settingsStore.Get()

	// 只更新 JSON 中显式提供的字段，避免 bool 零值覆盖
	if v, ok := raw["novelPath"]; ok {
		json.Unmarshal(v, &current.NovelPath)
	}
	if v, ok := raw["endpoint"]; ok {
		json.Unmarshal(v, &current.Endpoint)
		current.AIConfigured = true
	}
	if v, ok := raw["apiKey"]; ok {
		json.Unmarshal(v, &current.APIKey)
		current.AIConfigured = true
	}
	if v, ok := raw["aiModel"]; ok {
		json.Unmarshal(v, &current.AIModel)
	}
	if v, ok := raw["autoSave"]; ok {
		json.Unmarshal(v, &current.AutoSave)
	}
	if v, ok := raw["autoSaveMs"]; ok {
		json.Unmarshal(v, &current.AutoSaveMs)
	}
	if v, ok := raw["defaultFontSize"]; ok {
		json.Unmarshal(v, &current.DefaultFontSize)
	}
	if v, ok := raw["defaultLineSpacing"]; ok {
		json.Unmarshal(v, &current.DefaultLineSpacing)
	}

	if err := h.settingsStore.Update(current); err != nil {
		writeError(w, http.StatusInternalServerError, "保存设置失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleGetAPIKey 获取API Key（用于编辑时展示掩码）
// GET /api/settings/apikey
func (h *SettingsHandler) HandleGetAPIKey(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	key := settings.APIKey
	// 返回掩码版本
	if len(key) > 8 {
		key = key[:4] + "****" + key[len(key)-4:]
	}
	writeJSON(w, http.StatusOK, map[string]string{"apiKey": key})
}
