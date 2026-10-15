package utils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"html"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/russross/blackfriday.v2"
)

var Document = NewDocument("./data", "./data/markdowns")

const (
	Document_Default_FileName = "README"
	Document_Page_Suffix      = ".md"
)

const (
	Document_Type_Page = 1
	Document_Type_Dir  = 2
)

func NewDocument(documentAbsDir string, markdownAbsDir string) *document {
	return &document{
		DocumentAbsDir: documentAbsDir,
		MarkdownAbsDir: markdownAbsDir,
	}
}

type document struct {
	DocumentAbsDir string
	MarkdownAbsDir string
	lock           sync.Mutex
}

// get document page file by parentPath
func (d *document) GetPageFileByParentPath(name string, docType int, parentPath string) (pageFile string) {
	if docType == Document_Type_Page {
		pageFile = fmt.Sprintf("%s/%s%s", parentPath, name, Document_Page_Suffix)
	} else {
		pageFile = fmt.Sprintf("%s/%s/%s%s", parentPath, name, Document_Default_FileName, Document_Page_Suffix)
	}
	return
}

//get document path by spaceName
func (d *document) GetDefaultPageFileBySpaceName(name string) string {
	return fmt.Sprintf("%s/%s%s", name, Document_Default_FileName, Document_Page_Suffix)
}

// get document abs pageFile
func (d *document) GetAbsPageFileByPageFile(pageFile string) string {
	return d.MarkdownAbsDir + "/" + pageFile
}

// get document content by pageFile
func (d *document) GetContentByPageFile(pageFile string) (content string, err error) {
	return File.GetFileContents(d.GetAbsPageFileByPageFile(pageFile))
}

// create document
func (d *document) Create(pageFile string) error {
	if pageFile == "" {
		return nil
	}
	d.lock.Lock()
	absFilePath := d.GetAbsPageFileByPageFile(pageFile)
	absDir := filepath.Dir(absFilePath)
	err := os.MkdirAll(absDir, 0777)
	if err != nil {
		d.lock.Unlock()
		return err
	}
	d.lock.Unlock()
	return File.CreateFile(absFilePath)
}

// create and write document
func (d *document) CreateAndWrite(pageFile string, content string) error {
	if pageFile == "" {
		return nil
	}
	d.lock.Lock()
	absFilePath := d.GetAbsPageFileByPageFile(pageFile)
	absDir := filepath.Dir(absFilePath)
	err := os.MkdirAll(absDir, 0777)
	if err != nil {
		d.lock.Unlock()
		return err
	}
	d.lock.Unlock()
	return File.WriteFile(absFilePath, content)
}

// replace document content
func (d *document) Replace(pageFile string, content string) error {
	if pageFile == "" {
		return nil
	}
	d.lock.Lock()
	absFilePath := d.GetAbsPageFileByPageFile(pageFile)
	absDir := filepath.Dir(absFilePath)
	err := os.MkdirAll(absDir, 0777)
	if err != nil {
		d.lock.Unlock()
		return err
	}
	d.lock.Unlock()
	return ioutil.WriteFile(absFilePath, []byte(content), os.ModePerm)
}

// update document
func (d *document) Update(oldPageFile string, name string, content string, docType int, nameIsChange bool) (err error) {

	d.lock.Lock()
	defer d.lock.Unlock()

	absOldPageFile := d.GetAbsPageFileByPageFile(oldPageFile)

	err = ioutil.WriteFile(absOldPageFile, []byte(content), os.ModePerm)
	if err != nil {
		return
	}
	if nameIsChange {
		filePath := filepath.Dir(absOldPageFile)
		if docType == Document_Type_Page {
			err = os.Rename(absOldPageFile, filePath+"/"+name+Document_Page_Suffix)
		} else {
			err = os.Rename(filePath, filepath.Dir(filePath)+"/"+name)
		}
		if err != nil {
			return
		}
	}
	return nil
}

func (d *document) UpdateSpaceName(oldSpaceName string, newName string) error {

	d.lock.Lock()
	defer d.lock.Unlock()

	spaceOldDir := d.GetAbsPageFileByPageFile(oldSpaceName)
	spaceNewDir := d.GetAbsPageFileByPageFile(newName)
	if spaceNewDir == spaceOldDir {
		return nil
	}
	err := os.Rename(spaceOldDir, spaceNewDir)
	return err
}

// delete document
func (d *document) Delete(path string, docType int) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	absPageFile := d.GetAbsPageFileByPageFile(path)

	ok, _ := File.PathIsExists(absPageFile)
	if !ok {
		return nil
	}
	if docType == Document_Type_Page {
		return os.Remove(absPageFile)
	}

	return os.RemoveAll(filepath.Dir(absPageFile))
}

func (d *document) DeleteSpace(name string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	absSpaceDir := d.GetAbsPageFileByPageFile(name)

	ok, _ := File.PathIsExists(absSpaceDir)
	if !ok {
		return nil
	}

	return os.RemoveAll(absSpaceDir)
}

func (d *document) Move(movePath string, targetPath string, docType int) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	absOldPageFile := d.GetAbsPageFileByPageFile(movePath)
	absTargetPageFile := d.GetAbsPageFileByPageFile(targetPath)

	if docType == Document_Type_Page {
		return os.Rename(absOldPageFile, absTargetPageFile)
	}
	return os.Rename(filepath.Dir(absOldPageFile), filepath.Dir(absTargetPageFile))
}

// delete document attachment
func (d *document) DeleteAttachment(attachments []map[string]string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if len(attachments) == 0 {
		return nil
	}
	// delete attachment file
	for _, attachment := range attachments {
		if len(attachment) == 0 || attachment["path"] == "" {
			continue
		}
		file := filepath.Join(d.DocumentAbsDir, attachment["path"])
		_ = os.Remove(file)
	}
	return nil
}

// GetSnapshotFilePath returns the snapshot file path, isolated under data/history/{spaceId}/{documentId}/{logDocumentId}.md.gz
// It completely avoids polluting data/markdowns, and seamlessly migrates legacy files if found.
func (d *document) GetSnapshotFilePath(spaceId string, documentId string, logDocumentId string) string {
	newPath := filepath.Join(d.DocumentAbsDir, "history", spaceId, documentId, fmt.Sprintf("%s.md.gz", logDocumentId))
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	// Check legacy path and migrate if present
	legacyPath := filepath.Join(d.MarkdownAbsDir, spaceId, "history", documentId, fmt.Sprintf("%s.md.gz", logDocumentId))
	if _, err := os.Stat(legacyPath); err == nil {
		_ = os.MkdirAll(filepath.Dir(newPath), 0777)
		if err := os.Rename(legacyPath, newPath); err == nil {
			_ = os.Remove(filepath.Dir(legacyPath))
			_ = os.Remove(filepath.Join(d.MarkdownAbsDir, spaceId, "history"))
			_ = os.Remove(filepath.Join(d.MarkdownAbsDir, spaceId))
			return newPath
		}
		return legacyPath
	}
	return newPath
}

// SaveSnapshot compresses content with gzip and saves it in data/history/{spaceId}/{documentId}/{logDocumentId}.md.gz
// It automatically maintains a maximum retention of maxKeep (default 30) revisions per document.
func (d *document) SaveSnapshot(spaceId string, documentId string, logDocumentId string, content string) error {
	if content == "" || spaceId == "" || documentId == "" || logDocumentId == "" {
		return nil
	}
	d.lock.Lock()
	defer d.lock.Unlock()

	historyDir := filepath.Join(d.DocumentAbsDir, "history", spaceId, documentId)
	if err := os.MkdirAll(historyDir, 0777); err != nil {
		return err
	}

	snapshotFile := filepath.Join(historyDir, fmt.Sprintf("%s.md.gz", logDocumentId))

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write([]byte(content)); err != nil {
		_ = gzWriter.Close()
		return err
	}
	if err := gzWriter.Close(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(snapshotFile, buf.Bytes(), 0644); err != nil {
		return err
	}

	// Retention policy: Keep at most 30 snapshots per document
	d.cleanOldSnapshots(historyDir, 30)
	return nil
}

func (d *document) cleanOldSnapshots(dir string, maxKeep int) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil || len(entries) <= maxKeep {
		return
	}
	// Sort by ModTime ascending (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ModTime().Before(entries[j].ModTime())
	})
	deleteCount := len(entries) - maxKeep
	for i := 0; i < deleteCount; i++ {
		_ = os.Remove(filepath.Join(dir, entries[i].Name()))
	}
}

// GetSnapshotContent reads and decompresses a gzipped snapshot
func (d *document) GetSnapshotContent(spaceId string, documentId string, logDocumentId string) (string, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	snapshotFile := d.GetSnapshotFilePath(spaceId, documentId, logDocumentId)
	data, err := ioutil.ReadFile(snapshotFile)
	if err != nil {
		return "", err
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	decompressed, err := ioutil.ReadAll(gzReader)
	if err != nil {
		return "", err
	}
	return string(decompressed), nil
}

// HasSnapshot checks if a snapshot exists for this revision
func (d *document) HasSnapshot(spaceId string, documentId string, logDocumentId string) bool {
	snapshotFile := d.GetSnapshotFilePath(spaceId, documentId, logDocumentId)
	fi, err := os.Stat(snapshotFile)
	return err == nil && !fi.IsDir()
}

// GetSnapshotFileSize returns the gzipped snapshot size in bytes
func (d *document) GetSnapshotFileSize(spaceId string, documentId string, logDocumentId string) int64 {
	snapshotFile := d.GetSnapshotFilePath(spaceId, documentId, logDocumentId)
	if fi, err := os.Stat(snapshotFile); err == nil {
		return fi.Size()
	}
	return 0
}

// GenerateStandaloneHTML converts markdown content into an offline standalone single-page HTML file
func (d *document) GenerateStandaloneHTML(title string, spaceName string, author string, updateTime string, markdownContent string) string {
	renderedHtml := string(blackfriday.Run([]byte(markdownContent)))
	safeTitle := html.EscapeString(title)
	safeSpace := html.EscapeString(spaceName)
	safeAuthor := html.EscapeString(author)
	nowStr := time.Now().Format("2006-01-02 15:04:05")

	authorHtml := ""
	if safeAuthor != "" {
		authorHtml = fmt.Sprintf(`<span class="share-meta-item"><svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path><circle cx="12" cy="7" r="4"></circle></svg>作者：%s</span><span class="share-meta-dot">·</span>`, safeAuthor)
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{SAFE_TITLE}} - MM-Wiki 离线文档</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.8/dist/katex.min.css" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.8.0/build/styles/github.min.css" crossorigin="anonymous">
    <script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.8/dist/katex.min.js" crossorigin="anonymous"></script>
    <script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.8/dist/contrib/auto-render.min.js" crossorigin="anonymous" onload="if(window.renderMathInElement){renderMathInElement(document.body,{delimiters:[{left:'$$',right:'$$',display:true},{left:'$',right:'$',display:false}]});}"></script>
    <script defer src="https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.8.0/build/highlight.min.js" crossorigin="anonymous" onload="if(window.hljs){document.querySelectorAll('pre code').forEach(function(el){hljs.highlightElement(el);});}"></script>
    <style>
        * { box-sizing: border-box; }
        html, body {
            margin: 0;
            padding: 0;
            background-color: #f6f8fa;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "WenQuanYi Micro Hei", sans-serif;
            color: #1f2328;
            line-height: 1.75;
            -webkit-font-smoothing: antialiased;
        }

        /* 居中阅读 Flex 包装器：居中卡片 + 大屏右侧吸顶大纲 */
        .share-doc-flex-wrapper {
            display: flex;
            justify-content: center;
            align-items: flex-start;
            gap: 32px;
            width: 100%;
            max-width: 1280px;
            margin: 32px auto 80px auto;
            padding: 0 24px;
            box-sizing: border-box;
            position: relative;
            transition: max-width 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        }

        /* 居中模式（默认黄金阅读视宽 960px） */
        .share-doc-container {
            flex: 1;
            min-width: 0;
            max-width: 960px;
            width: 100%;
            position: relative;
            box-sizing: border-box;
            transition: max-width 0.25s cubic-bezier(0.16, 1, 0.3, 1);
        }

        /* 全宽自适应模式 */
        .share-doc-flex-wrapper.is-full {
            max-width: 100% !important;
            margin: 20px auto 60px auto !important;
            padding: 0 32px !important;
        }
        .share-doc-flex-wrapper.is-full .share-doc-container {
            max-width: 100% !important;
        }

        /* 核心文档白底卡片 */
        .share-doc-card {
            background: #ffffff;
            border: 1px solid #d0d7de;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);
            overflow: hidden;
            width: 100%;
            box-sizing: border-box;
        }

        /* 头部标题与操作区 */
        .share-doc-header {
            padding: 28px 40px 20px 40px;
            border-bottom: 1px solid #eaecef;
            background: #fafbfc;
        }

        .share-doc-header-top {
            display: flex;
            align-items: flex-start;
            justify-content: space-between;
            gap: 16px;
            margin-bottom: 12px;
        }

        .share-doc-title {
            margin: 0;
            font-size: 26px;
            font-weight: 700;
            color: #1f2328;
            line-height: 1.35;
            display: flex;
            align-items: center;
            flex-wrap: wrap;
            gap: 8px;
            flex: 1;
            min-width: 0;
            word-break: break-word;
        }

        .share-space-badge {
            display: inline-block;
            vertical-align: middle;
            font-size: 12px;
            font-weight: 500;
            color: #2563eb;
            background: #eff6ff;
            border: 1px solid #bfdbfe;
            border-radius: 9999px;
            padding: 2px 10px;
            line-height: 1.4;
        }

        .share-doc-actions {
            display: flex;
            align-items: center;
            gap: 8px;
            flex-shrink: 0;
        }

        .doc-action-btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 6px 14px;
            font-size: 13px;
            font-weight: 500;
            border-radius: 6px;
            border: 1px solid #d0d7de;
            background: #ffffff;
            color: #24292e;
            text-decoration: none;
            transition: all 0.15s ease;
            cursor: pointer;
            box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
            line-height: normal;
        }

        .doc-action-btn:hover {
            background: #f3f4f6;
            border-color: #8c959f;
            color: #1f2328;
        }

        .share-doc-meta {
            display: flex;
            align-items: center;
            flex-wrap: wrap;
            gap: 10px;
            font-size: 13px;
            color: #656d76;
        }

        .share-meta-item {
            display: inline-flex;
            align-items: center;
            gap: 4px;
        }

        .share-meta-dot {
            color: #d0d7de;
        }

        /* 正文区域 */
        .share-doc-body {
            padding: 36px 40px 60px 40px;
        }

        .markdown-body {
            font-size: 15px;
            line-height: 1.75;
            word-wrap: break-word;
            color: #1f2328;
        }

        .markdown-body h1, .markdown-body h2, .markdown-body h3,
        .markdown-body h4, .markdown-body h5, .markdown-body h6 {
            margin-top: 28px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.35;
            color: #1f2328;
            scroll-margin-top: 32px;
        }

        .markdown-body h1 { font-size: 24px; padding-bottom: 8px; border-bottom: 1px solid #d8dee4; }
        .markdown-body h2 { font-size: 20px; padding-bottom: 6px; border-bottom: 1px solid #d8dee4; }
        .markdown-body h3 { font-size: 17px; }

        .markdown-body p, .markdown-body blockquote, .markdown-body ul, .markdown-body ol,
        .markdown-body dl, .markdown-body table, .markdown-body pre {
            margin-top: 0;
            margin-bottom: 16px;
        }

        .markdown-body blockquote {
            padding: 10px 18px;
            color: #57606a;
            border-left: 4px solid #d0d7de;
            background-color: #f6f8fa;
            border-radius: 0 6px 6px 0;
        }

        /* 现代 Callout 提示框组件 */
        .callout {
            margin: 20px 0;
            padding: 14px 18px;
            border-radius: 8px;
            border-left: 4px solid #2563eb;
            background-color: #f0f7ff;
            font-size: 14.5px;
            line-height: 1.65;
        }
        .callout-title {
            font-weight: 600;
            margin-bottom: 6px;
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 14px;
        }
        .callout-content p {
            margin: 0;
        }
        .callout-content p + p {
            margin-top: 8px;
        }
        .callout-note, .callout-info { border-left-color: #2563eb; background-color: #f0f7ff; color: #1e40af; }
        .callout-note .callout-title, .callout-info .callout-title { color: #1d4ed8; }
        .callout-tip, .callout-success { border-left-color: #10b981; background-color: #ecfdf5; color: #065f46; }
        .callout-tip .callout-title, .callout-success .callout-title { color: #047857; }
        .callout-warning { border-left-color: #f59e0b; background-color: #fffbeb; color: #92400e; }
        .callout-warning .callout-title { color: #b45309; }
        .callout-danger, .callout-caution, .callout-important { border-left-color: #ef4444; background-color: #fef2f2; color: #991b1b; }
        .callout-danger .callout-title, .callout-caution .callout-title, .callout-important .callout-title { color: #b91c1c; }

        .markdown-body code {
            padding: 2px 6px;
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
            font-size: 85%;
            background-color: #eff1f3;
            border-radius: 4px;
            color: #24292e;
        }

        .markdown-body pre {
            position: relative;
            padding: 16px 20px;
            overflow: auto;
            font-size: 85%;
            line-height: 1.45;
            background-color: #f6f8fa;
            border-radius: 8px;
            border: 1px solid #d0d7de;
        }

        .markdown-body pre code {
            padding: 0;
            background-color: transparent;
            border: 0;
            word-break: normal;
            white-space: pre;
        }

        .code-copy-btn {
            position: absolute;
            top: 8px;
            right: 8px;
            padding: 4px 10px;
            font-size: 12px;
            color: #64748b;
            background: #ffffff;
            border: 1px solid #cbd5e1;
            border-radius: 5px;
            cursor: pointer;
            opacity: 0;
            transition: all 0.15s ease;
            box-shadow: 0 1px 2px rgba(0,0,0,0.05);
        }
        .markdown-body pre:hover .code-copy-btn {
            opacity: 1;
        }
        .code-copy-btn:hover {
            color: #0f172a;
            background: #f1f5f9;
            border-color: #94a3b8;
        }
        .code-copy-btn.copied {
            color: #16a34a;
            border-color: #86efac;
            background: #f0fdf4;
            opacity: 1;
        }

        .markdown-body table {
            border-spacing: 0;
            border-collapse: collapse;
            display: table;
            width: 100%;
            overflow: auto;
            margin: 20px 0;
            border: 1px solid #d0d7de;
            border-radius: 6px;
        }

        .markdown-body table th, .markdown-body table td {
            padding: 10px 16px;
            border: 1px solid #d0d7de;
        }

        .markdown-body table th {
            font-weight: 600;
            background-color: #f6f8fa;
        }

        .markdown-body table tr:nth-child(2n) {
            background-color: #fbfcfd;
        }

        .markdown-body img {
            max-width: 100%;
            height: auto;
            border-radius: 6px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.08);
        }

        .markdown-body hr {
            height: 2px;
            padding: 0;
            margin: 24px 0;
            background-color: #d8dee4;
            border: 0;
        }

        /* 底部版权 */
        .share-footer {
            text-align: center;
            padding: 24px 0 40px 0;
            font-size: 13px;
            color: #8c959f;
        }

        /* 大屏专属右侧固定吸顶大纲导航 (Sticky Outline TOC) */
        .share-doc-sticky-toc-wrapper {
            width: 250px;
            flex-shrink: 0;
            position: sticky;
            top: 32px;
            max-height: calc(100vh - 64px);
            overflow-y: auto;
            padding-left: 18px;
            border-left: 1px solid #eef2f6;
            box-sizing: border-box;
        }

        @media (max-width: 1299px) {
            .share-doc-sticky-toc-wrapper {
                display: none !important;
            }
        }

        @media (min-width: 1300px) {
            .share-doc-sticky-toc-wrapper {
                display: block;
            }
            body.has-sticky-toc #header-toc-btn,
            body.has-sticky-toc #view-toc-floating-btn {
                display: none !important;
            }
        }

        .sticky-toc-title {
            font-size: 12px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            color: #94a3b8;
            margin-bottom: 12px;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .sticky-toc-content ul {
            list-style: none;
            padding-left: 12px;
            margin: 0;
        }

        .sticky-toc-content > ul {
            padding-left: 0;
        }

        .sticky-toc-content li {
            margin: 5px 0;
            font-size: 13px;
            line-height: 1.4;
        }

        .sticky-toc-content li.toc-level-1 { font-weight: 600; }
        .sticky-toc-content li.toc-level-2 { padding-left: 8px; }
        .sticky-toc-content li.toc-level-3 { padding-left: 16px; font-size: 12.5px; }
        .sticky-toc-content li.toc-level-4 { padding-left: 24px; font-size: 12px; }

        .sticky-toc-content a {
            color: #64748b;
            text-decoration: none;
            display: block;
            padding: 3px 6px;
            border-radius: 4px;
            transition: all 0.15s ease;
            word-break: break-word;
        }

        .sticky-toc-content a:hover {
            color: #2563eb;
            background: #f1f5f9;
        }

        .sticky-toc-content a.active {
            color: #2563eb;
            font-weight: 600;
            background: #eff6ff;
        }

        /* 移动端 / 窄屏大纲弹窗与悬浮按钮 */
        .view-floating-btn {
            position: fixed;
            bottom: 24px;
            right: 24px;
            width: 44px;
            height: 44px;
            border-radius: 50%;
            background: #ffffff;
            border: 1px solid #d0d7de;
            box-shadow: 0 4px 14px rgba(0,0,0,0.12);
            cursor: pointer;
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            transition: all 0.2s cubic-bezier(0.16, 1, 0.3, 1);
            color: #475569;
        }
        .view-floating-btn:hover {
            background: #2563eb;
            border-color: #2563eb;
            color: #fff;
            transform: translateY(-2px);
            box-shadow: 0 6px 18px rgba(37, 99, 235, 0.35);
        }

        .view-toc-panel {
            display: none;
            position: fixed;
            top: 70px;
            right: 24px;
            width: 300px;
            max-height: 520px;
            background: #ffffff;
            border-radius: 12px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.15), 0 1px 3px rgba(0, 0, 0, 0.08);
            border: 1px solid #e1e4e8;
            z-index: 1050;
            flex-direction: column;
            overflow: hidden;
            animation: tocFadeIn 0.18s ease;
        }

        @keyframes tocFadeIn {
            from { opacity: 0; transform: translateY(-8px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .view-toc-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 12px 16px;
            background: #f8fafc;
            border-bottom: 1px solid #eef2f6;
            font-size: 13.5px;
            font-weight: 600;
            color: #1e293b;
        }

        .view-toc-close {
            border: none;
            background: transparent;
            font-size: 20px;
            line-height: 1;
            cursor: pointer;
            color: #64748b;
            padding: 0;
        }
        .view-toc-close:hover { color: #0f172a; }

        .view-toc-body {
            padding: 12px 16px 20px 16px;
            overflow-y: auto;
            flex: 1;
        }
        .view-toc-body ul {
            list-style: none;
            padding-left: 12px;
            margin: 0;
        }
        .view-toc-body > ul { padding-left: 0; }
        .view-toc-body li { margin: 5px 0; font-size: 13px; line-height: 1.4; }
        .view-toc-body a {
            color: #64748b;
            text-decoration: none;
            display: block;
            padding: 4px 6px;
            border-radius: 4px;
            transition: all 0.15s ease;
            word-break: break-word;
        }
        .view-toc-body a:hover { color: #2563eb; background: #f1f5f9; }

        @media (max-width: 768px) {
            .share-doc-flex-wrapper {
                padding: 0 12px !important;
                margin: 12px auto 40px auto !important;
            }
            .share-doc-header, .share-doc-body {
                padding: 20px 16px !important;
            }
            .share-doc-header-top {
                flex-direction: column !important;
                gap: 12px !important;
            }
            .share-doc-title {
                font-size: 21px !important;
            }
            .share-doc-actions {
                width: 100% !important;
                flex-wrap: wrap;
            }
            .view-toc-panel {
                top: auto !important;
                bottom: 0 !important;
                left: 0 !important;
                right: 0 !important;
                width: 100% !important;
                max-height: 70vh !important;
                border-radius: 16px 16px 0 0 !important;
                box-shadow: 0 -10px 40px rgba(0, 0, 0, 0.25) !important;
            }
        }

        /* 打印与 PDF 另存净化排版 */
        @media print {
            body, .share-doc-card, .share-doc-container, .share-doc-flex-wrapper {
                background: #ffffff !important;
                padding: 0 !important;
                margin: 0 !important;
                border: none !important;
                box-shadow: none !important;
                max-width: 100% !important;
                width: 100% !important;
            }
            .no-print, .share-doc-sticky-toc-wrapper, .view-floating-btn, .view-toc-panel {
                display: none !important;
            }
            .share-doc-header {
                background: transparent !important;
                padding: 0 0 16px 0 !important;
                border-bottom: 2px solid #000000 !important;
            }
            .share-doc-body {
                padding: 24px 0 !important;
            }
            .markdown-body pre, .markdown-body table, .markdown-body blockquote, .markdown-body img {
                page-break-inside: avoid;
            }
            .markdown-body h1, .markdown-body h2, .markdown-body h3, .markdown-body h4 {
                page-break-after: avoid;
            }
        }
    </style>
</head>
<body>
    <div class="share-doc-flex-wrapper" id="share-flex-wrapper">
        <div class="share-doc-container">
            <div class="share-doc-card">
                <!-- 头部标题与元信息 -->
                <header class="share-doc-header">
                    <div class="share-doc-header-top">
                        <h1 class="share-doc-title">
                            <svg viewBox="0 0 24 24" width="24" height="24" fill="none" stroke="#2563eb" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;">
                                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                                <polyline points="14 2 14 8 20 8"></polyline>
                                <line x1="16" y1="13" x2="8" y2="13"></line>
                                <line x1="16" y1="17" x2="8" y2="17"></line>
                                <polyline points="10 9 9 9 8 9"></polyline>
                            </svg>
                            <span>{{SAFE_TITLE}}</span>
                            <span class="share-space-badge">{{SAFE_SPACE}}</span>
                        </h1>
                        <div class="share-doc-actions no-print">
                            <button type="button" class="doc-action-btn" id="header-toc-btn" style="display: none;" title="查看文章目录大纲">
                                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
                                <span>目录</span>
                            </button>
                            <button type="button" class="doc-action-btn" id="export-width-toggle-btn" title="切换全宽/居中阅读模式">
                                <svg id="width-icon-expand" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 3 21 3 21 9"></polyline><polyline points="9 21 3 21 3 15"></polyline><line x1="21" y1="3" x2="14" y2="10"></line><line x1="3" y1="21" x2="10" y2="14"></line></svg>
                                <svg id="width-icon-compress" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none;"><polyline points="4 14 10 14 10 20"></polyline><polyline points="20 10 14 10 14 4"></polyline><line x1="14" y1="10" x2="21" y2="3"></line><line x1="3" y1="21" x2="10" y2="14"></line></svg>
                                <span id="width-toggle-text">全宽</span>
                            </button>
                            <button type="button" class="doc-action-btn" onclick="window.print()" title="打印文档或另存为 PDF">
                                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 6 2 18 2 18 9"></polyline><path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"></path><rect x="6" y="14" width="12" height="8"></rect></svg>
                                <span>打印 / 另存为 PDF</span>
                            </button>
                        </div>
                    </div>
                    <div class="share-doc-meta">
                        <span class="share-meta-item">
                            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                            空间：{{SAFE_SPACE}}
                        </span>
                        <span class="share-meta-dot">·</span>
                        {{AUTHOR_HTML}}
                        <span class="share-meta-item">
                            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>
                            导出时间：{{NOW_STR}}
                        </span>
                    </div>
                </header>

                <!-- 正文区域 -->
                <div class="share-doc-body">
                    <article class="markdown-body" id="wiki-article">
                        {{RENDERED_HTML}}
                    </article>
                </div>
            </div>

            <!-- 底部版权 -->
            <footer class="share-footer no-print">
                本文档由 MM-Wiki 知识库系统自动生成导出
            </footer>
        </div>

        <!-- 大屏右侧固定吸顶大纲导航 (Sticky Outline) -->
        <div class="share-doc-sticky-toc-wrapper no-print" id="sticky-toc-wrapper" style="display: none;">
            <div class="sticky-toc-title">
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="#2563eb" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
                <span>本文大纲</span>
            </div>
            <div class="sticky-toc-content" id="sticky-doc-toc"></div>
        </div>
    </div>

    <!-- 移动端/窄屏悬浮大纲胶囊按钮 -->
    <button type="button" class="view-floating-btn no-print" id="view-toc-floating-btn" title="查看文章目录大纲" style="display: none;">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"></line><line x1="8" y1="12" x2="21" y2="12"></line><line x1="8" y1="18" x2="21" y2="18"></line><line x1="3" y1="6" x2="3.01" y2="6"></line><line x1="3" y1="12" x2="3.01" y2="12"></line><line x1="3" y1="18" x2="3.01" y2="18"></line></svg>
    </button>

    <!-- 移动端/窄屏弹窗大纲抽屉 -->
    <div class="view-toc-panel no-print" id="view-toc-panel">
        <div class="view-toc-header">
            <span>本文大纲</span>
            <button type="button" class="view-toc-close" id="view-toc-close">&times;</button>
        </div>
        <div class="view-toc-body" id="page-view-dir-toc"></div>
    </div>

    <script>
    (function() {
        // 1. 全宽/居中阅读模式切换
        var savedMode = localStorage.getItem("mmwiki_export_width_mode") || "narrow";
        var flexWrapper = document.getElementById("share-flex-wrapper");
        var expandIcon = document.getElementById("width-icon-expand");
        var compressIcon = document.getElementById("width-icon-compress");
        var widthText = document.getElementById("width-toggle-text");
        var widthBtn = document.getElementById("export-width-toggle-btn");

        function applyWidthMode(mode) {
            if (!flexWrapper) return;
            if (mode === "full") {
                flexWrapper.classList.add("is-full");
                if (expandIcon) expandIcon.style.display = "none";
                if (compressIcon) compressIcon.style.display = "inline-block";
                if (widthText) widthText.textContent = "居中";
                if (widthBtn) widthBtn.setAttribute("title", "当前为全宽模式，点击切换为居中阅读模式");
            } else {
                flexWrapper.classList.remove("is-full");
                if (expandIcon) expandIcon.style.display = "inline-block";
                if (compressIcon) compressIcon.style.display = "none";
                if (widthText) widthText.textContent = "全宽";
                if (widthBtn) widthBtn.setAttribute("title", "当前为居中模式，点击切换为全宽自适应");
            }
        }
        applyWidthMode(savedMode);

        if (widthBtn) {
            widthBtn.addEventListener("click", function() {
                var current = flexWrapper.classList.contains("is-full") ? "full" : "narrow";
                var next = current === "full" ? "narrow" : "full";
                localStorage.setItem("mmwiki_export_width_mode", next);
                applyWidthMode(next);
            });
        }

        // 2. Callout 提示框智能转换
        document.querySelectorAll('.markdown-body blockquote').forEach(function(bq) {
            var p = bq.querySelector('p');
            if (!p) return;
            var html = p.innerHTML;
            var m = html.match(/^\s*\[!([A-Z]+)\](?:\s+([^<\n]*))?(?:<br\s*\/?>|\n)?([\s\S]*)$/i);
            if (m) {
                var type = m[1].toLowerCase();
                var title = m[2] || type.toUpperCase();
                var rest = m[3] || '';
                var icons = {
                    note: '📝', info: 'ℹ️', tip: '💡', warning: '⚠️', danger: '🚨', important: '❗', caution: '🛑', success: '✅'
                };
                var icon = icons[type] || '📌';
                var callout = document.createElement('div');
                callout.className = 'callout callout-' + type;
                callout.innerHTML = '<div class="callout-title">' + icon + ' ' + title + '</div><div class="callout-content"><p>' + rest + '</p></div>';
                bq.parentNode.replaceChild(callout, bq);
            }
        });

        // 3. 代码块一键复制按钮
        document.querySelectorAll('.markdown-body pre').forEach(function(pre) {
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'code-copy-btn no-print';
            btn.textContent = '复制';
            btn.addEventListener('click', function() {
                var code = pre.querySelector('code');
                var text = code ? code.innerText : pre.innerText;
                navigator.clipboard.writeText(text).then(function() {
                    btn.textContent = '已复制!';
                    btn.classList.add('copied');
                    setTimeout(function() {
                        btn.textContent = '复制';
                        btn.classList.remove('copied');
                    }, 1800);
                }).catch(function() {
                    var ta = document.createElement('textarea');
                    ta.value = text;
                    document.body.appendChild(ta);
                    ta.select();
                    document.execCommand('copy');
                    document.body.removeChild(ta);
                    btn.textContent = '已复制!';
                    btn.classList.add('copied');
                    setTimeout(function() {
                        btn.textContent = '复制';
                        btn.classList.remove('copied');
                    }, 1800);
                });
            });
            pre.appendChild(btn);
        });

        // 4. 本文大纲生成与 ScrollSpy 导航
        var article = document.getElementById('wiki-article');
        if (!article) return;
        var headings = article.querySelectorAll('h1, h2, h3, h4, h5, h6');
        if (!headings || headings.length === 0) return;

        var stickyTocContent = document.getElementById('sticky-doc-toc');
        var drawerNav = document.getElementById('page-view-dir-toc');
        var ul = document.createElement('ul');

        headings.forEach(function(h, idx) {
            if (!h.id) {
                h.id = 'heading-' + idx;
            }
            var level = parseInt(h.tagName.substring(1), 10);
            var li = document.createElement('li');
            li.className = 'toc-level-' + level;
            var a = document.createElement('a');
            a.href = '#' + h.id;
            a.textContent = h.textContent.trim();
            li.appendChild(a);
            ul.appendChild(li);
        });

        if (stickyTocContent) stickyTocContent.appendChild(ul.cloneNode(true));
        if (drawerNav) drawerNav.appendChild(ul);

        var stickyTocWrapper = document.getElementById('sticky-toc-wrapper');
        var headerTocBtn = document.getElementById('header-toc-btn');
        var floatingBtn = document.getElementById('view-toc-floating-btn');
        var drawer = document.getElementById('view-toc-panel');
        var drawerClose = document.getElementById('view-toc-close');

        if (stickyTocWrapper) stickyTocWrapper.style.display = 'block';
        if (headerTocBtn) headerTocBtn.style.display = 'inline-flex';
        document.body.classList.add('has-sticky-toc');

        function toggleDrawer(e) {
            if (e) e.stopPropagation();
            if (!drawer) return;
            if (drawer.style.display === 'flex') {
                drawer.style.display = 'none';
            } else {
                drawer.style.display = 'flex';
            }
        }

        if (headerTocBtn) headerTocBtn.addEventListener('click', toggleDrawer);
        if (floatingBtn) floatingBtn.addEventListener('click', toggleDrawer);
        if (drawerClose) drawerClose.addEventListener('click', function(e) { e.stopPropagation(); drawer.style.display = 'none'; });

        // 点击外部关闭抽屉
        document.addEventListener('click', function(e) {
            if (drawer && drawer.style.display === 'flex') {
                if (!drawer.contains(e.target) && e.target !== headerTocBtn && e.target !== floatingBtn) {
                    drawer.style.display = 'none';
                }
            }
        });

        // 点击大纲链接平滑滚动并关闭抽屉
        document.querySelectorAll('.sticky-toc-content a, .view-toc-body a').forEach(function(link) {
            link.addEventListener('click', function(e) {
                var targetId = link.getAttribute('href');
                if (targetId && targetId.startsWith('#')) {
                    var target = document.getElementById(targetId.substring(1));
                    if (target) {
                        e.preventDefault();
                        target.scrollIntoView({ behavior: 'smooth', block: 'start' });
                        if (drawer) drawer.style.display = 'none';
                    }
                }
            });
        });

        // ScrollSpy 滚动高亮与浮动按钮显示
        window.addEventListener('scroll', function() {
            var st = window.pageYOffset || document.documentElement.scrollTop;
            if (st > 140) {
                if (floatingBtn && window.innerWidth < 1300) floatingBtn.style.display = 'flex';
            } else {
                if (floatingBtn) floatingBtn.style.display = 'none';
            }

            var currentId = '';
            headings.forEach(function(h) {
                var top = h.getBoundingClientRect().top + st;
                if (st >= top - 60) {
                    currentId = h.id;
                }
            });

            if (currentId) {
                document.querySelectorAll('.sticky-toc-content a, .view-toc-body a').forEach(function(a) {
                    if (a.getAttribute('href') === '#' + currentId) {
                        a.classList.add('active');
                    } else {
                        a.classList.remove('active');
                    }
                });
            }
        });
    })();
    </script>
</body>
</html>`

	res := strings.Replace(htmlTemplate, "{{SAFE_TITLE}}", safeTitle, -1)
	res = strings.Replace(res, "{{SAFE_SPACE}}", safeSpace, -1)
	res = strings.Replace(res, "{{AUTHOR_HTML}}", authorHtml, 1)
	res = strings.Replace(res, "{{NOW_STR}}", nowStr, 1)
	res = strings.Replace(res, "{{RENDERED_HTML}}", renderedHtml, 1)

	return res
}


