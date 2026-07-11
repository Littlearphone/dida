package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"dida/ai"
	"dida/models"
	"dida/store"
)

// AIHandler AI功能相关的HTTP请求处理器
type AIHandler struct {
	settingsStore *store.SettingsStore
}

// NewAIHandler 创建AI处理器
func NewAIHandler(ss *store.SettingsStore) *AIHandler {
	return &AIHandler{settingsStore: ss}
}

// createClient 根据当前设置创建AI客户端
func (h *AIHandler) createClient() *ai.DeepSeekClient {
	settings := h.settingsStore.Get()
	return ai.NewDeepSeekClient(settings.Endpoint, settings.APIKey, settings.AIModel)
}

// HandleCheck 检查AI连接状态
// GET /api/ai/status
func (h *AIHandler) HandleCheck(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	configured := settings.APIKey != "" && settings.Endpoint != ""
	status := map[string]interface{}{
		"configured": configured,
		"model":      settings.AIModel,
		"endpoint":   settings.Endpoint,
	}
	if configured {
		client := h.createClient()
		if err := client.CheckConnection(); err != nil {
			status["connected"] = false
			status["error"] = err.Error()
		} else {
			status["connected"] = true
		}
	}
	writeJSON(w, http.StatusOK, status)
}

// HandleSplitChapters 使用AI拆分章节
// POST /api/ai/split-chapters
func (h *AIHandler) HandleSplitChapters(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req models.ChapterSplitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "内容不能为空")
		return
	}

	log.Printf("[API] AI拆分章节 | 内容长度=%d 字符 | endpoint=%s | model=%s",
		len([]rune(req.Content)), settings.Endpoint, settings.AIModel)

	client := h.createClient()
	result, err := client.SplitChapters(req.Content)
	if err != nil {
		log.Printf("[API] AI拆分失败: %v", err)
		writeError(w, http.StatusInternalServerError, "AI拆分失败: "+err.Error())
		return
	}
	log.Printf("[API] AI拆分完成 | 章节数=%d | 角色数=%d", len(result.Chapters), len(result.Characters))
	writeJSON(w, http.StatusOK, result)
}

// HandleExtractInfo 提取小说信息（大纲、角色、关系、事件）
// POST /api/ai/extract-info
func (h *AIHandler) HandleExtractInfo(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req struct {
		Chapters    []models.Chapter `json:"chapters"`
		FullContent string           `json:"fullContent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	client := h.createClient()
	result, err := client.ExtractNovelInfo(req.Chapters, req.FullContent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI提取失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// HandleContinueWrite AI续写
// POST /api/ai/continue-write
func (h *AIHandler) HandleContinueWrite(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req models.ContinueWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	client := h.createClient()
	result, err := client.ContinueWrite(req.ChapterContent, req.Outline, req.Requirement)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI续写失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}

// HandlePolish AI润色
// POST /api/ai/polish
func (h *AIHandler) HandlePolish(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req models.PolishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	client := h.createClient()
	result, err := client.Polish(req.Content, req.IsSelection, req.Outline, req.Requirement)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI润色失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.AIResult{
		Original: req.Content,
		Result:   result,
	})
}

// HandleExpand AI扩写
// POST /api/ai/expand
func (h *AIHandler) HandleExpand(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req models.ExpandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	client := h.createClient()
	result, err := client.Expand(req.Content, req.IsSelection, req.Outline, req.Requirement)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI扩写失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.AIResult{
		Original: req.Content,
		Result:   result,
	})
}
