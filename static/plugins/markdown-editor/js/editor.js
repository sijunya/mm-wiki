/**
 * MM-Wiki Modern Markdown Editor (Adapted from markdown.com.cn/editor)
 * Tailored for MM-Wiki Architecture
 */

(function (window) {
  'use strict';

  function MMWikiEditor(options) {
    this.options = Object.assign({
      containerId: 'mm-editor-app',
      editorId: 'editor',
      previewId: 'preview',
      resizerId: 'resizer',
      hiddenInputId: 'markdown_content_input',
      documentId: '101',
      initialContent: '',
      uploadUrl: '/image/upload',
      onSave: null
    }, options);

    this.editor = document.getElementById(this.options.editorId);
    this.preview = document.getElementById(this.options.previewId);
    this.resizer = document.getElementById(this.options.resizerId);
    this.hiddenInput = document.getElementById(this.options.hiddenInputId);
    this.container = document.getElementById(this.options.containerId);
    
    this.wordCountEl = document.getElementById('word-count');
    this.cursorPosEl = document.getElementById('cursor-pos');
    this.saveStatusEl = document.getElementById('save-status');
    this.lineNumbersEl = document.getElementById('line-numbers');
    this.syncScrollCheck = document.getElementById('sync-scroll-check');

    this.storageKey = 'mm_wiki_doc_' + this.options.documentId;
    this.themeKey = 'mm_wiki_editor_theme';
    this.isScrolling = false;
    this.syncScroll = true;

    this.init();
  }

  MMWikiEditor.prototype.init = function () {
    this.initTheme();
    this.initMarked();
    this.initContent();
    this.initEvents();
    this.initSyncScroll();
    this.initResizer();
    this.updateStats();
    this.updateLineNumbers();
  };

  // ===== 主题初始化 =====
  MMWikiEditor.prototype.initTheme = function () {
    var savedTheme = localStorage.getItem(this.themeKey);
    if (savedTheme === 'dark' || (!savedTheme && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      document.documentElement.setAttribute('data-theme', 'dark');
      this.isDark = true;
    } else {
      document.documentElement.removeAttribute('data-theme');
      this.isDark = false;
    }
  };

  MMWikiEditor.prototype.toggleTheme = function () {
    this.isDark = !this.isDark;
    if (this.isDark) {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem(this.themeKey, 'dark');
    } else {
      document.documentElement.removeAttribute('data-theme');
      localStorage.setItem(this.themeKey, 'light');
    }
  };

  // ===== Marked 与渲染管线 =====
  MMWikiEditor.prototype.initMarked = function () {
    var self = this;
    if (typeof marked === 'undefined') {
      console.error('Marked library not loaded');
      return;
    }

    var renderer = new marked.Renderer();

    // 自定义代码高亮
    renderer.code = function (code, language) {
      if (language === 'mermaid') {
        return '<div class="mermaid">' + self.escapeHtml(code) + '</div>';
      }
      var validLang = !!(language && hljs.getLanguage(language));
      var highlighted = validLang ? hljs.highlight(code, { language: language }).value : hljs.highlightAuto(code).value;
      var langLabel = language ? '<span class="code-lang-tag">' + language + '</span>' : '';
      return '<div class="code-block-wrapper">' + langLabel + '<pre><code class="hljs ' + (language || '') + '">' + highlighted + '</code></pre></div>';
    };

    // 表格自定义
    renderer.table = function (header, body) {
      return '<div class="table-container"><table class="table table-bordered table-striped"><thead>' + header + '</thead><tbody>' + body + '</tbody></table></div>';
    };

    // 任务列表
    renderer.listitem = function (text, task, checked) {
      if (task) {
        return '<li class="task-list-item"><input type="checkbox" ' + (checked ? 'checked' : '') + ' disabled> ' + text + '</li>';
      }
      return '<li>' + text + '</li>';
    };

    marked.setOptions({
      renderer: renderer,
      gfm: true,
      breaks: true,
      pedantic: false
    });
  };

  MMWikiEditor.prototype.escapeHtml = function (str) {
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
  };

  MMWikiEditor.prototype.render = function () {
    if (!this.editor || !this.preview) return;
    var rawText = this.editor.value;

    // 1. 处理 [TOC]
    var hasTOC = /^\s*\[TOCM?\]\s*$/im.test(rawText);
    var textForMarked = rawText.replace(/^\s*\[TOCM?\]\s*$/gim, '<div id="toc-placeholder"></div>');

    // 2. 解析 Markdown
    var html = marked.parse(textForMarked);

    // 3. 生成目录 TOC
    if (hasTOC) {
      html = this.injectTOC(html, rawText);
    }

    // 4. 安全过滤 DOMPurify
    if (typeof DOMPurify !== 'undefined') {
      html = DOMPurify.sanitize(html, {
        ADD_TAGS: ['input'],
        ADD_ATTR: ['type', 'checked', 'disabled']
      });
    }

    this.preview.innerHTML = html;

    // 5. 渲染数学公式 KaTeX
    if (typeof renderMathInElement !== 'undefined') {
      renderMathInElement(this.preview, {
        delimiters: [
          { left: '$$', right: '$$', display: true },
          { left: '$', right: '$', display: false }
        ],
        throwOnError: false
      });
    }

    // 同步隐藏域给 MM-Wiki 后端
    if (this.hiddenInput) {
      this.hiddenInput.value = rawText;
    }
  };

  MMWikiEditor.prototype.injectTOC = function (html, raw) {
    var headings = [];
    var lines = raw.split('\n');
    lines.forEach(function (line) {
      var match = line.match(/^(#{1,6})\s+(.+)$/);
      if (match) {
        var level = match[1].length;
        var text = match[2].trim();
        var id = text.toLowerCase().replace(/[^\w\u4e00-\u9fa5]+/g, '-');
        headings.push({ level: level, text: text, id: id });
      }
    });

    if (headings.length === 0) {
      return html.replace('<div id="toc-placeholder"></div>', '');
    }

    var tocHtml = '<div class="toc-box"><h4>目录</h4><ul class="toc-list">';
    headings.forEach(function (h) {
      tocHtml += '<li class="toc-item toc-level-' + h.level + '"><a href="#' + h.id + '">' + h.text + '</a></li>';
    });
    tocHtml += '</ul></div>';

    return html.replace('<div id="toc-placeholder"></div>', tocHtml);
  };

  // ===== 内容加载与草稿备份 =====
  MMWikiEditor.prototype.initContent = function () {
    var self = this;
    var initial = this.options.initialContent || '';
    var draft = localStorage.getItem(this.storageKey);

    if (draft && draft !== initial) {
      // 模拟 MM-Wiki 询问是否恢复本地草稿
      if (confirm('【MM-Wiki 提示】检测到本地有未提交的草稿内容，是否恢复？')) {
        this.editor.value = draft;
      } else {
        this.editor.value = initial;
      }
    } else {
      this.editor.value = initial;
    }

    this.render();
  };

  MMWikiEditor.prototype.saveDraft = function () {
    if (!this.editor) return;
    localStorage.setItem(this.storageKey, this.editor.value);
    if (this.saveStatusEl) {
      this.saveStatusEl.textContent = '草稿已暂存 ' + new Date().toLocaleTimeString();
    }
  };

  // ===== 统计与行号 =====
  MMWikiEditor.prototype.updateStats = function () {
    if (!this.editor) return;
    var val = this.editor.value;
    if (this.wordCountEl) {
      this.wordCountEl.textContent = val.length + ' 字';
    }

    if (this.cursorPosEl) {
      var pos = this.editor.selectionStart;
      var lines = val.substring(0, pos).split('\n');
      var line = lines.length;
      var col = lines[lines.length - 1].length + 1;
      this.cursorPosEl.textContent = '行 ' + line + ', 列 ' + col;
    }
  };

  MMWikiEditor.prototype.updateLineNumbers = function () {
    if (!this.lineNumbersEl || !this.editor) return;
    var lineCount = this.editor.value.split('\n').length;
    var html = '';
    for (var i = 1; i <= lineCount; i++) {
      html += '<div>' + i + '</div>';
    }
    this.lineNumbersEl.innerHTML = html;
  };

  // ===== 双向同步滚动 =====
  MMWikiEditor.prototype.initSyncScroll = function () {
    var self = this;
    if (!this.editor || !this.preview) return;

    var editorWrapper = this.editor.parentElement;

    function syncFromEditor() {
      if (!self.syncScroll || self.isScrolling) return;
      self.isScrolling = true;
      var percentage = self.editor.scrollTop / (self.editor.scrollHeight - self.editor.clientHeight || 1);
      self.preview.scrollTop = percentage * (self.preview.scrollHeight - self.preview.clientHeight);
      if (self.lineNumbersEl) {
        self.lineNumbersEl.scrollTop = self.editor.scrollTop;
      }
      setTimeout(function () { self.isScrolling = false; }, 20);
    }

    function syncFromPreview() {
      if (!self.syncScroll || self.isScrolling) return;
      self.isScrolling = true;
      var percentage = self.preview.scrollTop / (self.preview.scrollHeight - self.preview.clientHeight || 1);
      self.editor.scrollTop = percentage * (self.editor.scrollHeight - self.editor.clientHeight);
      if (self.lineNumbersEl) {
        self.lineNumbersEl.scrollTop = self.editor.scrollTop;
      }
      setTimeout(function () { self.isScrolling = false; }, 20);
    }

    this.editor.addEventListener('scroll', syncFromEditor);
    this.preview.addEventListener('scroll', syncFromPreview);

    if (this.syncScrollCheck) {
      this.syncScrollCheck.addEventListener('change', function () {
        self.syncScroll = this.checked;
      });
    }
  };

  // ===== 分栏宽度拖拽调整 =====
  MMWikiEditor.prototype.initResizer = function () {
    var self = this;
    if (!this.resizer || !this.editor || !this.preview) return;

    var isDragging = false;
    var container = this.editor.closest('.editor-container');

    this.resizer.addEventListener('mousedown', function (e) {
      isDragging = true;
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', function (e) {
      if (!isDragging || !container) return;
      var rect = container.getBoundingClientRect();
      var offsetX = e.clientX - rect.left;
      var totalWidth = rect.width;
      var percent = (offsetX / totalWidth) * 100;

      if (percent > 15 && percent < 85) {
        var editorPane = self.editor.closest('.editor-pane');
        var previewPane = self.preview.closest('.preview-pane');
        if (editorPane && previewPane) {
          editorPane.style.flex = '0 0 ' + percent + '%';
          previewPane.style.flex = '0 0 ' + (100 - percent) + '%';
        }
      }
    });

    document.addEventListener('mouseup', function () {
      if (isDragging) {
        isDragging = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }
    });
  };

  // ===== 语法插入工具 =====
  MMWikiEditor.prototype.insertAction = function (action) {
    var ed = this.editor;
    var start = ed.selectionStart;
    var end = ed.selectionEnd;
    var selected = ed.value.substring(start, end);
    var before = ed.value.substring(0, start);
    var after = ed.value.substring(end);
    var replacement = '';
    var newCursor = start;

    switch (action) {
      case 'bold':
        replacement = '**' + (selected || '粗体文本') + '**';
        newCursor = start + 2;
        break;
      case 'italic':
        replacement = '*' + (selected || '斜体文本') + '*';
        newCursor = start + 1;
        break;
      case 'strikethrough':
        replacement = '~~' + (selected || '删除线文本') + '~~';
        newCursor = start + 2;
        break;
      case 'heading-1':
        replacement = '# ' + (selected || '一级标题') + '\n';
        break;
      case 'heading-2':
        replacement = '## ' + (selected || '二级标题') + '\n';
        break;
      case 'heading-3':
        replacement = '### ' + (selected || '三级标题') + '\n';
        break;
      case 'quote':
        replacement = '> ' + (selected || '引用内容') + '\n';
        break;
      case 'code':
        if (selected.includes('\n') || !selected) {
          replacement = '```javascript\n' + (selected || 'console.log("hello");') + '\n```\n';
        } else {
          replacement = '`' + selected + '`';
        }
        break;
      case 'table':
        replacement = '\n| 列1 | 列2 | 列3 |\n| :--- | :---: | ---: |\n| 内容1 | 内容2 | 内容3 |\n';
        break;
      case 'list':
        replacement = '- ' + (selected || '无序列表项') + '\n';
        break;
      case 'ordered-list':
        replacement = '1. ' + (selected || '有序列表项') + '\n';
        break;
      case 'task-list':
        replacement = '- [ ] ' + (selected || '待办事项') + '\n';
        break;
      case 'hr':
        replacement = '\n---\n';
        break;
      case 'toc':
        replacement = '\n[TOC]\n';
        break;
      case 'link':
        replacement = '[' + (selected || '链接描述') + '](https://example.com)';
        break;
      case 'image':
        replacement = '![' + (selected || '图片描述') + '](https://markdown.com.cn/images/qrcode_for_gh_82cf87d482f0_258.jpg)';
        break;
      default:
        return;
    }

    ed.value = before + replacement + after;
    ed.focus();
    ed.setSelectionRange(start + replacement.length, start + replacement.length);
    this.render();
    this.updateStats();
    this.updateLineNumbers();
    this.saveDraft();
  };

  // ===== 剪贴板图片粘贴上传 =====
  MMWikiEditor.prototype.handlePaste = function (e) {
    var self = this;
    var items = (e.clipboardData || window.clipboardData).items;
    if (!items) return;

    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      if (item.kind === 'file' && item.type.match(/^image\//i)) {
        e.preventDefault();
        var file = item.getAsFile();
        self.uploadImageFile(file);
        break;
      }
    }
  };

  MMWikiEditor.prototype.uploadImageFile = function (file) {
    var self = this;
    var formData = new FormData();
    var time = Date.now();
    var newName = time + '-' + file.name;
    formData.append('editormd-image-file', file, newName);

    var uploadUrl = this.options.uploadUrl + '?document_id=' + this.options.documentId + '&guid=' + time;
    
    // 显示状态
    if (this.saveStatusEl) {
      this.saveStatusEl.textContent = '正在上传图片...';
    }

    var xhr = new XMLHttpRequest();
    xhr.open('POST', uploadUrl, true);
    xhr.onload = function () {
      try {
        var res = JSON.parse(xhr.responseText);
        if (res.success === 1) {
          var imgMd = '\n![' + file.name + '](' + res.url + ')\n';
          self.insertTextAtCursor(imgMd);
          if (self.saveStatusEl) self.saveStatusEl.textContent = '图片已上传';
        } else {
          alert('图片上传失败: ' + (res.message || '未知错误'));
        }
      } catch (err) {
        alert('上传解析响应失败: ' + err.message);
      }
    };
    xhr.onerror = function () {
      alert('图片上传网络请求异常');
    };
    xhr.send(formData);
  };

  MMWikiEditor.prototype.insertTextAtCursor = function (text) {
    var ed = this.editor;
    var start = ed.selectionStart;
    var end = ed.selectionEnd;
    ed.value = ed.value.substring(0, start) + text + ed.value.substring(end);
    ed.focus();
    ed.setSelectionRange(start + text.length, start + text.length);
    this.render();
    this.updateStats();
    this.updateLineNumbers();
    this.saveDraft();
  };

  // ===== 事件监听与绑定 =====
  MMWikiEditor.prototype.initEvents = function () {
    var self = this;

    // 输入与状态更新
    this.editor.addEventListener('input', function () {
      self.render();
      self.updateStats();
      self.updateLineNumbers();
      self.saveDraft();
    });

    this.editor.addEventListener('click', function () {
      self.updateStats();
    });

    this.editor.addEventListener('keyup', function () {
      self.updateStats();
    });

    // 粘贴图片
    this.editor.addEventListener('paste', function (e) {
      self.handlePaste(e);
    });

    // 工具栏按钮绑定
    var toolBtns = document.querySelectorAll('.tool-btn, .tool-dropdown-item');
    toolBtns.forEach(function (btn) {
      btn.addEventListener('click', function (e) {
        var action = this.getAttribute('data-action');
        if (!action) return;

        if (action === 'theme') {
          self.toggleTheme();
        } else if (action === 'preview-only') {
          self.togglePreviewOnly();
        } else {
          self.insertAction(action);
        }
      });
    });

    // 快捷键监听 (Ctrl+S / Cmd+S, Ctrl+B, Ctrl+I 等)
    window.addEventListener('keydown', function (e) {
      var isCtrl = e.ctrlKey || e.metaKey;
      if (isCtrl && e.key.toLowerCase() === 's') {
        e.preventDefault();
        if (typeof self.options.onSave === 'function') {
          self.options.onSave();
        }
      } else if (isCtrl && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        self.insertAction('bold');
      } else if (isCtrl && e.key.toLowerCase() === 'i') {
        e.preventDefault();
        self.insertAction('italic');
      }
    });
  };

  MMWikiEditor.prototype.togglePreviewOnly = function () {
    var edPane = this.editor.closest('.editor-pane');
    if (!edPane) return;
    if (edPane.style.display === 'none') {
      edPane.style.display = '';
    } else {
      edPane.style.display = 'none';
    }
  };

  MMWikiEditor.prototype.getValue = function () {
    return this.editor ? this.editor.value : '';
  };

  MMWikiEditor.prototype.setValue = function (val) {
    if (this.editor) {
      this.editor.value = val;
      this.render();
      this.updateStats();
      this.updateLineNumbers();
    }
  };

  window.MMWikiEditor = MMWikiEditor;

})(window);
