package controllers

import (
	"fmt"
	"github.com/chaiyd/mm-wiki/app"
	"github.com/chaiyd/mm-wiki/app/models"
	"github.com/chaiyd/mm-wiki/app/utils"
	"os"
	"path"
)

type AttachmentController struct {
	BaseController
}

func formatFileSize(bytes int64) string {
	if bytes <= 0 {
		return "-"
	}
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024.0)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024.0*1024.0))
	}
	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024.0*1024.0*1024.0))
}

func (this *AttachmentController) Page() {

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
	isVisit, isEditor, isManager := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.ViewError("您没有权限访问该空间下的文档！")
	}

	scope := this.GetString("scope", "doc")
	if scope != "space" {
		scope = "doc"
	}

	// 1. 获取当前文档附件列表与数量
	docAttachments, err := models.AttachmentModel.GetAttachmentsByDocumentIdAndSource(documentId, models.Attachment_Source_Default)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 附件失败：" + err.Error())
		this.ViewError("查找文档附件失败！")
	}
	docCount := len(docAttachments)

	// 2. 获取本空间全部文档与空间附件列表与数量
	spaceDocuments, err := models.DocumentModel.GetDocumentsBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找空间文档失败：" + err.Error())
		this.ViewError("查找空间文档失败！")
	}
	docNameMap := make(map[string]string)
	allDocIds := []string{}
	for _, doc := range spaceDocuments {
		dId := doc["document_id"]
		allDocIds = append(allDocIds, dId)
		docNameMap[dId] = doc["name"]
	}

	spaceAttachments, err := models.AttachmentModel.GetAttachmentsByDocumentIdsAndSource(allDocIds, models.Attachment_Source_Default)
	if err != nil {
		this.ErrorLog("查找空间附件失败：" + err.Error())
		this.ViewError("查找空间附件失败！")
	}
	spaceCount := len(spaceAttachments)

	// 根据 scope 决定渲染列表
	var attachments []map[string]string
	if scope == "space" {
		attachments = spaceAttachments
	} else {
		attachments = docAttachments
	}

	// 填充所属文档名与物理文件大小
	userIds := []string{}
	for _, attachment := range attachments {
		userIds = append(userIds, attachment["user_id"])
		dName := docNameMap[attachment["document_id"]]
		if dName == "" {
			dName = document["name"]
		}
		attachment["document_name"] = dName

		fullPath := path.Join(app.DocumentAbsDir, attachment["path"])
		if fi, sErr := os.Stat(fullPath); sErr == nil {
			attachment["file_size"] = formatFileSize(fi.Size())
		} else {
			attachment["file_size"] = "-"
		}
	}

	// 获取上传用户名
	users, err := models.UserModel.GetUsersByUserIds(userIds)
	if err == nil {
		usernameMap := make(map[string]string)
		for _, user := range users {
			usernameMap[user["user_id"]] = user["username"]
		}
		for _, attachment := range attachments {
			attachment["username"] = usernameMap[attachment["user_id"]]
		}
	}

	this.Data["attachments"] = attachments
	this.Data["document_id"] = documentId
	this.Data["document"] = document
	this.Data["space_id"] = spaceId
	this.Data["space"] = space
	this.Data["scope"] = scope
	this.Data["doc_count"] = docCount
	this.Data["space_count"] = spaceCount
	this.Data["is_upload"] = isEditor
	this.Data["is_delete"] = isManager
	this.viewLayout("attachment/page", "attachment")
}

func (this *AttachmentController) Upload() {

	if !this.IsPost() {
		this.ViewError("请求方式有误！", "/space/index")
	}
	documentId := this.GetString("document_id", "")
	if documentId == "" {
		this.uploadJsonError("参数错误！", "/space/index")
	}

	// handle document
	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查找空间文档 " + documentId + " 失败：" + err.Error())
		this.uploadJsonError("查找文档失败！")
	}
	if len(document) == 0 {
		this.uploadJsonError("文档不存在！")
	}

	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 所在空间失败：" + err.Error())
		this.uploadJsonError("查找文档失败！")
	}
	if len(space) == 0 {
		this.uploadJsonError("文档所在空间不存在！")
	}
	// check space visit_level
	_, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isEditor {
		this.uploadJsonError("您没有权限操作该空间下的文档！")
	}

	// handle upload
	f, h, err := this.GetFile("attachment")
	if err != nil {
		this.ErrorLog("上传附件数据错误: " + err.Error())
		this.uploadJsonError("上传附件数据错误")
		return
	}
	if h == nil || f == nil {
		this.ErrorLog("上传附件错误")
		this.uploadJsonError("上传附件错误")
		return
	}
	_ = f.Close()

	// file save dir
	saveDir := fmt.Sprintf("%s/%s/%s", app.AttachmentAbsDir, spaceId, documentId)
	ok, _ := utils.File.PathIsExists(saveDir)
	if !ok {
		err := os.MkdirAll(saveDir, 0777)
		if err != nil {
			this.ErrorLog("上传附件错误: " + err.Error())
			this.uploadJsonError("上传附件失败")
			return
		}
	}
	// check file is exists
	attachmentFile := path.Join(saveDir, h.Filename)
	ok, _ = utils.File.PathIsExists(attachmentFile)
	if ok {
		this.uploadJsonError("该附件已经存在！")
	}
	// save file
	err = this.SaveToFile("attachment", attachmentFile)
	if err != nil {
		this.ErrorLog("附件保存失败: " + err.Error())
		this.uploadJsonError("附件保存失败")
	}

	// insert db
	attachment := map[string]interface{}{
		"user_id":     this.UserId,
		"document_id": documentId,
		"name":        h.Filename,
		"path":        fmt.Sprintf("attachment/%s/%s/%s", spaceId, documentId, h.Filename),
		"source":      models.Attachment_Source_Default,
	}
	_, err = models.AttachmentModel.Insert(attachment, spaceId)
	if err != nil {
		_ = os.Remove(attachmentFile)
		this.ErrorLog("上传附件错误: " + err.Error())
		this.uploadJsonError("附件信息保存失败")
	}

	scope := this.GetString("scope", "doc")
	this.InfoLog(fmt.Sprintf("文档 %s 上传附件 %s 成功", documentId, h.Filename))
	this.jsonSuccess("附件上传成功", "", fmt.Sprintf("/attachment/page?document_id=%s&scope=%s", documentId, scope))
}

func (this *AttachmentController) Delete() {

	if !this.IsPost() {
		this.ViewError("请求方式有误！", "/space/index")
	}
	attachmentId := this.GetString("attachment_id", "")
	if attachmentId == "" {
		this.jsonError("没有选择附件！")
	}

	attachment, err := models.AttachmentModel.GetAttachmentByAttachmentId(attachmentId)
	if err != nil {
		this.ErrorLog("删除附件 " + attachmentId + " 失败: " + err.Error())
		this.jsonError("删除附件失败")
	}
	if len(attachment) == 0 {
		this.jsonError("附件不存在")
	}

	documentId := attachment["document_id"]
	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查找附件所属空间文档 " + documentId + " 失败：" + err.Error())
		this.jsonError("查找附件所属文档失败！")
	}
	if len(document) == 0 {
		this.jsonError("附件所属文档不存在！")
	}

	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找附件所属文档 " + documentId + " 所在空间失败：" + err.Error())
		this.jsonError("查找附件所属文档空间失败！")
	}
	if len(space) == 0 {
		this.jsonError("附件所属文档所在空间不存在！")
	}
	// check space visit_level
	_, _, isManager := this.GetDocumentPrivilege(space)
	if !isManager {
		this.jsonError("您没有权限删除该空间下的文档！")
	}
	attachmentName := attachment["name"]
	attachmentSource := attachment["source"]

	// delete db
	err = models.AttachmentModel.DeleteAttachmentDBFile(attachmentId)
	if err != nil {
		this.ErrorLog("删除附件 " + attachmentId + " 失败: " + err.Error())
		this.jsonError("删除附件失败")
	}

	// update document log
	go func(userId string, documentId string, attachmentName string, spaceId string) {
		_, _ = models.LogDocumentModel.UpdateAction(userId, documentId, "删除了附件 "+attachmentName, spaceId)
	}(this.UserId, documentId, attachmentName, spaceId)

	scope := this.GetString("scope", "doc")
	redirect := fmt.Sprintf("/attachment/page?document_id=%s&scope=%s", documentId, scope)
	if attachmentSource == fmt.Sprintf("%d", models.Attachment_Source_Image) {
		redirect = fmt.Sprintf("/attachment/image?document_id=%s&scope=%s", documentId, scope)
	}

	this.InfoLog("删除文档 " + documentId + " 附件 " + attachmentName + " 成功")
	this.jsonSuccess("删除成功", nil, redirect)
}

func (this *AttachmentController) Download() {

	attachmentId := this.GetString("attachment_id", "")
	if attachmentId == "" {
		this.ViewError("没有选择附件！")
	}

	attachment, err := models.AttachmentModel.GetAttachmentByAttachmentId(attachmentId)
	if err != nil {
		this.ErrorLog("下载附件 " + attachmentId + " 失败: " + err.Error())
		this.ViewError("下载附件失败")
	}
	if len(attachment) == 0 {
		this.ViewError("附件不存在")
	}

	documentId := attachment["document_id"]
	document, err := models.DocumentModel.GetDocumentByDocumentId(documentId)
	if err != nil {
		this.ErrorLog("查找附件所属空间文档 " + documentId + " 失败：" + err.Error())
		this.ViewError("查找附件所属文档失败！")
	}
	if len(document) == 0 {
		this.ViewError("附件所属文档不存在！")
	}

	spaceId := document["space_id"]
	space, err := models.SpaceModel.GetSpaceBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找附件所属文档 " + documentId + " 所在空间失败：" + err.Error())
		this.ViewError("查找附件所属文档空间失败！")
	}
	if len(space) == 0 {
		this.ViewError("附件所属文档所在空间不存在！")
	}
	// check space visit_level
	isVisit, _, _ := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.ViewError("您没有权限访问或下载该空间下的资料！")
	}
	attachmentFilePath := path.Join(app.DocumentAbsDir, attachment["path"])
	attachmentName := attachment["name"]

	this.Ctx.Output.Download(attachmentFilePath, attachmentName)
}

func (this *AttachmentController) Image() {

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
	isVisit, isEditor, _ := this.GetDocumentPrivilege(space)
	if !isVisit {
		this.ViewError("您没有权限访问该空间下的文档！")
	}

	scope := this.GetString("scope", "doc")
	if scope != "space" {
		scope = "doc"
	}

	// 1. 获取当前文档的图片列表与数量
	docImages, err := models.AttachmentModel.GetAttachmentsByDocumentIdAndSource(documentId, models.Attachment_Source_Image)
	if err != nil {
		this.ErrorLog("查找文档 " + documentId + " 图片失败：" + err.Error())
		this.ViewError("查找文档图片失败！")
	}
	docCount := len(docImages)

	// 2. 获取本空间全部文档与空间图片列表与数量
	spaceDocuments, err := models.DocumentModel.GetDocumentsBySpaceId(spaceId)
	if err != nil {
		this.ErrorLog("查找空间文档失败：" + err.Error())
		this.ViewError("查找空间文档失败！")
	}
	docNameMap := make(map[string]string)
	allDocIds := []string{}
	for _, doc := range spaceDocuments {
		dId := doc["document_id"]
		allDocIds = append(allDocIds, dId)
		docNameMap[dId] = doc["name"]
	}

	spaceImages, err := models.AttachmentModel.GetAttachmentsByDocumentIdsAndSource(allDocIds, models.Attachment_Source_Image)
	if err != nil {
		this.ErrorLog("查找空间图片失败：" + err.Error())
		this.ViewError("查找空间图片失败！")
	}
	spaceCount := len(spaceImages)

	// 根据 scope 决定渲染列表
	var attachments []map[string]string
	if scope == "space" {
		attachments = spaceImages
	} else {
		attachments = docImages
	}

	// 补充所属文档名称与文件大小
	userIds := []string{}
	for _, attachment := range attachments {
		userIds = append(userIds, attachment["user_id"])
		dName := docNameMap[attachment["document_id"]]
		if dName == "" {
			dName = document["name"]
		}
		attachment["document_name"] = dName

		fullPath := path.Join(app.DocumentAbsDir, attachment["path"])
		if fi, sErr := os.Stat(fullPath); sErr == nil {
			attachment["file_size"] = formatFileSize(fi.Size())
		} else {
			attachment["file_size"] = "-"
		}
	}

	// 获取用户名映射
	users, err := models.UserModel.GetUsersByUserIds(userIds)
	if err == nil {
		usernameMap := make(map[string]string)
		for _, user := range users {
			usernameMap[user["user_id"]] = user["username"]
		}
		for _, attachment := range attachments {
			attachment["username"] = usernameMap[attachment["user_id"]]
		}
	}

	this.Data["attachments"] = attachments
	this.Data["document_id"] = documentId
	this.Data["document"] = document
	this.Data["space_id"] = spaceId
	this.Data["space"] = space
	this.Data["scope"] = scope
	this.Data["doc_count"] = docCount
	this.Data["space_count"] = spaceCount
	this.Data["is_upload"] = isEditor
	this.Data["is_delete"] = isEditor
	this.viewLayout("attachment/image", "attachment")
}
