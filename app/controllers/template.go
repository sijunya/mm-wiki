package controllers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chaiyd/mm-wiki/app"
	"github.com/chaiyd/mm-wiki/app/models"
	"github.com/chaiyd/mm-wiki/app/utils"

	"github.com/astaxie/beego"
)

type TemplateController struct {
	beego.Controller
	UserId         string
	User           map[string]string
	controllerName string
	actionName     string
}

type JsonResponse struct {
	Code     int                    `json:"code"`
	Message  interface{}            `json:"message"`
	Data     interface{}            `json:"data"`
	Redirect map[string]interface{} `json:"redirect"`
}

// prepare
func (this *TemplateController) Prepare() {
	controllerName, action := this.GetControllerAndAction()
	this.controllerName = strings.ToLower(controllerName[0 : len(controllerName)-10])
	this.actionName = strings.ToLower(action)
	this.Data["navName"] = this.controllerName
	this.Data["version"] = app.Version
	this.Data["copyright"] = app.CopyRight

	if this.controllerName == "author" {
		return
	}

	if !this.isLogin() {
		if this.IsAjax() {
			this.JsonError("未登录或登录已失效！", nil, "/author/index")
		} else {
			this.Redirect("/author/index", 302)
		}
		this.StopRun()
	}

	if !this.checkAccess() {
		if this.IsPost() {
			this.JsonError("抱歉，您没有权限操作！", nil, "/system/main/index")
		} else {
			this.ViewError("您没有权限访问该页面！", "/system/main/index")
		}
		this.StopRun()
	}
}

// check is login
func (this *TemplateController) isLogin() bool {

	if this.controllerName == "page" && this.actionName == "display" {
		return true
	}

	passport := beego.AppConfig.String("author::passport")
	cookie := this.Ctx.GetCookie(passport)
	// cookie is empty
	if cookie == "" {
		return false
	}

	cookieValue, _ := utils.Encrypt.Base64Decode(cookie)
	identifyList := strings.Split(cookieValue, "@")
	if cookieValue == "" || len(identifyList) != 2 {
		return false
	}
	username := identifyList[0]
	identify := identifyList[1]

	var userValue map[string]string

	user := this.GetSession("author")
	if user != nil {
		if uMap, ok := user.(map[string]string); ok {
			if username == uMap["username"] {
				userValue = uMap
			}
		}
	}

	// 如果 Session 丢失（例如服务重启、内存回收、Session GC 过期、浏览器关闭），从有效 Cookie 自动免密恢复会话
	if userValue == nil {
		dbUser, err := models.UserModel.GetUserByUsername(username)
		if err != nil || len(dbUser) == 0 {
			return false
		}
		// 检查用户是否被禁用或删除
		if dbUser["is_delete"] != "0" || dbUser["is_forbidden"] != "0" {
			return false
		}
		userValue = dbUser
	}

	// 校验 identify
	// 兼容稳定签名（用户名+密码哈希）、纯密码哈希与旧版带 UA 签名，防止浏览器静默升级导致误登出，同时密码修改立即全端失效
	expectedUserPass := utils.Encrypt.Md5Encode(userValue["username"] + userValue["password"])
	expectedPure := utils.Encrypt.Md5Encode(userValue["password"])
	expectedWithUA := utils.Encrypt.Md5Encode(this.Ctx.Request.UserAgent() + userValue["password"])
	if identify != expectedUserPass && identify != expectedPure && identify != expectedWithUA {
		return false
	}

	// 刷新用户信息与 Session
	newUser, err := models.UserModel.GetUserByUserId(userValue["user_id"])
	if err != nil || len(newUser) == 0 || newUser["is_delete"] != "0" || newUser["is_forbidden"] != "0" {
		this.ErrorLog("登录状态刷新用户失败")
		return false
	}
	this.SetSession("author", newUser)
	this.User = newUser
	this.UserId = this.User["user_id"]

	// 滑动窗口自动续期 Cookie（30天），保证活跃用户在使用期间永远不会突然掉线
	cookieExpired, _ := beego.AppConfig.Int64("author::cookie_expired")
	if cookieExpired <= 0 {
		cookieExpired = 30 * 86400 // 默认 30 天
	}
	// 统一刷新为最新稳定签名的 passport cookie
	stableIdentify := utils.Encrypt.Md5Encode(newUser["username"] + newUser["password"])
	newPassportValue := utils.Encrypt.Base64Encode(newUser["username"] + "@" + stableIdentify)
	this.Ctx.SetCookie(passport, newPassportValue, cookieExpired)

	// 查找系统名称
	systemName := models.ConfigModel.GetConfigValueByKey(models.ConfigKeySystemName, "Markdown Mini Wiki")
	this.Data["system_name"] = systemName
	this.Data["login_user_id"] = this.UserId
	this.Data["login_username"] = this.User["username"]
	this.Data["login_role_id"] = this.User["role_id"]

	// success
	return true
}

// check access
func (this *TemplateController) checkAccess() bool {
	path := this.Ctx.Request.URL.Path
	mca := strings.Split(strings.Trim(path, "/"), "/")

	// must /system/controller/action
	if (len(mca) >= 3) && (strings.ToLower(mca[0]) == "system") {
		this.Data["navName"] = "system"
		// no check '/system/main/index' '/system/main/default'
		if (this.controllerName == "main" && this.actionName == "index") || this.controllerName == "main" && this.actionName == "default" {
			return true
		}
		if this.IsRoot() {
			return true
		}
		_, controllers, err := models.PrivilegeModel.GetTypePrivilegesByUserId(this.UserId)
		if err != nil {
			this.ErrorLog("获取用户 " + this.UserId + " 权限失败：" + err.Error())
			return false
		}
		for _, controller := range controllers {
			action := strings.ToLower(controller["action"])
			controller := strings.ToLower(controller["controller"])
			if this.controllerName == controller && this.actionName == action {
				return true
			}
		}
		return false
	}
	return true
}

// view layout
func (this *TemplateController) ViewLayout(viewName, layout string) {
	this.Layout = layout + ".html"
	this.TplName = viewName + ".html"
	this.Data["title"] = "MM-Wiki"
	this.Data["copyright"] = app.CopyRight
	this.Render()
}

// view layout
func (this *TemplateController) ViewError(content string, redirect ...string) {
	this.Layout = "error/default.html"
	this.TplName = "layouts/default.html"
	var url = ""
	var sleep = "2"
	if len(redirect) == 1 {
		url = redirect[0]
	}
	if len(redirect) > 1 {
		sleep = redirect[1]
	}
	if content == "" {
		content = "操作失败"
	}
	this.Data["content"] = content
	this.Data["url"] = url
	this.Data["sleep"] = sleep
	this.Data["copyright"] = app.CopyRight
	this.Render()
}

// return json success
func (this *TemplateController) JsonSuccess(message interface{}, data ...interface{}) {
	url := ""
	sleep := 300
	var _data interface{}
	if len(data) > 0 {
		_data = data[0]
	}
	if len(data) > 1 {
		url = data[1].(string)
	}
	if len(data) > 2 {
		sleep = data[2].(int)
	}
	this.Data["json"] = JsonResponse{
		Code:    1,
		Message: message,
		Data:    _data,
		Redirect: map[string]interface{}{
			"url":   url,
			"sleep": sleep,
		},
	}

	j, err := json.MarshalIndent(this.Data["json"], "", "\t")
	if err != nil {
		this.Abort(err.Error())
	} else {
		this.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
		this.Abort(string(j))
	}
}

// return json error
func (this *TemplateController) JsonError(message interface{}, data ...interface{}) {
	url := ""
	sleep := 1000
	var _data interface{}
	if len(data) > 0 {
		_data = data[0]
	}
	if len(data) > 1 {
		url = data[1].(string)
	}
	if len(data) > 2 {
		sleep = data[2].(int)
	}
	this.Data["json"] = JsonResponse{
		Code:    0,
		Message: message,
		Data:    _data,
		Redirect: map[string]interface{}{
			"url":   url,
			"sleep": sleep,
		},
	}
	j, err := json.MarshalIndent(this.Data["json"], "", " \t")
	if err != nil {
		this.Abort(err.Error())
	} else {
		this.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
		this.Abort(string(j))
	}
}

// get client ip
func (this *TemplateController) GetClientIp() string {
	s := this.Ctx.Input.IP()
	if s == "" {
		s = strings.Split(this.Ctx.Request.RemoteAddr, ":")[0]
	}
	return s
	//s := strings.Split(this.Ctx.Request.RemoteAddr, ":")
	//return s[0]
}

// paginator
func (this *TemplateController) SetPaginator(per int, nums int64) *utils.Paginator {
	p := utils.NewPaginator(this.Ctx.Request, per, nums)
	this.Data["paginator"] = p
	return p
}

// is Post
func (this *TemplateController) IsPost() bool {
	return this.Ctx.Input.Method() == "POST"
}

// is Get
func (this *TemplateController) IsGet() bool {
	return this.Ctx.Input.Method() == "GET"
}

// 是否是超级管理员
func (this *TemplateController) IsRoot() bool {
	return this.User["role_id"] == fmt.Sprintf("%d", models.Role_Root_Id)
}

func (this *TemplateController) GetRangeInt(key string, def int, min int, max int) (n int, err error) {
	n, err = this.GetInt(key, def)
	if err != nil {
		return
	}
	if n < min {
		n = min
	}
	if n > max {
		n = max
	}
	return n, nil
}

func (this *TemplateController) GetDocumentPrivilege(space map[string]string) (isVisit, isEditor, isManager bool) {

	if this.IsRoot() {
		return true, true, true
	}
	spaceUser, _ := models.SpaceUserModel.GetSpaceUserBySpaceIdAndUserId(space["space_id"], this.UserId)
	if len(spaceUser) == 0 {
		if space["visit_level"] == models.Space_VisitLevel_Private {
			return false, false, false
		} else {
			return true, false, false
		}
	}
	if spaceUser["privilege"] == fmt.Sprintf("%d", models.SpaceUser_Privilege_Editor) {
		return true, true, false
	}
	if spaceUser["privilege"] == fmt.Sprintf("%d", models.SpaceUser_Privilege_Manager) {
		return true, true, true
	}
	return true, false, false
}

// insert action log
func (this *TemplateController) RecordLog(message string, level int) {
	userAgent := this.Ctx.Request.UserAgent()
	referer := this.Ctx.Request.Referer()
	getParams := this.Ctx.Request.URL.String()
	path := this.Ctx.Request.URL.Path
	this.Ctx.Request.ParseForm()
	postParamsMap := map[string][]string(this.Ctx.Request.PostForm)
	postParams, _ := json.Marshal(postParamsMap)
	user := this.GetSession("author").(map[string]string)

	logValue := map[string]interface{}{
		"level":       level,
		"path":        path,
		"get":         getParams,
		"post":        string(postParams),
		"message":     message,
		"ip":          this.GetClientIp(),
		"user_agent":  userAgent,
		"referer":     referer,
		"user_id":     user["user_id"],
		"username":    user["username"],
		"create_time": time.Now().Unix(),
	}

	models.LogModel.Insert(logValue)
}

func (this *TemplateController) ErrorLog(message string) {
	this.RecordLog(message, models.Log_Level_Error)
}

func (this *TemplateController) WarningLog(message string) {
	this.RecordLog(message, models.Log_Level_Warning)
}

func (this *TemplateController) InfoLog(message string) {
	this.RecordLog(message, models.Log_Level_Info)
}

func (this *TemplateController) DebugLog(message string) {
	this.RecordLog(message, models.Log_Level_Debug)
}

func (this *TemplateController) GetLogInfoByCtx() map[string]interface{} {
	userAgent := this.Ctx.Request.UserAgent()
	referer := this.Ctx.Request.Referer()
	getParams := this.Ctx.Request.URL.String()
	path := this.Ctx.Request.URL.Path
	this.Ctx.Request.ParseForm()
	postParamsMap := map[string][]string(this.Ctx.Request.PostForm)
	postParams, _ := json.Marshal(postParamsMap)
	user := this.GetSession("author").(map[string]string)
	logValue := map[string]interface{}{
		"level":       models.Log_Level_Info,
		"path":        path,
		"get":         getParams,
		"post":        string(postParams),
		"message":     "",
		"ip":          this.GetClientIp(),
		"user_agent":  userAgent,
		"referer":     referer,
		"user_id":     user["user_id"],
		"username":    user["username"],
		"create_time": time.Now().Unix(),
	}
	return logValue
}
