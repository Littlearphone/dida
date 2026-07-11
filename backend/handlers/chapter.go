package handlers

import (
	"encoding/json"
	"net/http"

	"dida/store"
)

// ChapterHandler 章节相关的HTTP请求处理器
type ChapterHandler struct {
	novelStore *store.NovelStore
}

// NewChapterHandler 创建章节处理器
func NewChapterHandler(ns *store.NovelStore) *ChapterHandler {
	return &ChapterHandler{novelStore: ns}
}

// HandleCreate 创建新章节
// POST /api/chapters
func (h *ChapterHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NovelID string `json:"novelId"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Order   int    `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	ch, err := h.novelStore.CreateChapter(req.NovelID, req.Title, req.Content, req.Order)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建章节失败: "+err.Error())
		return
	}
	if ch == nil {
		writeError(w, http.StatusNotFound, "小说不存在")
		return
	}
	writeJSON(w, http.StatusCreated, ch)
}

// HandleGet 获取单个章节
// GET /api/chapters/{id}
func (h *ChapterHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ch := h.novelStore.GetChapter(id)
	if ch == nil {
		writeError(w, http.StatusNotFound, "章节不存在")
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

// HandleUpdate 更新章节内容
// PUT /api/chapters/{id}
func (h *ChapterHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.novelStore.GetChapter(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "章节不存在")
		return
	}

	var req struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Content != "" || r.ContentLength > 0 {
		// 允许清空内容
		existing.Content = req.Content
	}

	if err := h.novelStore.UpdateChapter(existing); err != nil {
		writeError(w, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}

// HandleDelete 删除章节
// DELETE /api/chapters/{id}
func (h *ChapterHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.novelStore.DeleteChapter(id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleAutoSave 自动保存章节（在内容无变化若干秒后调用）
// PUT /api/chapters/{id}/autosave
func (h *ChapterHandler) HandleAutoSave(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	existing := h.novelStore.GetChapter(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "章节不存在")
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	existing.Content = req.Content
	if err := h.novelStore.UpdateChapter(existing); err != nil {
		writeError(w, http.StatusInternalServerError, "自动保存失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, existing)
}
