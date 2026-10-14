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
		authorHtml = fmt.Sprintf(`<span class="wiki-meta-item">👤 作者：%s</span>`, safeAuthor)
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{SAFE_TITLE}} - MM-Wiki 离线文档</title>
    <style>
        * { box-sizing: border-box; }
        html, body {
            margin: 0;
            padding: 0;
            background-color: #f6f8fa;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "WenQuanYi Micro Hei", Roboto, sans-serif;
            color: #1f2328;
            line-height: 1.7;
        }
        .wiki-standalone-topbar {
            background-color: #ffffff;
            border-bottom: 1px solid #d0d7de;
            padding: 12px 28px;
            display: flex;
            align-items: center;
            justify-content: space-between;
            position: sticky;
            top: 0;
            z-index: 100;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
            width: 100%;
        }
        .wiki-brand {
            font-size: 14px;
            font-weight: 600;
            color: #57606a;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .wiki-brand-tag {
            background: #e1f0ff;
            color: #0969da;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: 500;
        }
        .wiki-actions {
            display: flex;
            gap: 10px;
        }
        .wiki-btn {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            padding: 6px 14px;
            font-size: 13px;
            font-weight: 500;
            background-color: #ffffff;
            border: 1px solid #d0d7de;
            border-radius: 6px;
            cursor: pointer;
            color: #24292e;
            text-decoration: none;
            transition: all 0.15s ease;
        }
        .wiki-btn:hover {
            background-color: #f3f4f6;
            border-color: #8c959f;
        }
        .wiki-container {
            max-width: 960px;
            margin: 32px auto 80px auto;
            background: #ffffff;
            border: 1px solid #d0d7de;
            border-radius: 10px;
            padding: 48px 56px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.04);
            box-sizing: border-box;
        }
        .wiki-header {
            border-bottom: 1px solid #d8dee4;
            padding-bottom: 24px;
            margin-bottom: 32px;
        }
        .wiki-title {
            font-size: 30px;
            font-weight: 700;
            color: #1f2328;
            margin: 0 0 16px 0;
            line-height: 1.3;
        }
        .wiki-meta {
            font-size: 13px;
            color: #57606a;
            display: flex;
            flex-wrap: wrap;
            gap: 16px;
        }
        .wiki-meta-item {
            display: inline-flex;
            align-items: center;
            gap: 4px;
        }
        .markdown-body {
            font-size: 15px;
            line-height: 1.75;
            word-wrap: break-word;
        }
        .markdown-body h1, .markdown-body h2, .markdown-body h3,
        .markdown-body h4, .markdown-body h5, .markdown-body h6 {
            margin-top: 24px;
            margin-bottom: 16px;
            font-weight: 600;
            line-height: 1.35;
            color: #1f2328;
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
            padding: 8px 16px;
            color: #57606a;
            border-left: 4px solid #d0d7de;
            background-color: #f6f8fa;
            border-radius: 0 4px 4px 0;
        }
        .markdown-body code {
            padding: 2px 6px;
            font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace;
            font-size: 85%;
            background-color: #eff1f3;
            border-radius: 4px;
            color: #24292e;
        }
        .markdown-body pre {
            padding: 16px;
            overflow: auto;
            font-size: 85%;
            line-height: 1.45;
            background-color: #f6f8fa;
            border-radius: 6px;
            border: 1px solid #d0d7de;
        }
        .markdown-body pre code {
            padding: 0;
            background-color: transparent;
            border: 0;
            word-break: normal;
            white-space: pre;
        }
        .markdown-body table {
            border-spacing: 0;
            border-collapse: collapse;
            display: table;
            width: 100%;
            overflow: auto;
            margin-bottom: 16px;
        }
        .markdown-body table th, .markdown-body table td {
            padding: 8px 14px;
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
            border-radius: 4px;
            box-shadow: 0 1px 4px rgba(0,0,0,0.08);
        }
        .markdown-body hr {
            height: 2px;
            padding: 0;
            margin: 24px 0;
            background-color: #d8dee4;
            border: 0;
        }
        .wiki-footer {
            margin-top: 48px;
            padding-top: 16px;
            border-top: 1px solid #d8dee4;
            font-size: 12px;
            color: #8c959f;
            text-align: center;
        }
        @media (max-width: 768px) {
            .wiki-container {
                margin: 0;
                padding: 24px 18px;
                border-radius: 0;
                border: none;
            }
        }
        @media print {
            body, .wiki-container {
                background: #ffffff !important;
                padding: 0 !important;
                margin: 0 !important;
                border: none !important;
                box-shadow: none !important;
                max-width: 100% !important;
            }
            .no-print, .wiki-standalone-topbar {
                display: none !important;
            }
            .wiki-header {
                border-bottom: 2px solid #000 !important;
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
    <div class="wiki-standalone-topbar no-print">
        <div class="wiki-brand">
            <span>MM-Wiki 离线文档</span>
            <span class="wiki-brand-tag">{{SAFE_SPACE}}</span>
        </div>
        <div class="wiki-actions">
            <button onclick="window.print()" class="wiki-btn">
                <span>🖨️ 打印 / 另存为 PDF</span>
            </button>
        </div>
    </div>
    <div class="wiki-container">
        <header class="wiki-header">
            <h1 class="wiki-title">{{SAFE_TITLE}}</h1>
            <div class="wiki-meta">
                <span class="wiki-meta-item">📁 空间：{{SAFE_SPACE}}</span>
                {{AUTHOR_HTML}}
                <span class="wiki-meta-item">🕒 导出时间：{{NOW_STR}}</span>
            </div>
        </header>
        <article class="markdown-body">
            {{RENDERED_HTML}}
        </article>
        <footer class="wiki-footer no-print">
            本文档由 MM-Wiki 知识库系统自动生成导出
        </footer>
    </div>
</body>
</html>`

	res := strings.Replace(htmlTemplate, "{{SAFE_TITLE}}", safeTitle, -1)
	res = strings.Replace(res, "{{SAFE_SPACE}}", safeSpace, -1)
	res = strings.Replace(res, "{{AUTHOR_HTML}}", authorHtml, 1)
	res = strings.Replace(res, "{{NOW_STR}}", nowStr, 1)
	res = strings.Replace(res, "{{RENDERED_HTML}}", renderedHtml, 1)

	return res
}


