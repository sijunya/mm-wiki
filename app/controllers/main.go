package controllers

import (
	"github.com/chaiyd/mm-wiki/app/models"
	"github.com/chaiyd/mm-wiki/app/services"
	"strings"
)

type MainController struct {
	BaseController
}

func (this *MainController) Index() {

	collectDocs, err := models.CollectionModel.GetCollectionsByUserIdAndType(this.UserId, models.Collection_Type_Doc)
	if err != nil {
		this.ErrorLog("查找收藏文档错误: " + err.Error())
		this.ViewError("查找收藏文档错误！")
	}
	docIds := []string{}
	for _, collectDoc := range collectDocs {
		docIds = append(docIds, collectDoc["resource_id"])
	}

	documents, err := models.DocumentModel.GetDocumentsByDocumentIds(docIds)
	if err != nil {
		this.ErrorLog("查找收藏文档错误: " + err.Error())
		this.ViewError("查找收藏文档错误！")
	}

	this.Data["documents"] = documents
	this.Data["count"] = len(documents)
	this.viewLayout("main/index", "main")
}

func (this *MainController) Default() {

	page, _ := this.GetInt("page", 1)
	number, _ := this.GetInt("number", 10)
	maxPage := 10
	if page >= maxPage {
		page = maxPage
	}
	//number := 8
	limit := (page - 1) * number

	userId := this.UserId

	logDocuments, err := models.LogDocumentModel.GetLogDocumentsByLimit(userId, limit, number)
	if err != nil {
		this.ErrorLog("查找更新文档列表失败：" + err.Error())
		this.ViewError("查找更新文档列表失败！")
	}

	count, err := models.LogDocumentModel.CountLogDocuments()
	if err != nil {
		this.ErrorLog("查找更新文档总数失败：" + err.Error())
		this.ViewError("查找更新文档列表失败！")
	}
	if count >= int64(maxPage*number) {
		count = int64(maxPage * number)
	}

	userIds := []string{}
	docIds := []string{}
	for _, logDocument := range logDocuments {
		userIds = append(userIds, logDocument["user_id"])
		docIds = append(docIds, logDocument["document_id"])
	}
	users, err := models.UserModel.GetUsersByUserIds(userIds)
	if err != nil {
		this.ErrorLog("查找更新文档用户失败：" + err.Error())
		this.ViewError("查找更新文档列表失败！")
	}
	docs, err := models.DocumentModel.GetAllDocumentsByDocumentIds(docIds)
	if err != nil {
		this.ErrorLog("查找文档信息失败：" + err.Error())
		this.ViewError("查找更新文档列表失败！")
	}
	for _, logDocument := range logDocuments {
		logDocument["username"] = ""
		for _, user := range users {
			if logDocument["user_id"] == user["user_id"] {
				logDocument["username"] = user["username"]
				logDocument["given_name"] = user["given_name"]
				break
			}
		}
		for _, doc := range docs {
			if logDocument["document_id"] == doc["document_id"] {
				logDocument["document_name"] = doc["name"]
				logDocument["document_type"] = doc["type"]
				break
			}
		}
	}

	// link
	links, err := models.LinkModel.GetLinksOrderBySequence()
	if err != nil {
		this.ErrorLog("查找快捷链接失败：" + err.Error())
		this.ViewError("查找快捷链接失败！")
	}
	// contacts
	contacts, err := models.ContactModel.GetAllContact()
	if err != nil {
		this.ErrorLog("查找快捷链接失败：" + err.Error())
		this.ViewError("查找快捷链接失败！")
	}

	// main title config
	mainTitle := models.ConfigModel.GetConfigValueByKey(models.ConfigKeyMainTitle, "")
	mainDescription := models.ConfigModel.GetConfigValueByKey(models.ConfigKeyMainDescription, "")

	this.Data["panel_title"] = mainTitle
	this.Data["panel_description"] = mainDescription
	this.Data["logDocuments"] = logDocuments
	this.Data["links"] = links
	this.Data["contacts"] = contacts
	this.SetPaginator(number, count)
	this.viewLayout("main/default", "default")
}

func (this *MainController) About() {
	this.viewLayout("main/about", "default")
}

// 搜索，支持综合全文、标题、正文检索
func (this *MainController) Search() {

	keyword := strings.TrimSpace(this.GetString("keyword", ""))
	searchType := strings.TrimSpace(this.GetString("search_type", "all"))
	if searchType != "title" && searchType != "content" {
		searchType = "all"
	}

	this.Data["search_type"] = searchType
	this.Data["keyword"] = keyword
	this.Data["count"] = 0
	if keyword == "" {
		this.viewLayout("main/search", "default")
		return
	}

	// 获取该用户有权限的空间
	publicSpaces, err := models.SpaceModel.GetSpacesByVisitLevel(models.Space_VisitLevel_Public)
	if err != nil {
		this.ErrorLog("搜索文档列表获取用户空间权限出错：" + err.Error())
		this.ViewError("搜索文档错误！")
	}
	spaceUsers, err := models.SpaceUserModel.GetSpaceUsersByUserId(this.UserId)
	if err != nil {
		this.ErrorLog("搜索文档列表获取用户空间权限出错：" + err.Error())
		this.ViewError("搜索文档错误！")
	}
	permittedSpaceIds := []string{}
	spaceIdsMap := make(map[string]bool)
	for _, publicSpace := range publicSpaces {
		spId := publicSpace["space_id"]
		if !spaceIdsMap[spId] {
			spaceIdsMap[spId] = true
			permittedSpaceIds = append(permittedSpaceIds, spId)
		}
	}
	for _, spaceUser := range spaceUsers {
		spId := spaceUser["space_id"]
		if !spaceIdsMap[spId] {
			spaceIdsMap[spId] = true
			permittedSpaceIds = append(permittedSpaceIds, spId)
		}
	}

	searchResults, err := services.SearchService.Search(keyword, searchType, permittedSpaceIds)
	if err != nil {
		this.ErrorLog("搜索文档出错：" + err.Error())
		this.ViewError("搜索文档错误！")
	}

	// 转换为模板兼容格式
	documents := make([]map[string]string, 0, len(searchResults))
	for _, res := range searchResults {
		doc := map[string]string{
			"document_id":    res.DocumentId,
			"space_id":       res.SpaceId,
			"space_name":     res.SpaceName,
			"name":           res.Name,
			"type":           res.Type,
			"update_time":    res.UpdateTime,
			"match_type":     res.MatchType,
			"match_desc":     res.MatchDesc,
			"search_content": res.Snippet,
		}
		documents = append(documents, doc)
	}

	this.Data["search_type"] = searchType
	this.Data["keyword"] = keyword
	this.Data["documents"] = documents
	this.Data["count"] = len(documents)
	this.viewLayout("main/search", "default")
}
