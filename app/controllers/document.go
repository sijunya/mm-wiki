package controllers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chaiyd/mm-wiki/app/models"
	"github.com/chaiyd/mm-wiki/app/services"
	"github.com/chaiyd/mm-wiki/app/utils"
)

type DocumentController struct {
	BaseController
}

// document index
func (this *DocumentController) Index() {

	documentId := this.GetString("document_id", "")
	if documentId == "" {
		this.ViewError("页面参数错误！", "/space/index")
	}

	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查找空间文档 " + documentId + " 失败：" + err.Error())
		this.ViewError("查找文档失败！")
	}
	if len(document) == 0 {
		this.ViewError("文档不存在！")
	}
	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 所在空间失败：" + err.Error())
		this.ViewError("查找文档失败！")
	}
	if len(space) == 0 {
		this.ViewError("文档所在空间不存在！")
	}
	// check space visit_level
	isVisit, _, _ := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.ViewError("您没有权限访问该空间下的文档！")
	}

	// get default space document
	spaceDocument, err := models.DocumentModel.GetSpaceDefaultDocument(spaceId)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 失败：" + err.Error())
		this.ViewError("查找文档失败！")
	}
	if len(spaceDocument) == 0 {
		this.ViewError(" 空间首页文档不存在！")
	}

	// get space all document
	documents, err := models.DocumentModel.GetAllSpaceDocuments(spaceId)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 所在空间失败：" + err.Error())
		this.ViewError("查找文档失败！")
	}

	// get space privilege
	_, isEditor, isDelete := this.GetDocumentPrivilege(space)

	this.Data["is_editor"] = isEditor
	this.Data["is_delete"] = isDelete
	this.Data["documents"] = documents
	this.Data["default_document_id"] = documentId
	this.Data["space"] = space
	this.Data["space_document"] = spaceDocument
	this.viewLayout("document/index", "document")
}

// add document
func (this *DocumentController) Add() {

	spaceId := this.GetString("space_id", "0")
	parentId := this.GetString("parent_id", "0")

	if spaceId == "0" {
		this.ViewError("没有选择空间！")
	}
	if parentId == "0" {
		this.ViewError("没有选择上级！")
	}
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("添加文档失败：" + err.Error())
		this.ViewError("添加文档失败！")
	}
	if len(space) == 0 {
		this.ViewError("空间不存在！")
	}

	// check space document privilege
	_, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isEditor {
		this.ViewError("您没有权限在该空间下创建文档！")
	}

	parentDocument, err := models.DocumentModel.GetDocumentByDocumentId(parentId)
	if err != nil {
		this.ErrorLog("添加文档 " + parentId + " 失败：" + err.Error())
		this.ViewError("添加文档失败！")
	}
	if len(parentDocument) == 0 {
		this.ViewError("父文档不存在！")
	}
	path := parentDocument["path"] + "," + parentId
	// get parent documents by path
	parentDocuments, err := models.DocumentModel.GetParentDocumentsByPath(path)
	if err != nil {
		this.ErrorLog("查找父文档失败：" + err.Error())
		this.ViewError("查找父文档失败！")
	}
	if len(parentDocuments) == 0 {
		this.ViewError("父文档不存在！")
	}

	this.Data["parent_documents"] = parentDocuments
	this.Data["parent_id"] = parentId
	this.Data["space_id"] = spaceId
	this.viewLayout("document/form", "default")

}

// save document
func (this *DocumentController) Save() {

	if !this.IsPost() {
		this.ViewError("请求方式有误！", "/main/index")
	}
	spaceId := strings.TrimSpace(this.GetString("space_id", "0"))
	parentId := strings.TrimSpace(this.GetString("parent_id", "0"))
	docType, _ := this.GetInt("type", models.Document_Type_Page)
	name := strings.TrimSpace(this.GetString("name", ""))

	if spaceId == "0" {
		this.jsonError("没有选择空间！")
	}
	if parentId == "0" {
		this.jsonError("没有选择父文档！")
	}
	if name == "" {
		this.jsonError("文档名称不能为空！")
	}
	match, err := regexp.MatchString(`[\\\\/:*?\"<>、|]`, name)
	if err != nil {
		this.jsonError("文档名称格式不正确！")
	}
	if match {
		this.jsonError("文档名称格式不正确！")
	}
	if name == utils.Document_Default_FileName {
		this.jsonError("文档名称不能为 " + utils.Document_Default_FileName + " ！")
	}
	if docType != models.Document_Type_Page &&
		docType != models.Document_Type_Dir {
		this.jsonError("文档类型错误！")
	}

	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("创建保存文档失败：" + err.Error())
		this.jsonError("创建文档失败！")
	}
	if len(space) == 0 {
		this.jsonError("空间不存在！")
	}

	// check space document privilege
	_, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isEditor {
		this.jsonError("您没有权限在该空间下创建文档！")
	}

	parentDocument, err := models.DocumentModel.GetDocumentByDocumentId(parentId)
	if err != nil {
		this.ErrorLog("创建保存文档失败：" + err.Error())
		this.jsonError("创建文档失败！")
	}
	if len(parentDocument) == 0 {
		this.jsonError("父文档不存在！")
	}
	if parentDocument["type"] != fmt.Sprintf("%d", models.Document_Type_Dir) {
		this.jsonError("父文档不是目录！")
	}

	// check document name
	document, err := models.DocumentModel.GetDocumentByNameParentIdAndSpaceId(name, parentId, spaceId, docType)
	if err != nil {
		this.ErrorLog("创建保存文档失败：" + err.Error())
		this.jsonError("创建文档失败！")
	}
	if len(document) != 0 {
		this.jsonError("该文档名称已经存在！")
	}

	insertDocument := map[string]interface{}{
		"parent_id":      parentId,
		"space_id":       spaceId,
		"name":           name,
		"type":           docType,
		"path":           parentDocument["path"] + "," + parentId,
		"create_user_id": this.UserId,
		"edit_user_id":   this.UserId,
	}
	documentId, err := models.DocumentModel.Insert(insertDocument)
	if err != nil {
		this.ErrorLog("创建文档失败：" + err.Error())
		this.jsonError("创建文档失败")
	}
	this.InfoLog("创建文档 " + utils.Convert.IntToString(documentId, 10) + " 成功")
	this.jsonSuccess("创建文档成功", nil, "/document/index?document_id="+utils.Convert.IntToString(documentId, 10))
}

// document history
func (this *DocumentController) History() {

	page, _ := this.GetInt("page", 1)
	documentId := this.GetString("document_id", "0")
	number, _ := this.GetRangeInt("number", 10, 10, 100)
	limit := (page - 1) * number

	if documentId == "0" {
		this.ViewError("没有选择文档目录！")
	}

	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查看文档 " + documentId + " 修改历史失败：" + err.Error())
		this.ViewError("查看文档修改历史失败！")
	}
	if len(document) == 0 {
		this.jsonError("文档不存在！")
	}

	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 所在空间失败：" + err.Error())
		this.ViewError("查找文档失败！")
	}
	if len(space) == 0 {
		this.ViewError("文档所在空间不存在！")
	}

	// check space visit_level
	isVisit, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.ViewError("您没有权限查看该空间修改历史！")
	}

	logDocuments, err := models.LogDocumentModel.GetLogDocumentsByDocumentIdAndLimit(documentId, limit, number)
	if err != nil {
		this.ErrorLog("查看文档 " + documentId + " 修改历史失败：" + err.Error())
		this.ViewError("查看文档修改历史失败！")
	}
	count, err := models.LogDocumentModel.CountLogDocumentsByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查看文档 " + documentId + " 修改历史失败：" + err.Error())
		this.ViewError("查看文档修改历史失败！")
	}

	userIds := []string{}
	for _, logDocument := range logDocuments {
		userIds = append(userIds, logDocument["user_id"])
	}
	users, err := models.UserModel.GetUsersByUserIds(userIds)
	if err != nil {
		this.ErrorLog("查看文档 " + documentId + " 修改历史失败：" + err.Error())
		this.ViewError("查看文档修改历史失败！")
	}
	for _, logDocument := range logDocuments {
		logDocument["username"] = ""
		for _, user := range users {
			if logDocument["user_id"] == user["user_id"] {
				logDocument["username"] = user["username"]
				break
			}
		}
		logId := logDocument["log_document_id"]
		if utils.Document.HasSnapshot(spaceId, documentId, logId) {
			logDocument["has_snapshot"] = "1"
			logDocument["snapshot_size"] = formatFileSize(utils.Document.GetSnapshotFileSize(spaceId, documentId, logId))
		} else {
			logDocument["has_snapshot"] = "0"
			logDocument["snapshot_size"] = "-"
		}
	}

	// Current active document content
	_, pageFile, _ := models.DocumentModel.GetParentDocumentsByDocument(document)
	currentContent, _ := utils.Document.GetContentByPageFile(pageFile)

	this.Data["logDocuments"] = logDocuments
	this.Data["document_id"] = documentId
	this.Data["document"] = document
	this.Data["space"] = space
	this.Data["is_editor"] = isEditor
	this.Data["current_content"] = currentContent
	this.SetPaginator(number, count)
	this.viewLayout("document/history", "default")
}

// Get snapshot content
func (this *DocumentController) Snapshot() {
	documentId := this.GetString("document_id", "")
	logDocumentId := this.GetString("log_document_id", "")
	if documentId == "" || logDocumentId == "" {
		this.jsonError("参数错误")
	}

	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil || len(document) == 0 {
		this.jsonError("文档不存在")
	}
	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil || len(space) == 0 {
		this.jsonError("空间不存在")
	}
	isVisit, _, _ := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.jsonError("您没有权限查看历史快照！")
	}

	content, err := utils.Document.GetSnapshotContent(spaceId, documentId, logDocumentId)
	if err != nil {
		this.jsonError("获取快照内容失败：" + err.Error())
	}

	this.jsonSuccess("获取快照成功", map[string]interface{}{
		"document_id":     documentId,
		"log_document_id": logDocumentId,
		"content":         content,
	})
}

// Rollback to historical snapshot
func (this *DocumentController) Rollback() {
	if !this.IsPost() {
		this.jsonError("请求方式错误")
	}
	documentId := this.GetString("document_id", "")
	logDocumentId := this.GetString("log_document_id", "")
	if documentId == "" || logDocumentId == "" {
		this.jsonError("参数错误")
	}

	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil || len(document) == 0 {
		this.jsonError("文档不存在")
	}
	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil || len(space) == 0 {
		this.jsonError("空间不存在")
	}
	_, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isEditor {
		this.jsonError("您没有权限回滚该空间下的文档！")
	}

	err = models.DocumentModel.RollbackDBAndFile(documentId, spaceId, logDocumentId, this.UserId)
	if err != nil {
		this.ErrorLog(fmt.Sprintf("回滚文档 %s 至版本 %s 失败: %s", documentId, logDocumentId, err.Error()))
		this.jsonError("回滚文档失败: " + err.Error())
	}

	this.InfoLog(fmt.Sprintf("回滚文档 %s 至历史版本 %s 成功", documentId, logDocumentId))
	this.jsonSuccess("文档回滚成功！", nil, fmt.Sprintf("/document/history?document_id=%s", documentId))
}

// move document
func (this *DocumentController) Move() {

	documentId := this.GetString("document_id", "0")
	targetId := this.GetString("target_id", "0")
	moveType := this.GetString("move_type", "inner") // "prev", "next", or "inner"

	if documentId == "0" {
		this.jsonError("没有选择文档节点！")
	}
	if targetId == "0" {
		this.jsonError("没有选择目标文档节点！")
	}
	if documentId == targetId {
		this.jsonError("源文档与目标文档相同，无需移动！")
	}

	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil || len(document) == 0 {
		this.jsonError("移动文档不存在！")
	}

	targetDocument, err := models.DocumentModel.GetDocumentByDocumentId(targetId)
	if err != nil || len(targetDocument) == 0 {
		this.jsonError("目标文档不存在！")
	}

	if document["space_id"] != targetDocument["space_id"] {
		this.jsonError("文档和目标文档不在同一空间！")
	}

	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil || len(space) == 0 {
		this.jsonError("文档所在空间不存在！")
	}

	// check space document privilege
	_, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isEditor {
		this.jsonError("您没有权限移动或排序列该空间下的文档！")
	}

	// Calculate target parent
	var newParentId string
	var targetParentDocument map[string]string

	if moveType == "inner" {
		if targetDocument["type"] != fmt.Sprintf("%d", models.Document_Type_Dir) {
			this.jsonError("不能将文档拖入非目录类型的文档中！")
		}
		newParentId = targetId
		targetParentDocument = targetDocument
	} else if moveType == "prev" || moveType == "next" {
		newParentId = targetDocument["parent_id"]
		if newParentId == "0" {
			this.jsonError("无法将文档提升到空间根节点同级！")
		}
		targetParentDocument, err = models.DocumentModel.GetDocumentByDocumentId(newParentId)
		if err != nil || len(targetParentDocument) == 0 {
			this.jsonError("获取目标上级文档失败！")
		}
	} else {
		this.jsonError("未知的移动类型！")
	}

	// Prevent directory self-nesting / cycles
	if document["type"] == fmt.Sprintf("%d", models.Document_Type_Dir) {
		if newParentId == documentId {
			this.jsonError("不能将目录移动到自身！")
		}
		targetParentPath := targetParentDocument["path"] + "," + targetParentDocument["document_id"]
		if strings.Contains(targetParentPath+",", ","+documentId+",") || targetParentDocument["document_id"] == documentId {
			this.jsonError("不能将目录移动到自己的子目录下！")
		}
	}

	isParentChanged := (document["parent_id"] != newParentId)

	// If cross-level move (parent changed): move file on disk and update parent_id + path
	if isParentChanged {
		_, oldPageFile, err := models.DocumentModel.GetParentDocumentsByDocument(document)
		if err != nil {
			this.ErrorLog("查找移动文档父文档失败：" + err.Error())
			this.jsonError("获取源文档路径失败！")
		}

		newDocMap := map[string]string{
			"space_id":  spaceId,
			"parent_id": newParentId,
			"name":      document["name"],
			"type":      document["type"],
			"path":      targetParentDocument["path"] + "," + newParentId,
		}
		_, newPageFile, err := models.DocumentModel.GetParentDocumentsByDocument(newDocMap)
		if err != nil {
			this.ErrorLog("计算新文档路径失败：" + err.Error())
			this.jsonError("计算目标文档路径失败！")
		}

		absNewPageFile := utils.Document.GetAbsPageFileByPageFile(newPageFile)
		if exists, _ := utils.File.PathIsExists(absNewPageFile); exists {
			this.jsonError("目标目录下已存在同名文档，无法移动！")
		}

		// Move on disk
		err = utils.Document.Move(oldPageFile, newPageFile, utils.Convert.StringToInt(document["type"]))
		if err != nil {
			this.ErrorLog("移动物理文件失败：" + err.Error())
			this.jsonError("移动文档物理文件失败！")
		}

		// Update database parent_id and path
		newPath := targetParentDocument["path"] + "," + newParentId
		updateValue := map[string]interface{}{
			"parent_id":    newParentId,
			"path":         newPath,
			"edit_user_id": this.UserId,
			"update_time":  time.Now().Unix(),
		}
		_, err = models.DocumentModel.Update(documentId, updateValue, fmt.Sprintf("移动文档到 %s", targetParentDocument["name"]), spaceId)
		if err != nil {
			this.ErrorLog("更新移动文档数据库记录失败：" + err.Error())
			this.jsonError("更新文档层级记录失败！")
		}

		// If moved item is a directory, update all descendants paths
		if document["type"] == fmt.Sprintf("%d", models.Document_Type_Dir) {
			oldPrefix := document["path"] + "," + documentId
			newPrefix := newPath + "," + documentId
			_ = models.DocumentModel.UpdateDescendantsPath(spaceId, oldPrefix, newPrefix)
		}
	}

	// Now handle sequence ordering
	var finalSequence int
	if moveType == "inner" {
		maxSeq, _ := models.DocumentModel.GetDocumentMaxSequence(newParentId, spaceId)
		finalSequence = maxSeq + 1
	} else if moveType == "prev" {
		targetSeq := utils.Convert.StringToInt(targetDocument["sequence"])
		_ = models.DocumentModel.ShiftSequence(spaceId, newParentId, targetSeq, "gte", documentId)
		finalSequence = targetSeq
	} else if moveType == "next" {
		targetSeq := utils.Convert.StringToInt(targetDocument["sequence"])
		_ = models.DocumentModel.ShiftSequence(spaceId, newParentId, targetSeq, "gt", documentId)
		finalSequence = targetSeq + 1
	}

	// Update sequence for the moved document
	seqUpdateValue := map[string]interface{}{
		"sequence":     finalSequence,
		"edit_user_id": this.UserId,
	}
	_, err = models.DocumentModel.Update(documentId, seqUpdateValue, "更新文档排序", spaceId)
	if err != nil {
		this.ErrorLog("更新文档排序失败：" + err.Error())
	}

	this.InfoLog(fmt.Sprintf("文档 %s 成功移动到 parent_id=%s sequence=%d", documentId, newParentId, finalSequence))
	this.jsonSuccess("文档层级与排序更新成功", map[string]interface{}{
		"document_id": documentId,
		"parent_id":   newParentId,
		"sequence":    finalSequence,
	}, fmt.Sprintf("/document/index?document_id=%s", documentId))
}

// delete document
func (this *DocumentController) Delete() {

	documentId := this.GetString("document_id", "0")

	if documentId == "0" {
		this.jsonError("没有选择文档！")
	}
	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("删除文档失败：" + err.Error())
		this.jsonError("删除文档失败！")
	}
	if len(document) == 0 {
		this.jsonError("文档不存在！")
	}
	if document["type"] == fmt.Sprintf("%d", models.Document_Type_Dir) {
		childDocs, err := models.DocumentModel.GetDocumentsByParentId(document["document_id"])
		if err != nil {
			this.ErrorLog("删除文档失败：" + err.Error())
			this.jsonError("删除文档失败！")
		}
		if len(childDocs) > 0 {
			this.jsonError("请先删除或移动目录下所有文档！")
		}
	}
	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(document["space_id"])
	if err != nil {
		this.ErrorLog("删除文档失败：" + err.Error())
		this.jsonError("删除文档失败！")
	}
	if len(space) == 0 {
		this.jsonError("文档空间不存在！")
	}
	// check space document privilege
	_, _, isManager := this.GetDocumentPrivilege(space)
	if !isManager {
		this.jsonError("您没有权限删除该空间下的文档！")
	}

	_, pageFile, err := models.DocumentModel.GetParentDocumentsByDocument(document)
	if err != nil {
		this.ErrorLog("删除文档 " + documentId + " 失败：" + err.Error())
		this.jsonError("删除文档失败！")
	}

	err = models.DocumentModel.DeleteDBAndFile(documentId, spaceId, this.UserId, pageFile, document["type"])
	if err != nil {
		this.ErrorLog("删除文档 " + documentId + " 失败：" + err.Error())
		this.jsonError("删除文档失败！")
	}

	// delete attachment
	err = models.AttachmentModel.DeleteAttachmentsDBFileByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("删除文档 " + documentId + " 附件失败：" + err.Error())
	}

	// 删除文档索引
	go func(documentId string) {
		services.DocIndexService.ForceDelDocIdIndex(documentId)
	}(documentId)

	this.InfoLog("删除文档 " + documentId + " 成功")
	this.jsonSuccess("删除文档成功", "", "/document/index?document_id="+document["parent_id"])
}
