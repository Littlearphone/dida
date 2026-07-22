package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"dida/models"
	"dida/store"
)

// 编译时缓存 HTML 标签正则
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// stripHTML 去除 HTML 标签，将段落结构转为换行，解码常见实体
func stripHTML(html string) string {
	s := strings.ReplaceAll(html, "</p>", "\n\n") // 段落结束 → 双换行保留分段
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = htmlTagRegex.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	// 清除行尾空白
	spaceRe := regexp.MustCompile(`[ \t]+\n`)
	s = spaceRe.ReplaceAllString(s, "\n")
	// 最多保留两行空行
	multiNewline := regexp.MustCompile(`\n{3,}`)
	s = multiNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// NovelHandler 小说相关的HTTP请求处理器
type NovelHandler struct {
	novelStore *store.NovelStore
}

// NewNovelHandler 创建小说处理器
func NewNovelHandler(ns *store.NovelStore) *NovelHandler {
	return &NovelHandler{novelStore: ns}
}

// HandleList 返回所有小说列表
// GET /api/novels
func (h *NovelHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	novels := h.novelStore.ListNovels()
	writeJSON(w, http.StatusOK, novels)
}

// HandleCreate 创建新小说
// POST /api/novels
func (h *NovelHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title  string `json:"title"`
		Author string `json:"author"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "小说标题不能为空")
		return
	}

	novel, err := h.novelStore.CreateNovel(req.Title, req.Author)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建小说失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, novel)
}

// HandleGet 获取单个小说详情（含章节）
// GET /api/novels/{id}
func (h *NovelHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	novel := h.novelStore.GetNovel(id)
	if novel == nil {
		writeError(w, http.StatusNotFound, "小说不存在")
		return
	}
	writeJSON(w, http.StatusOK, novel)
}

// HandleUpdate 更新小说元数据（大纲、角色、关系、事件、简介等）
// PUT /api/novels/{id}
func (h *NovelHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	novel := h.novelStore.GetNovel(id)
	if novel == nil {
		writeError(w, http.StatusNotFound, "小说不存在")
		return
	}

	var req struct {
		Title         *string                    `json:"title,omitempty"`
		Outline       *string                    `json:"outline,omitempty"`
		Description   *string                    `json:"description,omitempty"`
		Characters    []models.Character         `json:"characters,omitempty"`
		Relationships []models.NovelRelationship `json:"relationships,omitempty"`
		Events        []models.Event             `json:"events,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.Title != nil {
		novel.Title = *req.Title
	}
	if req.Outline != nil {
		novel.Outline = *req.Outline
	}
	if req.Description != nil {
		novel.Description = *req.Description
	}
	if req.Characters != nil {
		novel.Characters = req.Characters
	}
	if req.Relationships != nil {
		novel.Relationships = req.Relationships
	}
	if req.Events != nil {
		novel.Events = req.Events
	}

	if err := h.novelStore.UpdateNovel(novel); err != nil {
		writeError(w, http.StatusInternalServerError, "更新失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, novel)
}

// HandleDelete 删除小说
// DELETE /api/novels/{id}
func (h *NovelHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.novelStore.DeleteNovel(id); err != nil {
		writeError(w, http.StatusInternalServerError, "删除失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleImport 导入小说（含AI拆分）
// POST /api/novels/import
func (h *NovelHandler) HandleImport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Content     string `json:"content"`     // 完整的原始内容
		SkipAISplit bool   `json:"skipAISplit"` // 是否跳过AI拆分
		Chapters    []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"chapters,omitempty"` // 如果跳过AI拆分，直接传入章节
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	var chapters []*models.Chapter

	if req.SkipAISplit && len(req.Chapters) > 0 {
		// 用户手动提供了章节列表
		for _, ch := range req.Chapters {
			chapters = append(chapters, &models.Chapter{
				Title:   ch.Title,
				Content: ch.Content,
			})
		}
	} else if !req.SkipAISplit && req.Content != "" {
		// 前端会先调用AI拆分，然后传拆分好的结果过来
		// 这里的逻辑由前端控制
		writeError(w, http.StatusBadRequest, "请先通过AI接口拆分章节后再导入")
		return
	} else {
		writeError(w, http.StatusBadRequest, "请提供章节内容")
		return
	}

	novel, err := h.novelStore.ImportNovel(req.Title, chapters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "导入失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, novel)
}

// HandleGetChapters 获取小说所有章节
// GET /api/novels/{id}/chapters
func (h *NovelHandler) HandleGetChapters(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	chapters := h.novelStore.GetChaptersByNovel(id)
	if chapters == nil {
		writeError(w, http.StatusNotFound, "小说不存在")
		return
	}
	writeJSON(w, http.StatusOK, chapters)
}

// HandleReorderChapters 批量重排章节顺序
// PUT /api/novels/{id}/chapters/reorder
func (h *NovelHandler) HandleReorderChapters(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		ChapterIDs []string `json:"chapterIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if err := h.novelStore.ReorderChapters(id, req.ChapterIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "重排失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleExport 导出整本小说为纯文本
// GET /api/novels/{id}/export?format=txt
func (h *NovelHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	novel := h.novelStore.GetNovel(id)
	if novel == nil {
		writeError(w, http.StatusNotFound, "小说不存在")
		return
	}

	chapters := h.novelStore.GetChaptersByNovel(id)
	if len(chapters) == 0 {
		writeError(w, http.StatusBadRequest, "没有可导出的章节")
		return
	}

	format := r.URL.Query().Get("format")
	var content, filename, mimeType string

	switch format {
	case "markdown", "md":
		mimeType = "text/markdown; charset=utf-8"
		filename = sanitizeFilename(novel.Title) + ".md"
		content = exportMarkdown(novel, chapters)
	default:
		mimeType = "text/plain; charset=utf-8"
		filename = sanitizeFilename(novel.Title) + ".txt"
		content = exportPlainText(novel, chapters)
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	w.Write([]byte(content))
}

// sanitizeFilename 去除文件名中的非法字符
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"\\", "", "/", "", ":", "", "*", "",
		"?", "", "\"", "", "<", "", ">", "", "|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// exportPlainText 生成纯文本格式的完整小说内容
func exportPlainText(novel *models.Novel, chapters []*models.Chapter) string {
	var b strings.Builder
	sep := strings.Repeat("━", 48)

	// 小说标题上下有分割线
	b.WriteString(sep + "\n")
	b.WriteString(novel.Title + "\n")
	b.WriteString(sep + "\n")
	b.WriteString("\n")
	if novel.Author != "" {
		b.WriteString("作者：" + novel.Author + "\n")
	}
	if novel.Description != "" {
		b.WriteString("简介：" + novel.Description + "\n")
	}
	if novel.Outline != "" {
		b.WriteString("大纲：" + novel.Outline + "\n")
	}
	b.WriteString("\n")

	for _, ch := range chapters {
		// 标题：带上章节序号，上下各有一条分割线
		title := ch.Title
		if title == "" {
			title = fmt.Sprintf("第%d章", ch.Order)
		} else {
			title = fmt.Sprintf("第%d章 %s", ch.Order, title)
		}
		b.WriteString(sep + "\n")
		b.WriteString(title + "\n")
		b.WriteString(sep + "\n\n")
		b.WriteString(stripHTML(ch.Content) + "\n\n\n")
	}

	return b.String()
}

// exportMarkdown 生成 Markdown 格式的完整小说内容
func exportMarkdown(novel *models.Novel, chapters []*models.Chapter) string {
	var b strings.Builder

	b.WriteString("# " + novel.Title + "\n\n")
	if novel.Author != "" {
		b.WriteString("**作者：** " + novel.Author + "\n\n")
	}
	if novel.Description != "" {
		b.WriteString("> " + novel.Description + "\n\n")
	}
	b.WriteString("---\n\n")

	for _, ch := range chapters {
		// 标题：带上章节序号，前面有分割线
		title := ch.Title
		if title == "" {
			title = fmt.Sprintf("第%d章", ch.Order)
		} else {
			title = fmt.Sprintf("第%d章 %s", ch.Order, title)
		}
		b.WriteString("---\n\n")
		b.WriteString("## " + title + "\n\n")
		b.WriteString(stripHTML(ch.Content) + "\n\n")
	}

	return b.String()
}

// 通用JSON响应写入
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// 通用错误响应写入
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
