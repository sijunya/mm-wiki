package services

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/astaxie/beego/logs"
	"github.com/chaiyd/mm-wiki/app/models"
)

var SearchService = NewSearchService()

type SearchResult struct {
	DocumentId string `json:"document_id"`
	SpaceId    string `json:"space_id"`
	SpaceName  string `json:"space_name"`
	Name       string `json:"name"`
	Type       string `json:"type"`        // "1": 页面, "2": 目录
	UpdateTime string `json:"update_time"`
	MatchType  string `json:"match_type"`  // "both", "title", "content"
	MatchDesc  string `json:"match_desc"`  // 描述标签
	Snippet    string `json:"snippet"`     // 命中的正文高亮摘要
	Score      int    `json:"score"`       // 排序权重
}

type searchService struct {
	reImg    *regexp.Regexp
	reLink   *regexp.Regexp
	reFence  *regexp.Regexp
	reHeader *regexp.Regexp
	reSpace  *regexp.Regexp
}

func NewSearchService() *searchService {
	return &searchService{
		reImg:    regexp.MustCompile(`!\[([^\]]*)\]\([^\)]+\)`),
		reLink:   regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`),
		reFence:  regexp.MustCompile("`{1,3}[a-zA-Z0-9_-]*"),
		reHeader: regexp.MustCompile(`(?m)^#+\s*`),
		reSpace:  regexp.MustCompile(`\s+`),
	}
}

// CleanMarkdown 清理 Markdown 排版符号，保留文本和代码核心，用于生成干净的摘要
func (s *searchService) CleanMarkdown(text string) string {
	text = s.reImg.ReplaceAllString(text, "[图片]")
	text = s.reLink.ReplaceAllString(text, "$1")
	text = s.reFence.ReplaceAllString(text, " ")
	text = s.reHeader.ReplaceAllString(text, "")
	text = s.reSpace.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// ExtractSnippet 提取包含关键词的上下文摘要
func (s *searchService) ExtractSnippet(content string, keyword string, preLen, postLen int) string {
	if keyword == "" || content == "" {
		return ""
	}
	clean := s.CleanMarkdown(content)
	cleanLower := strings.ToLower(clean)
	kwLower := strings.ToLower(keyword)

	idx := strings.Index(cleanLower, kwLower)
	if idx == -1 {
		return ""
	}

	prefixRunes := []rune(clean[:idx])
	matchRunes := []rune(clean[idx : idx+len(keyword)])
	suffixRunes := []rune(clean[idx+len(keyword):])

	start := 0
	hasPrefixEllipsis := false
	if len(prefixRunes) > preLen {
		start = len(prefixRunes) - preLen
		hasPrefixEllipsis = true
	}
	snippetPrefix := string(prefixRunes[start:])

	end := len(suffixRunes)
	hasSuffixEllipsis := false
	if len(suffixRunes) > postLen {
		end = postLen
		hasSuffixEllipsis = true
	}
	snippetSuffix := string(suffixRunes[:end])

	result := ""
	if hasPrefixEllipsis {
		result += "..."
	}
	result += snippetPrefix + string(matchRunes) + snippetSuffix
	if hasSuffixEllipsis {
		result += "..."
	}

	return result
}

// Search 执行综合全文与标题检索
// keyword: 搜索关键词
// searchType: "all" (全部), "title" (仅标题), "content" (仅正文)
// permittedSpaceIds: 用户有权限访问的空间ID列表
func (s *searchService) Search(keyword string, searchType string, permittedSpaceIds []string) ([]*SearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" || len(permittedSpaceIds) == 0 {
		return []*SearchResult{}, nil
	}

	if searchType != "title" && searchType != "content" {
		searchType = "all"
	}

	// 1. 构建权限空间映射表与名称表
	spaceMap := make(map[string]string)
	spaces, err := models.SpaceModel.GetSpaces()
	if err == nil {
		for _, sp := range spaces {
			spaceMap[sp["space_id"]] = sp["name"]
		}
	}
	permittedMap := make(map[string]bool)
	for _, id := range permittedSpaceIds {
		permittedMap[id] = true
	}

	// 2. 获取系统中所有未删除的文档
	allDocs, err := models.DocumentModel.GetAllDocuments()
	if err != nil {
		logs.Error("[SearchService] GetAllDocuments err: %s", err.Error())
		return nil, err
	}

	// 过滤有权限的文档
	targetDocs := make([]map[string]string, 0, len(allDocs))
	for _, doc := range allDocs {
		spaceId := doc["space_id"]
		if permittedMap[spaceId] {
			targetDocs = append(targetDocs, doc)
		}
	}

	keywordLower := strings.ToLower(keyword)
	results := make([]*SearchResult, 0)
	var lock sync.Mutex

	// 3. 并发扫描文档标题与正文
	concurrency := 10
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, doc := range targetDocs {
		wg.Add(1)
		sem <- struct{}{}

		go func(d map[string]string) {
			defer func() {
				<-sem
				wg.Done()
			}()

			docName := d["name"]
			docNameLower := strings.ToLower(docName)
			titleMatched := strings.Contains(docNameLower, keywordLower)

			contentMatched := false
			snippet := ""
			matchCount := 0

			// 如果不需要搜索正文，且标题未命中，则直接跳过
			if searchType == "title" && !titleMatched {
				return
			}

			// 需要检查正文的情形
			if searchType != "title" {
				content, _, err := models.DocumentModel.GetDocumentContentByDocument(d)
				if err == nil && content != "" {
					contentLower := strings.ToLower(content)
					if strings.Contains(contentLower, keywordLower) {
						contentMatched = true
						matchCount = strings.Count(contentLower, keywordLower)
						snippet = s.ExtractSnippet(content, keyword, 40, 60)
					}
				}
			}

			// 根据检索模式过滤
			isHit := false
			matchType := ""
			matchDesc := ""
			score := 0

			if titleMatched && contentMatched {
				isHit = true
				matchType = "both"
				matchDesc = "标题与正文匹配"
				score = 1000 + matchCount*50
				if docNameLower == keywordLower {
					score += 500
				}
			} else if titleMatched {
				if searchType == "all" || searchType == "title" {
					isHit = true
					matchType = "title"
					matchDesc = "标题匹配"
					score = 500
					if docNameLower == keywordLower {
						score += 500
					}
				}
			} else if contentMatched {
				if searchType == "all" || searchType == "content" {
					isHit = true
					matchType = "content"
					matchDesc = "正文匹配"
					score = 100 + matchCount*20
				}
			}

			if !isHit {
				return
			}

			spaceId := d["space_id"]
			res := &SearchResult{
				DocumentId: d["document_id"],
				SpaceId:    spaceId,
				SpaceName:  spaceMap[spaceId],
				Name:       docName,
				Type:       d["type"],
				UpdateTime: d["update_time"],
				MatchType:  matchType,
				MatchDesc:  matchDesc,
				Snippet:    snippet,
				Score:      score,
			}

			lock.Lock()
			results = append(results, res)
			lock.Unlock()
		}(doc)
	}

	wg.Wait()

	// 4. 加权相关性排序：分数高优先；分数相同按更新时间最新排序
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		timeI, _ := strconv.ParseInt(results[i].UpdateTime, 10, 64)
		timeJ, _ := strconv.ParseInt(results[j].UpdateTime, 10, 64)
		return timeI > timeJ
	})

	return results, nil
}
