/**
 * main
 * Copyright (c) 2018
 */

var sliderLayout = null;

function mainRightHeight() {
    var mainHeight = $(window).height() - $('header').height() - 55;
    $('#mainFrame').height(mainHeight);
}

// webui-popover
function initPopover() {
    $("[data-toggle='web-popover']").webuiPopover({animation: 'pop',autoHide:3000});
    $('[data-toggle="tooltip"]').tooltip();
}

function iniMenu() {
    $('.menu-nav > li > a').click(function() {
        $(".menu-nav > li").each(function () {
            $(this).removeClass('active');
        });
        $(this).parent('li').addClass('active');
    });
}
$(window).resize(function () {
    mainRightHeight();
});

$(window).load(function() {
    mainRightHeight();
    initPopover();
    iniMenu();
});

$(document).ready(function () {
    sliderLayout = $('body').layout({
        west__size:                 230,
        west__spacing_open:		    4,
        west__spacing_closed:		4,
        west__togglerTip_closed:	"打开侧栏",
        west__togglerTip_open:	    "关闭侧栏",
        west__resizerTip:	        "重置大小",
        west__resizerCursor :       "col-resize",
        west__sliderTip:	        "打开侧栏",
        west__slideTrigger_open:	"click",
        west__slideTrigger_close:	"click",
        center__maskContents:		 true
    });
});

function layoutOpen() {
    if ($(window).width() > 768 && typeof sliderLayout !== 'undefined' && sliderLayout && sliderLayout.open) {
        sliderLayout.open("west");
    }
}

function layoutClose() {
    if (typeof sliderLayout !== 'undefined' && sliderLayout && sliderLayout.close) {
        sliderLayout.close("west");
    }
}

function layoutToggle() {
    if ($(window).width() <= 768) {
        window.toggleMobileDrawer();
    } else {
        if (typeof sliderLayout !== 'undefined' && sliderLayout && sliderLayout.toggle) {
            sliderLayout.toggle("west");
        }
    }
}

window.toggleMobileDrawer = function(forceOpen) {
    var $drawer = $("#main-left");
    if ($drawer.length === 0) return false;
    var $backdrop = $(".mobile-drawer-backdrop");
    if ($backdrop.length === 0) {
        $backdrop = $('<div class="mobile-drawer-backdrop" style="display:none;"></div>').appendTo("body");
        $backdrop.on("click", function() {
            window.toggleMobileDrawer(false);
        });
    }

    var shouldOpen = (typeof forceOpen !== 'undefined') ? forceOpen : !$drawer.hasClass("mobile-drawer-open");
    if (shouldOpen) {
        $drawer.addClass("mobile-drawer-open");
        $backdrop.stop(true, true).fadeIn(180);
        $("body").addClass("drawer-open-lock");
    } else {
        $drawer.removeClass("mobile-drawer-open");
        $backdrop.stop(true, true).fadeOut(180);
        $("body").removeClass("drawer-open-lock");
    }
    return shouldOpen;
};

function hiddenScrollY() {
    $("#mainFrame").attr("scrolling", "no");
}

function windowResized() {
    if ($(window).width() <= 768) {
        layoutClose();
    } else {
        window.toggleMobileDrawer(false);
        layoutOpen();
    }
}

$(windowResized);
$(window).resize(windowResized);
