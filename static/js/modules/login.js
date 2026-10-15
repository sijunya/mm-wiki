var Login = {

	errorMessage: "#login-error-alert",

	/**
	 * 登录
	 * @param element
	 */
	ajaxSubmit: function (element) {
		var $form = $(element);
		var usernameEle = $form.find("input[name='username']");
		var passwordEle = $form.find("input[name='password']");
		var submitEle = $form.find("button[name='submit'], [name='submit'], button[type='button'], input[name='submit']").first();
		var name = $.trim(usernameEle.val());
		var password = $.trim(passwordEle.val());

		function showError(msg) {
			var $alert = $("#login-error-alert");
			if ($alert.length > 0) {
				$alert.html('<i class="fa fa-exclamation-circle" style="margin-right: 6px;"></i> ' + msg).slideDown(150);
			}
			layer.msg(msg, { icon: 2, time: 2500, anim: 6 });
		}

		if (!name) {
			showError(usernameEle.attr("placeholder") || "请输入用户名");
			usernameEle.focus();
			return false;
		}
		if (!password) {
			showError(passwordEle.attr("placeholder") || "请输入密码");
			passwordEle.focus();
			return false;
		}

		var origBtnHtml = submitEle.length > 0 ? submitEle.html() : '<i class="fa fa-sign-in"></i> 登 录';
		if (submitEle.length > 0) {
			submitEle.prop('disabled', true).html('<i class="fa fa-spinner fa-spin"></i> 正在登录...');
		}
		$("#login-error-alert").slideUp(100);

		function response(result) {
			if (result.code == 0) {
				if (submitEle.length > 0) {
					submitEle.prop('disabled', false).html(origBtnHtml);
				}
				showError(result.message || "用户名或密码错误！");
				passwordEle.select();
			}
			if (result.code == 1) {
				if (submitEle.length > 0) {
					submitEle.html('<i class="fa fa-check"></i> ' + (result.message || '登录成功'));
				}
				var content = '<i class="fa fa-smile-o"></i> ' + (result.message || '登录成功');
				layer.msg(content, { icon: 1, time: 800 });
				if (result.redirect && result.redirect.url) {
					var sleepTime = (typeof result.redirect.sleep !== "undefined" && result.redirect.sleep !== null) ? result.redirect.sleep : 300;
					setTimeout(function() {
						location.href = result.redirect.url;
					}, sleepTime);
				} else {
					setTimeout(function() {
						location.href = "/";
					}, 300);
				}
			}
		}

		$.ajax({
			url: $form.attr("action") || "/author/login",
			type: $form.attr("method") || "POST",
			data: $form.serialize(),
			dataType: "json",
			success: response,
			error: function(xhr, status, error) {
				if (submitEle.length > 0) {
					submitEle.prop('disabled', false).html(origBtnHtml);
				}
				showError("网络请求异常 (" + (error || status) + ")，请稍后重试");
			}
		});
		return false;
	}
};