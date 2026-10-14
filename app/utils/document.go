package utils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
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

// SaveSnapshot compresses content with gzip and saves it under data/markdowns/{spaceId}/history/{documentId}/{logDocumentId}.md.gz
// It automatically maintains a maximum retention of maxKeep (default 30) revisions per document.
func (d *document) SaveSnapshot(spaceId string, documentId string, logDocumentId string, content string) error {
	if content == "" || spaceId == "" || documentId == "" || logDocumentId == "" {
		return nil
	}
	d.lock.Lock()
	defer d.lock.Unlock()

	historyDir := filepath.Join(d.MarkdownAbsDir, spaceId, "history", documentId)
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

	snapshotFile := filepath.Join(d.MarkdownAbsDir, spaceId, "history", documentId, fmt.Sprintf("%s.md.gz", logDocumentId))
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
	snapshotFile := filepath.Join(d.MarkdownAbsDir, spaceId, "history", documentId, fmt.Sprintf("%s.md.gz", logDocumentId))
	fi, err := os.Stat(snapshotFile)
	return err == nil && !fi.IsDir()
}

// GetSnapshotFileSize returns the gzipped snapshot size in bytes
func (d *document) GetSnapshotFileSize(spaceId string, documentId string, logDocumentId string) int64 {
	snapshotFile := filepath.Join(d.MarkdownAbsDir, spaceId, "history", documentId, fmt.Sprintf("%s.md.gz", logDocumentId))
	if fi, err := os.Stat(snapshotFile); err == nil {
		return fi.Size()
	}
	return 0
}

