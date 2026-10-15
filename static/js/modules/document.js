/**
 * Copyright (c) 2018 phachon@163.com
 * // var treeData = [
 //     { id:10, pId:0, name:"首页", open:true},
 //     { id:11, pId:0, name:"技术部"},
 //     { id:12, pId:0, name:"产品部"},
 //     { id:13, pId:0, name:"运营部", isParent:true},
 //     { id:111, pId:11, name:"前端开发组", isParent:true},
 //     { id:112, pId:11, name:"UED组", isParent:true},
 //     { id:113, pId:11, name:"后台开发组", isParent:true},
 //     { id:114, pId:11, name:"应用开发组", isParent:true},
 //     { id:115, pId:11, name:"基础研发组", isParent:true},
 //
 //     { id:121, pId:111, name:"前端CSS规范"},
 //     { id:122, pId:112, name:"UED 设计标准"},
 //     { id:123, pId:113, name:"PHP 代码规范"},
 //     { id:124, pId:114, name:"广告应用架构"},
 //     { id:125, pId:115, name:"消息中间件的设计"},
 //
 //     { id:121, pId:12, name:"视频组", isParent:true},
 //     { id:122, pId:12, name:"社交组", isParent:true},
 //     { id:123, pId:12, name:"会员组", isParent:true},
 //     { id:124, pId:12, name:"广告组", isParent:true}
 // ];
 */
var Document = {

    /**
     * list nav
     */
    ListTree: function (element, treeData, defaultId, isEditor, isDelete) {

        //配置信息
        var setting = {
            view: {
                showLine: false,
                showIcon: showIconForTree,
                addHoverDom: addHoverDom,
                removeHoverDom: removeHoverDom
            },
            edit: {
                enable: isEditor,
                showRemoveBtn: isDelete,
                showRenameBtn: false,
                drag: {
                    isCopy: false,
                    isMove: isEditor,
                    prev: canPrev,
                    next: canNext,
                    inner: canInner
                }
            },
            data: {
                simpleData: {
                    enable: true
                }
            },
            callback: {
                beforeClick: beforeClick,
                onClick: onClick,

                beforeEditName: beforeEditName,

                beforeRemove: beforeRemove,
                onRemove: onRemove,

                beforeRename: beforeRename,
                onRename: onRename,

                beforeDrag: beforeDrag,
                onDrag: onDrag,
                beforeDrop: beforeDrop,
                onDrop: onDrop,
                onRightClick: onRightClick
            }
        };

        if (isDelete == false) {
            setting.edit.showRemoveBtn = false;
        }

        function switchDocument(docId, docName, pushHistory) {
            if (!docId) return;

            // 1. 更新浏览器历史记录与地址栏
            if (pushHistory !== false) {
                var newUrl = "/document/index?document_id=" + docId;
                if (location.pathname + location.search !== newUrl) {
                    history.pushState({ document_id: docId, name: docName }, "", newUrl);
                }
            }

            // 2. 更新网页标题
            if (docName) {
                document.title = docName + " - MM-Wiki";
            }

            // 3. 联动 zTree 侧边栏选中高亮
            var zTreeMenu = $.fn.zTree.getZTreeObj("dir_tree");
            if (zTreeMenu) {
                var node = zTreeMenu.getNodeByParam("id", docId);
                if (node) {
                    zTreeMenu.selectNode(node, false);
                    zTreeMenu.expandNode(node, true, false, false);
                    if (!docName && node.name) {
                        document.title = node.name + " - MM-Wiki";
                    }
                }
            }

            // 4. 极速切换 iframe 视窗（平滑过渡，无整页白屏）
            var frame = document.getElementById("mainFrame");
            if (frame) {
                frame.style.opacity = "0.75";
                frame.onload = function () {
                    frame.style.opacity = "1";
                };
                frame.src = "/page/view?document_id=" + docId;
            }

            // 5. 移动端收起抽屉
            if ($(window).width() <= 768 && window.toggleMobileDrawer) {
                window.toggleMobileDrawer(false);
            }
        }

        window.switchDocument = switchDocument;
        window.syncActiveDocument = function (docId, docName) {
            if (docName) {
                document.title = docName + " - MM-Wiki";
            }
            var zTreeMenu = $.fn.zTree.getZTreeObj("dir_tree");
            if (zTreeMenu) {
                var node = zTreeMenu.getNodeByParam("id", docId);
                if (node) {
                    zTreeMenu.selectNode(node, false);
                }
            }
        };

        function beforeClick(treeId, treeNode) {
            switchDocument(treeNode.id, treeNode.name, true);
            return false;
        }

        function onClick() {
            console.log("点击节点后....");
        }

        function beforeEditName(treeId, treeNode) {
            console.log("开始修改节点...");
            return true;
        }

        function beforeRename(treeId, treeNode, newName) {
            console.log("修改完成提交...");
            return true;
        }

        function onRename(e, treeId, treeNode, isCancel) {
            console.log("修改之后刷新...");
            return true;
        }

        function beforeRemove(treeId, treeNode) {
            console.log("删除节点前...");
            console.log(treeNode);
            if (treeNode.isParent) {
                if (treeNode.children && treeNode.children.length > 0) {
                    Layers.failedMsg("请先删除或移动目录下所有文档！");
                    return false;
                }
            }

            var title = '<i class="fa fa-volume-up"></i> 确定要删除文档吗？';
            layer.confirm(title, {
                btn: ['是', '否'],
                skin: Layers.skin,
                btnAlign: 'c',
                title: "<i class='fa fa-warning'></i><strong> 警告</strong>"
            }, function () {
                Common.ajaxSubmit("/document/delete?document_id=" + treeNode.id);
            }, function () {

            });

            return false;
        }

        function onRemove(e, treeId, treeNode) {
            console.log("删除节点...");
            return false;
        }

        function showIconForTree(treeId, treeNode) {
            return true;
        }

        function isDescendant(parent, target) {
            if (!parent || !target) return false;
            if (parent.tId === target.tId) return true;
            var p = target.getParentNode ? target.getParentNode() : null;
            while (p) {
                if (p.tId === parent.tId) return true;
                p = p.getParentNode ? p.getParentNode() : null;
            }
            return false;
        }

        function canPrev(treeId, nodes, targetNode) {
            if (!targetNode || !isEditor || !nodes || nodes.length === 0) return false;
            var moveNode = nodes[0];
            if (moveNode.isParent && isDescendant(moveNode, targetNode)) return false;
            return true;
        }

        function canNext(treeId, nodes, targetNode) {
            if (!targetNode || !isEditor || !nodes || nodes.length === 0) return false;
            var moveNode = nodes[0];
            if (moveNode.isParent && isDescendant(moveNode, targetNode)) return false;
            return true;
        }

        function canInner(treeId, nodes, targetNode) {
            if (!targetNode || !isEditor || !nodes || nodes.length === 0) return false;
            if (!targetNode.isParent) return false;
            var moveNode = nodes[0];
            if (moveNode.isParent && isDescendant(moveNode, targetNode)) return false;
            return true;
        }

        function beforeDrag(treeId, treeNodes) {
            if (!isEditor) {
                return false;
            }
            return true;
        }

        function onDrag() {
            return true;
        }

        function beforeDrop(treeId, treeNodes, targetNode, moveType, isCopy) {
            if (!isEditor || !targetNode || !moveType || !treeNodes || treeNodes.length === 0) {
                return false;
            }
            var moveNode = treeNodes[0];
            if (moveNode.tId === targetNode.tId) {
                return false;
            }
            // 不能移动到自身或者子目录下
            if (moveNode.isParent && isDescendant(moveNode, targetNode)) {
                layer.msg("不能将目录移动到自己或其子目录下！", { icon: 2, time: 2000 });
                return false;
            }
            // 普通文档无法作为父级目录
            if (moveType === "inner" && !targetNode.isParent) {
                layer.msg("普通文档不可作为父级目录！", { icon: 2, time: 2000 });
                return false;
            }
            return true;
        }

        function onDrop(event, treeId, treeNodes, targetNode, moveType, isCopy) {
            if (!targetNode || !moveType || !treeNodes || treeNodes.length === 0) {
                return false;
            }
            var moveNode = treeNodes[0];
            var postData = {
                document_id: moveNode.id,
                target_id: targetNode.id,
                move_type: moveType
            };

            var loadIdx = layer.load(2, { time: 5000 });
            $.ajax({
                type: "POST",
                url: "/document/move",
                data: postData,
                dataType: "json",
                success: function (res) {
                    layer.close(loadIdx);
                    if (res.code === 1) {
                        layer.msg(res.message || "文档层级与排序更新成功", { icon: 1, time: 1000 });
                        if (moveType === "inner") {
                            moveNode.pId = targetNode.id;
                        } else {
                            moveNode.pId = targetNode.pId;
                        }
                    } else {
                        layer.msg(res.message || "移动失败，正在重置目录...", { icon: 2, time: 1500 });
                        setTimeout(function () {
                            location.reload();
                        }, 600);
                    }
                },
                error: function () {
                    layer.close(loadIdx);
                    layer.msg("网络请求异常，正在重置目录...", { icon: 2, time: 1500 });
                    setTimeout(function () {
                        location.reload();
                    }, 600);
                }
            });
            return true;
        }

        function onRightClick(event, treeId, treeNode) {
            if (!treeNode) {
                hideContextMenu();
                return;
            }
            event.preventDefault();
            var zTreeMenu = $.fn.zTree.getZTreeObj("dir_tree");
            zTreeMenu.selectNode(treeNode);
            showContextMenu(event, treeNode);
        }

        function ensureContextMenu() {
            if ($("#tree-context-menu").length > 0) return;
            var html = '<div id="tree-context-menu">' +
                '<a class="ctx-item" id="ctx-view"><i class="fa fa-book text-muted"></i> 查看文档</a>' +
                (isEditor ? '<a class="ctx-item" id="ctx-edit"><i class="fa fa-pencil text-primary"></i> 编辑文档</a>' : '') +
                (isEditor ? '<a class="ctx-item" id="ctx-add"><i class="fa fa-plus text-success"></i> 新建子文档</a>' : '') +
                '<div class="ctx-divider"></div>' +
                '<a class="ctx-item" id="ctx-copy-link"><i class="fa fa-link text-info"></i> 复制文档链接</a>' +
                (isDelete ? '<div class="ctx-divider"></div><a class="ctx-item text-danger" id="ctx-delete"><i class="fa fa-trash-o text-danger"></i> 删除文档</a>' : '') +
                '</div>';
            $("body").append(html);

            $(document).on("click scroll", function (e) {
                if (!$(e.target).closest("#tree-context-menu").length) {
                    hideContextMenu();
                }
            });
        }

        function showContextMenu(event, treeNode) {
            ensureContextMenu();
            var menu = $("#tree-context-menu");

            $("#ctx-view").off("click").on("click", function () {
                hideContextMenu();
                switchDocument(treeNode.id, treeNode.name, true);
            });

            if (isEditor) {
                $("#ctx-edit").off("click").on("click", function () {
                    hideContextMenu();
                    window.open("/page/edit?document_id=" + treeNode.id, "_blank");
                });

                if (treeNode.isParent) {
                    $("#ctx-add").show().off("click").on("click", function () {
                        hideContextMenu();
                        var content = "/document/add?space_id=" + treeNode.spaceId + "&parent_id=" + treeNode.id;
                        layer.open({
                            type: 2,
                            skin: Layers.skin,
                            title: '<strong>创建文档</strong>',
                            shadeClose: true,
                            shade: 0.6,
                            maxmin: true,
                            area: ["800px", "345px"],
                            content: content,
                            padding: "10px"
                        });
                    });
                } else {
                    $("#ctx-add").hide();
                }
            }

            $("#ctx-copy-link").off("click").on("click", function () {
                hideContextMenu();
                var link = window.location.protocol + "//" + window.location.host + "/document/index?document_id=" + treeNode.id;
                if (navigator.clipboard && navigator.clipboard.writeText) {
                    navigator.clipboard.writeText(link).then(function () {
                        layer.msg("已复制文档链接到剪贴板", { icon: 1, time: 1500 });
                    });
                } else {
                    var $temp = $("<input>");
                    $("body").append($temp);
                    $temp.val(link).select();
                    document.execCommand("copy");
                    $temp.remove();
                    layer.msg("已复制文档链接到剪贴板", { icon: 1, time: 1500 });
                }
            });

            if (isDelete) {
                $("#ctx-delete").off("click").on("click", function () {
                    hideContextMenu();
                    beforeRemove("dir_tree", treeNode);
                });
            }

            var x = event.clientX;
            var y = event.clientY;
            var winW = $(window).width();
            var winH = $(window).height();
            var menuW = 170;
            var menuH = 180;
            if (x + menuW > winW) x = winW - menuW - 10;
            if (y + menuH > winH) y = winH - menuH - 10;

            menu.css({ top: y + "px", left: x + "px" }).fadeIn(100);
        }

        function hideContextMenu() {
            $("#tree-context-menu").fadeOut(80);
        }

        function addHoverDom(treeId, treeNode) {
            if (isEditor == false) {
                return false;
            }
            if (treeNode.isParent === false || treeNode.isParent === undefined) {
                return false
            }
            var sObj = $("#" + treeNode.tId + "_span");
            var addBtn = $("#addBtn_" + treeNode.tId);
            if (addBtn.length > 0) return;

            var spanHtml = "<span class='button add' id='addBtn_" + treeNode.tId + "' title='新建文档' onfocus='this.blur();'></span>";
            sObj.append(spanHtml);

            // bind add
            var addBtn = $("#addBtn_" + treeNode.tId);
            if (addBtn) addBtn.bind("click", function () {
                var content = "/document/add?space_id=" + treeNode.spaceId + "&parent_id=" + treeNode.id;
                layer.open({
                    type: 2,
                    skin: Layers.skin,
                    title: '<strong>创建文档</strong>',
                    shadeClose: true,
                    shade: 0.6,
                    maxmin: true,
                    area: ["800px", "345px"],
                    content: content,
                    padding: "10px"
                });
                return false;
            });
        }

        function removeHoverDom(treeId, treeNode) {
            $("#addBtn_" + treeNode.tId).unbind().remove();
        }

        $(document).ready(function () {
            $.fn.zTree.init($(element), setting, treeData);
            var zTreeMenu = $.fn.zTree.getZTreeObj("dir_tree");
            var node = zTreeMenu.getNodeByParam("id", defaultId);
            zTreeMenu.selectNode(node, true);
            zTreeMenu.expandNode(node, true, false);
            //initialize fuzzysearch function
            fuzzySearch("dir_tree", '#document_search', null, true);

            // 监听浏览器前进/后退导航，无刷新即时切换
            window.addEventListener("popstate", function (e) {
                var docId = null;
                if (e.state && e.state.document_id) {
                    docId = e.state.document_id;
                } else {
                    var match = location.search.match(/document_id=(\d+)/);
                    if (match && match[1]) {
                        docId = parseInt(match[1]);
                    }
                }
                if (docId) {
                    switchDocument(docId, null, false);
                }
            });
        });
    }
};