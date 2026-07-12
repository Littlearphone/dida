package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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
// 支持传入已有元数据进行增量提取
func (h *AIHandler) HandleExtractInfo(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsStore.Get()
	if settings.APIKey == "" {
		writeError(w, http.StatusBadRequest, "请先配置AI接口")
		return
	}

	var req struct {
		Chapters           []models.Chapter           `json:"chapters"`
		FullContent        string                     `json:"fullContent"`
		ExistingOutline    string                     `json:"existingOutline"`
		ExistingCharacters []models.Character         `json:"existingCharacters"`
		ExistingRelations  []models.NovelRelationship `json:"existingRelations"`
		ExistingEvents     []models.Event             `json:"existingEvents"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	client := h.createClient()
	result, err := client.ExtractNovelInfo(req.Chapters, req.FullContent,
		req.ExistingOutline, req.ExistingCharacters, req.ExistingRelations, req.ExistingEvents)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI提取失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleEditSSE 统一的 SSE 流式处理模板
// execStream 是一个闭包，由各 handler 传入对应的 Stream 方法调用
func (h *AIHandler) handleEditSSE(w http.ResponseWriter, execStream func(func(string)) (string, error), logOp string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "不支持流式响应")
		return
	}

	fullText, err := execStream(func(chunk string) {
		data := fmt.Sprintf(`{"text":%s}`, jsonEncodeString(chunk))
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	})

	if err != nil {
		log.Printf("[API] 流式%s错误: %v", logOp, err)
		errData := fmt.Sprintf(`{"error":"%s"}`, jsonEncodeString(err.Error()))
		fmt.Fprintf(w, "data: %s\n\n", errData)
	}
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	if err == nil {
		log.Printf("[API] 流式%s完成 | 输出字数=%d", logOp, len([]rune(fullText)))
	}
}

// jsonEncodeString 对字符串进行 JSON 编码（转义特殊字符）
func jsonEncodeString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// HandleContinueWrite AI续写
// POST /api/ai/continue-write
// 支持流式响应（SSE）和非流式响应，取决于 Accept 头
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

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		h.handleEditSSE(w, func(onChunk func(string)) (string, error) {
			return client.ContinueWriteStream(req.ChapterContent, req.PreviousChapterContent, req.Outline, req.Requirement, req.Characters, req.Relationships, req.Events, onChunk)
		}, "续写")
		return
	}

	result, err := client.ContinueWrite(req.ChapterContent, req.PreviousChapterContent, req.Outline, req.Requirement, req.Characters, req.Relationships, req.Events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI续写失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"result": result})
}

// HandlePolish AI润色
// POST /api/ai/polish
// 支持流式响应（SSE）和非流式响应
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

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		h.handleEditSSE(w, func(onChunk func(string)) (string, error) {
			return client.PolishStream(req.Content, req.IsSelection, req.Outline, req.Requirement, req.PreviousChapterContent, req.Characters, req.Relationships, req.Events, onChunk)
		}, "润色")
		return
	}

	result, err := client.Polish(req.Content, req.IsSelection, req.Outline, req.Requirement, req.PreviousChapterContent, req.Characters, req.Relationships, req.Events)
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
// 支持流式响应（SSE）和非流式响应
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

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		h.handleEditSSE(w, func(onChunk func(string)) (string, error) {
			return client.ExpandStream(req.Content, req.IsSelection, req.Outline, req.Requirement, req.PreviousChapterContent, req.Characters, req.Relationships, req.Events, onChunk)
		}, "扩写")
		return
	}

	result, err := client.Expand(req.Content, req.IsSelection, req.Outline, req.Requirement, req.PreviousChapterContent, req.Characters, req.Relationships, req.Events)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI扩写失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models.AIResult{
		Original: req.Content,
		Result:   result,
	})
}
