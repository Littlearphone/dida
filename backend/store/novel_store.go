package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dida/models"

	"github.com/google/uuid"
)

// NovelStore 管理小说文件的持久化
// 每本小说存储在独立目录中，包含 meta.json 和 chapters/ 子目录
// 目录名格式: {书名}-{uuid}，章节文件名格式: {序号}-{标题}-{uuid}.json
type NovelStore struct {
	mu           sync.RWMutex
	novels       map[string]*models.Novel   // id -> novel 缓存
	chapters     map[string]*models.Chapter // id -> chapter 缓存
	novelPath    string                     // 小说存储根目录
	novelDirs    map[string]string          // novelID -> 实际目录完整路径
	chapterFiles map[string]string          // chapterID -> 实际文件完整路径
}

// NewNovelStore 创建小说存储器
func NewNovelStore(basePath string) *NovelStore {
	return &NovelStore{
		novels:       make(map[string]*models.Novel),
		chapters:     make(map[string]*models.Chapter),
		novelPath:    basePath,
		novelDirs:    make(map[string]string),
		chapterFiles: make(map[string]string),
	}
}

// sanitizeFilename 移除文件名中的非法字符（Windows: \ / : * ? " < > |）
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"\\", "", "/", "", ":", "", "*", "",
		"?", "", "\"", "", "<", "", ">", "", "|", "",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// novelDirName 生成小说目录名: {书名}-{uuid}
func (ns *NovelStore) novelDirName(novel *models.Novel) string {
	safeTitle := sanitizeFilename(novel.Title)
	if safeTitle == "" {
		safeTitle = "untitled"
	}
	return safeTitle + "-" + novel.ID
}

// chapterFileName 生成章节文件名: {序号}-{标题}-{uuid}.json
// 尾部附带 uuid 确保文件名始终可反解，避免同名覆盖
func (ns *NovelStore) chapterFileName(ch *models.Chapter) string {
	safeTitle := sanitizeFilename(ch.Title)
	if safeTitle == "" {
		safeTitle = "untitled"
	}
	return fmt.Sprintf("%04d-%s-%s.json", ch.Order, safeTitle, ch.ID)
}

// getNovelDir 获取小说目录的完整路径（优先从缓存读取，兼容旧版纯 UUID 目录名）
func (ns *NovelStore) getNovelDir(id string) string {
	if dir, ok := ns.novelDirs[id]; ok {
		return dir
	}
	return filepath.Join(ns.novelPath, id)
}

// LoadAll 从磁盘加载所有小说及章节文件路径
func (ns *NovelStore) LoadAll() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	entries, err := os.ReadDir(ns.novelPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(ns.novelPath, 0755)
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		novelDir := filepath.Join(ns.novelPath, entry.Name())
		novel, chapters, err := loadNovelDir(novelDir)
		if err != nil {
			continue // 跳过损坏的小说目录
		}
		ns.novels[novel.ID] = novel
		ns.novelDirs[novel.ID] = novelDir

		// 通过目录列表匹配章节文件路径，避免重复读取文件内容
		chaptersDir := filepath.Join(novelDir, "chapters")
		chEntries, err := os.ReadDir(chaptersDir)
		if err == nil {
			for _, chEntry := range chEntries {
				if chEntry.IsDir() {
					continue
				}
				name := chEntry.Name()
				// 匹配新格式 {order}-{title}-{uuid}.json 或旧格式 {uuid}.json
				for _, ch := range chapters {
					if strings.HasSuffix(name, ch.ID+".json") {
						ns.chapterFiles[ch.ID] = filepath.Join(chaptersDir, name)
						break
					}
				}
			}
		}

		// 注册章节到缓存
		for _, ch := range chapters {
			ns.chapters[ch.ID] = ch
		}
	}
	return nil
}

// loadNovelDir 从目录加载一本小说及其章节
func loadNovelDir(dir string) (*models.Novel, []*models.Chapter, error) {
	metaPath := filepath.Join(dir, "meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, nil, err
	}
	var novel models.Novel
	if err := json.Unmarshal(data, &novel); err != nil {
		return nil, nil, err
	}

	chaptersDir := filepath.Join(dir, "chapters")
	chapters := make([]*models.Chapter, 0)
	chEntries, err := os.ReadDir(chaptersDir)
	if err == nil {
		for _, chEntry := range chEntries {
			if chEntry.IsDir() {
				continue
			}
			chData, err := os.ReadFile(filepath.Join(chaptersDir, chEntry.Name()))
			if err != nil {
				continue
			}
			var ch models.Chapter
			if err := json.Unmarshal(chData, &ch); err != nil {
				continue
			}
			chapters = append(chapters, &ch)
		}
	}
	return &novel, chapters, nil
}

// CreateNovel 创建新小说
func (ns *NovelStore) CreateNovel(title, author string) (*models.Novel, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	now := time.Now()
	novel := &models.Novel{
		ID:         uuid.New().String(),
		Title:      title,
		Author:     author,
		ChapterIDs: make([]string, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := ns.saveNovelToDisk(novel); err != nil {
		return nil, err
	}
	ns.novels[novel.ID] = novel
	return novel, nil
}

// GetNovel 通过ID获取小说
func (ns *NovelStore) GetNovel(id string) *models.Novel {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.novels[id]
}

// ListNovels 返回所有小说列表（按更新时间倒序）
func (ns *NovelStore) ListNovels() []*models.Novel {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	list := make([]*models.Novel, 0, len(ns.novels))
	for _, n := range ns.novels {
		list = append(list, n)
	}
	// 按更新时间倒序排列
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].UpdatedAt.After(list[i].UpdatedAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
	return list
}

// DeleteNovel 删除小说及其所有文件
func (ns *NovelStore) DeleteNovel(id string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	novel, ok := ns.novels[id]
	if !ok {
		return nil
	}

	// 清理章节缓存和文件路径记录
	for _, chID := range novel.ChapterIDs {
		delete(ns.chapters, chID)
		delete(ns.chapterFiles, chID)
	}

	// 删除磁盘目录
	if dir, ok := ns.novelDirs[id]; ok {
		os.RemoveAll(dir)
		delete(ns.novelDirs, id)
	} else {
		// 兼容旧版纯 UUID 目录名
		os.RemoveAll(filepath.Join(ns.novelPath, id))
	}

	delete(ns.novels, id)
	return nil
}

// UpdateNovel 更新小说元数据（含标题变更时自动重命名目录）
func (ns *NovelStore) UpdateNovel(novel *models.Novel) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	novel.UpdatedAt = time.Now()
	if err := ns.saveNovelToDisk(novel); err != nil {
		return err
	}
	ns.novels[novel.ID] = novel
	return nil
}

// ReorderChapters 更新小说的章节顺序（全量替换 ChapterIDs）
func (ns *NovelStore) ReorderChapters(novelID string, chapterIDs []string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	novel, ok := ns.novels[novelID]
	if !ok {
		return nil
	}
	novel.ChapterIDs = chapterIDs
	novel.UpdatedAt = time.Now()
	if err := ns.saveNovelToDisk(novel); err != nil {
		return err
	}
	ns.novels[novelID] = novel
	return nil
}

// saveNovelToDisk 将小说元数据写入磁盘，目录名格式 {书名}-{uuid}
// 标题变更时自动重命名目录并更新 novelDirs 缓存
func (ns *NovelStore) saveNovelToDisk(novel *models.Novel) error {
	targetDir := filepath.Join(ns.novelPath, ns.novelDirName(novel))

	// 检测目录是否需要重命名（标题变更）
	if oldDir, ok := ns.novelDirs[novel.ID]; ok && oldDir != targetDir {
		// 重命名前先确保父目录存在
		if err := os.MkdirAll(ns.novelPath, 0755); err != nil {
			return err
		}
		if err := os.Rename(oldDir, targetDir); err != nil {
			return fmt.Errorf("重命名小说目录失败: %w", err)
		}
		// 更新该小说下所有章节文件的缓存路径
		for id, oldPath := range ns.chapterFiles {
			ch, ok := ns.chapters[id]
			if ok && ch.NovelID == novel.ID {
				ns.chapterFiles[id] = filepath.Join(targetDir, "chapters", filepath.Base(oldPath))
			}
		}
	}

	if err := os.MkdirAll(filepath.Join(targetDir, "chapters"), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(novel, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(targetDir, "meta.json"), data, 0644); err != nil {
		return err
	}

	ns.novelDirs[novel.ID] = targetDir
	return nil
}

// CreateChapter 创建新章节
func (ns *NovelStore) CreateChapter(novelID, title, content string, order int) (*models.Chapter, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	novel, ok := ns.novels[novelID]
	if !ok {
		return nil, nil // 小说不存在
	}

	now := time.Now()
	ch := &models.Chapter{
		ID:        uuid.New().String(),
		NovelID:   novelID,
		Title:     title,
		Content:   content,
		Order:     order,
		WordCount: int64(len([]rune(content))),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := ns.saveChapterToDisk(ch); err != nil {
		return nil, err
	}

	// 更新小说的章节列表
	novel.ChapterIDs = append(novel.ChapterIDs, ch.ID)
	novel.ChapterIDs = reorderChapterIDs(novel.ChapterIDs, ch.ID, order)
	novel.WordCount += ch.WordCount
	novel.UpdatedAt = now
	ns.saveNovelToDisk(novel)

	ns.chapters[ch.ID] = ch
	return ch, nil
}

// reorderChapterIDs 将章节ID插入到正确的位置
func reorderChapterIDs(ids []string, id string, order int) []string {
	// 简单实现：先移除再按顺序插入
	result := make([]string, 0)
	for _, existing := range ids {
		if existing == id {
			continue
		}
		result = append(result, existing)
	}

	// 按order插入到正确位置
	if order <= 0 {
		return append([]string{id}, result...)
	}
	if order >= len(result) {
		return append(result, id)
	}
	newResult := make([]string, 0, len(result)+1)
	newResult = append(newResult, result[:order]...)
	newResult = append(newResult, id)
	newResult = append(newResult, result[order:]...)
	return newResult
}

// GetChapter 获取章节
func (ns *NovelStore) GetChapter(id string) *models.Chapter {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.chapters[id]
}

// GetChaptersByNovel 获取小说的所有章节（按顺序）
func (ns *NovelStore) GetChaptersByNovel(novelID string) []*models.Chapter {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	novel, ok := ns.novels[novelID]
	if !ok {
		return nil
	}

	chapters := make([]*models.Chapter, 0, len(novel.ChapterIDs))
	for _, chID := range novel.ChapterIDs {
		if ch, ok := ns.chapters[chID]; ok {
			chapters = append(chapters, ch)
		}
	}
	return chapters
}

// UpdateChapter 更新章节内容（标题/序号变更时自动重命名文件）
func (ns *NovelStore) UpdateChapter(chapter *models.Chapter) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if _, ok := ns.chapters[chapter.ID]; !ok {
		return nil
	}

	oldWordCount := ns.chapters[chapter.ID].WordCount
	chapter.WordCount = int64(len([]rune(chapter.Content)))
	chapter.UpdatedAt = time.Now()

	if err := ns.saveChapterToDisk(chapter); err != nil {
		return err
	}

	// 更新小说的总字数
	if novel, ok := ns.novels[chapter.NovelID]; ok {
		novel.WordCount += chapter.WordCount - oldWordCount
		novel.UpdatedAt = time.Now()
		ns.saveNovelToDisk(novel)
	}

	ns.chapters[chapter.ID] = chapter
	return nil
}

// DeleteChapter 删除章节
func (ns *NovelStore) DeleteChapter(id string) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ch, ok := ns.chapters[id]
	if !ok {
		return nil
	}

	// 从小说章节列表中移除
	if novel, ok := ns.novels[ch.NovelID]; ok {
		newIDs := make([]string, 0, len(novel.ChapterIDs)-1)
		for _, cid := range novel.ChapterIDs {
			if cid != id {
				newIDs = append(newIDs, cid)
			}
		}
		novel.ChapterIDs = newIDs
		novel.WordCount -= ch.WordCount
		novel.UpdatedAt = time.Now()
		ns.saveNovelToDisk(novel)
	}

	// 删除文件（优先使用缓存路径，兼容旧版纯 UUID 文件名）
	chFile := filepath.Join(ns.novelPath, ch.NovelID, "chapters", ch.ID+".json")
	if f, ok := ns.chapterFiles[ch.ID]; ok {
		chFile = f
	}
	os.Remove(chFile)
	delete(ns.chapterFiles, id)

	delete(ns.chapters, id)
	return nil
}

// saveChapterToDisk 将章节写入磁盘，文件名格式 {序号}-{标题}-{uuid}.json
// 标题或序号变更时自动清理旧文件
func (ns *NovelStore) saveChapterToDisk(ch *models.Chapter) error {
	novelDir := ns.getNovelDir(ch.NovelID)
	chaptersDir := filepath.Join(novelDir, "chapters")
	targetFile := filepath.Join(chaptersDir, ns.chapterFileName(ch))

	// 如果文件名变了，删除旧文件
	if oldFile, ok := ns.chapterFiles[ch.ID]; ok && oldFile != targetFile {
		if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	data, err := json.MarshalIndent(ch, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(targetFile, data, 0644); err != nil {
		return err
	}

	ns.chapterFiles[ch.ID] = targetFile
	return nil
}

// ImportNovel 导入小说（创建结构并保存）
func (ns *NovelStore) ImportNovel(title string, chapters []*models.Chapter) (*models.Novel, error) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	now := time.Now()
	novel := &models.Novel{
		ID:         uuid.New().String(),
		Title:      title,
		ChapterIDs: make([]string, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 先写小说meta
	if err := ns.saveNovelToDisk(novel); err != nil {
		return nil, err
	}

	// 再逐个写章节
	totalWords := int64(0)
	for i, ch := range chapters {
		ch.ID = uuid.New().String()
		ch.NovelID = novel.ID
		ch.Order = i + 1
		ch.WordCount = int64(len([]rune(ch.Content)))
		ch.CreatedAt = now
		ch.UpdatedAt = now
		totalWords += ch.WordCount

		if err := ns.saveChapterToDisk(ch); err != nil {
			return nil, err
		}
		novel.ChapterIDs = append(novel.ChapterIDs, ch.ID)
		ns.chapters[ch.ID] = ch
	}

	novel.WordCount = totalWords
	ns.saveNovelToDisk(novel)
	ns.novels[novel.ID] = novel
	return novel, nil
}
