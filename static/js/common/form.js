/**
 * Form.js 表单提交类
 * 依赖 jquery.form.js
 */

var Form = {

    /**
     * 提示 div
     */
    failedBox: '#failedBox',

    /**
     * 是否在弹框中
     */
    inPopup: false,

    /**
     * ajax submit
     * @param element
     * @param inPopup
     * @returns {boolean}
     */
    ajaxSubmit: function(element, inPopup) {

        if (inPopup) {
            Form.inPopup = true;
        }

        var $form = $(element);
        var $submitBtn = $form.find("button[type='button'], button[type='submit']").filter(function() {
            var text = $(this).text().trim();
            return text.indexOf('保存') !== -1 || text.indexOf('提交') !== -1 || text.indexOf('确定') !== -1;
        }).first();
        var origBtnHtml = $submitBtn.length > 0 ? $submitBtn.html() : '';
        if ($submitBtn.length > 0) {
            $submitBtn.prop('disabled', true).html('<i class="fa fa-spinner fa-spin"></i> 处理中...');
        }

        /**
         * 成功信息条
         * @param message
         * @param data
         */
        function successBox(message, data) {
            Common.successBox(Form.failedBox, message)
        }

        /**
         * 失败信息条
         * @param message
         * @param data
         */
        function failed(message, data) {
            Common.errorBox(Form.failedBox, message)
        }

        /**
         * request success
         * @param result
         */
        function response(result) {
            //console.log(result)
            if (result.code == 0) {
                if ($submitBtn.length > 0) {
                    $submitBtn.prop('disabled', false).html(origBtnHtml);
                }
                failed(result.message, result.data);
                $("body,html").animate({scrollTop:0}, 150);
            }
            if (result.code == 1) {
                if ($submitBtn.length > 0) {
                    $submitBtn.html('<i class="fa fa-check"></i> ' + (result.message || '操作成功'));
                }
                successBox(result.message, result.data);
                $("body,html").animate({scrollTop:0}, 150);
                if (result.redirect && result.redirect.url) {
                    var sleepTime = (result.redirect.sleep !== undefined && result.redirect.sleep !== null) ? result.redirect.sleep : 300;
                    if (Form.inPopup && sleepTime > 350) {
                        sleepTime = 300;
                    }
                    setTimeout(function() {
                        if (Form.inPopup) {
                            try {
                                if (parent && parent.layer) {
                                    parent.layer.closeAll('iframe');
                                }
                            } catch(e) {}
                            parent.location.href = result.redirect.url;
                        } else {
                            location.href = result.redirect.url;
                        }
                    }, sleepTime);
                }
            }
        }

        var options = {
            dataType: 'json',
            success: response,
            error: function() {
                if ($submitBtn.length > 0) {
                    $submitBtn.prop('disabled', false).html(origBtnHtml);
                }
                failed('网络请求异常，请稍后重试！');
            }
        };

        $form.ajaxSubmit(options);

        return false;
    }
};