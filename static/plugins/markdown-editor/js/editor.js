/**
 * MM-Wiki Modern Markdown Editor
 * Full 100% Feature Parity with markdown.com.cn/editor
 * Integrated seamlessly with MM-Wiki Enterprise Architecture
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

    this.saveStatus = document.getElementById('save-status');
    this.wordCountEl = document.getElementById('word-count');
    this.cursorPos = document.getElementById('cursor-pos');
    this.syncScrollCheck = document.getElementById('sync-scroll-check');
    this.lineNumbersEl = document.getElementById('line-numbers');
    this.lineNumbersToggle = document.getElementById('line-numbers-toggle');
    this.previewToggle = document.getElementById('preview-toggle');
    this.helpModal = document.getElementById('help-modal');
    this.tooltipEl = document.getElementById('tooltip');
    this.imgPreview = document.getElementById('img-preview');
    this.imgPreviewImg = document.getElementById('img-preview-img');

    this.storageKey = 'mm_wiki_doc_' + this.options.documentId;
    this.themeKey = 'mm_wiki_editor_theme';
    this.lineNumbersKey = 'mm_wiki_line_numbers';
    this.syncScrollKey = 'mm_wiki_sync_scroll';

    this.isDark = false;
    this.previewOnly = false;
    this.syncScroll = true;
    this.isScrolling = false;
    this.saveTimer = null;
    this.mermaidPromise = null;

    this.init();
  }

  // ===== 工具栏语法动作配置 =====
  MMWikiEditor.prototype.actions = {
    bold: { before: '**', after: '**', placeholder: '粗体文本' },
    italic: { before: '*', after: '*', placeholder: '斜体文本' },
    underline: { before: '<u>', after: '</u>', placeholder: '下划线文本' },
    strikethrough: { before: '~~', after: '~~', placeholder: '删除线文本' },
    code: { before: '```javascript\n', after: '\n```', placeholder: 'console.log("Hello, MM-Wiki!");' },
    quote: { before: '> ', after: '', placeholder: '引用内容' },
    link: { before: '[', after: '](https://example.com)', placeholder: '链接文本' },
    image: { before: '![', after: '](https://markdown.com.cn/images/qrcode_for_gh_82cf87d482f0_258.jpg)', placeholder: '图片描述' },
    hr: { before: '\n---\n', after: '', placeholder: '' },
    toc: { before: '\n[TOC]\n', after: '', placeholder: '' },
    table: {
      before: '\n| 列1 | 列2 | 列3 |\n| :--- | :---: | ---: |\n| 内容1 | 内容2 | 内容3 |\n',
      after: '', placeholder: ''
    },
    list: { before: '- ', after: '', placeholder: '列表项' },
    'ordered-list': { before: '1. ', after: '', placeholder: '有序列表项' },
    'task-list': { before: '- [ ] ', after: '', placeholder: '待办任务' },
    'heading-1': { before: '# ', after: '', placeholder: '一级标题' },
    'heading-2': { before: '## ', after: '', placeholder: '二级标题' },
    'heading-3': { before: '### ', after: '', placeholder: '三级标题' },
    'heading-4': { before: '#### ', after: '', placeholder: '四级标题' },
    'heading-5': { before: '##### ', after: '', placeholder: '五级标题' },
    'heading-6': { before: '###### ', after: '', placeholder: '六级标题' },
    sup: { before: '<sup>', after: '</sup>', placeholder: '上标' },
    sub: { before: '<sub>', after: '</sub>', placeholder: '下标' }
  };

  MMWikiEditor.prototype.init = function () {
    this.initTheme();
    this.initMarked();
    this.initSyncScroll();
    this.initLineNumbers();
    this.initContent();
    this.bindEvents();
    this.initResizer();
    this.updateStats();
    this.renderPreview();
  };

  // ===== 主题初始化与切换 =====
  MMWikiEditor.prototype.initTheme = function () {
    var savedTheme = localStorage.getItem(this.themeKey);
    if (savedTheme === 'dark' || (!savedTheme && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
      this.isDark = true;
      document.documentElement.setAttribute('data-theme', 'dark');
      this.switchHighlightTheme(true);
    } else {
      this.isDark = false;
      document.documentElement.removeAttribute('data-theme');
      this.switchHighlightTheme(false);
    }
  };

  MMWikiEditor.prototype.toggleTheme = function () {
    this.isDark = !this.isDark;
    if (this.isDark) {
      document.documentElement.setAttribute('data-theme', 'dark');
      localStorage.setItem(this.themeKey, 'dark');
      this.switchHighlightTheme(true);
    } else {
      document.documentElement.removeAttribute('data-theme');
      localStorage.setItem(this.themeKey, 'light');
      this.switchHighlightTheme(false);
    }
    this.renderPreview();
  };

  MMWikiEditor.prototype.switchHighlightTheme = function (isDark) {
    var hljsTheme = document.getElementById('hljs-theme');
    if (hljsTheme) {
      hljsTheme.href = isDark
        ? '/static/css/highlight-github-dark.min.css'
        : '/static/css/highlight-github.min.css';
    }
  };

  // ===== Mermaid 语法转换与静态工具函数 =====
  MMWikiEditor.convertSeqToMermaid = function (code) {
    if (/^\s*sequenceDiagram\b/mi.test(code)) return code;
    var lines = code.trim().split('\n');
    var result = ['sequenceDiagram'];
    for (var i = 0; i < lines.length; i++) {
      var line = lines[i].trim();
      if (!line || line.startsWith('//') || line.startsWith('#')) continue;
      var mermaidLine = line
        .replace(/^(\w+)=>(\w+):/, '$1->>$2:')
        .replace(/^(\w+)-->(\w+):/, '$1-->>$2:')
        .replace(/^(\w+)->(\w+)(?![->]):/, '$1->>$2:');
      result.push('    ' + mermaidLine);
    }
    return result.join('\n');
  };
  MMWikiEditor.prototype.convertSeqToMermaid = MMWikiEditor.convertSeqToMermaid;

  MMWikiEditor._mermaidPromise = null;
  MMWikiEditor.getMermaid = function () {
    if (MMWikiEditor._mermaidPromise) return MMWikiEditor._mermaidPromise;
    if (typeof mermaid !== 'undefined') {
      MMWikiEditor._mermaidPromise = Promise.resolve(mermaid);
      return MMWikiEditor._mermaidPromise;
    }
    MMWikiEditor._mermaidPromise = new Promise(function (resolve, reject) {
      var script = document.createElement('script');
      script.src = '/static/plugins/markdown-editor/js/mermaid.min.js';
      script.onload = function () { resolve(window.mermaid); };
      script.onerror = function () { reject(new Error('Mermaid 加载失败')); };
      document.head.appendChild(script);
    });
    return MMWikiEditor._mermaidPromise;
  };
  MMWikiEditor.prototype.getMermaid = MMWikiEditor.getMermaid;

  MMWikiEditor.escapeHtml = function (str) {
    if (str == null) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  };
  MMWikiEditor.prototype.escapeHtml = MMWikiEditor.escapeHtml;

  MMWikiEditor.stripFrontMatter = function (text) {
    return (text || '').replace(/^\s*---\s*\n[\s\S]*?\n---\s*(?:\n|$)/, '');
  };
  MMWikiEditor.prototype.stripFrontMatter = MMWikiEditor.stripFrontMatter;

  // ===== 脚注与目录预处理 =====
  MMWikiEditor.preprocessFootnotes = function (raw) {
    var defs = {};
    var defRegex = /^\[\^(\w+)\]:\s*(.+)$/gm;
    var m;
    while ((m = defRegex.exec(raw)) !== null) {
      defs[m[1]] = m[2];
    }
    var processed = raw.replace(/^\[\^\w+\]:\s*.+$/gm, '');
    var index = 1;
    var indexMap = {};
    processed = processed.replace(/\[\^(\w+)\]/g, function (match, key) {
      if (defs[key]) {
        if (!indexMap[key]) indexMap[key] = index++;
        return '<!--FNREF:' + key + '-->';
      }
      return match;
    });
    return { processed: processed, defs: defs, indexMap: indexMap };
  };
  MMWikiEditor.prototype.preprocessFootnotes = MMWikiEditor.preprocessFootnotes;

  MMWikiEditor.renderFootnotes = function (html, defs, indexMap) {
    html = html.replace(/<!--FNREF:(\w+)-->/g, function (match, key) {
      if (indexMap[key]) {
        return '<sup class="footnote-ref"><a id="fnref-' + key + '" href="#fn-' + key + '" title="查看脚注">[' + indexMap[key] + ']</a></sup>';
      }
      return match;
    });

    var footnoteHtml = '';
    for (var key in indexMap) {
      footnoteHtml += '<li id="fn-' + key + '"><p>' + marked.parseInline(defs[key]) + ' <a href="#fnref-' + key + '" title="返回引用">↩</a></p></li>';
    }
    if (footnoteHtml) {
      html += '<div class="footnotes"><ol>' + footnoteHtml + '</ol></div>';
    }
    return html;
  };
  MMWikiEditor.prototype.renderFootnotes = MMWikiEditor.renderFootnotes;

  MMWikiEditor.processTOC = function (html, raw) {
    if (!/\[TOC\]|\[toc\]/i.test(raw)) return html;

    var lines = raw.split('\n');
    var headings = [];
    var inCodeBlock = false;
    var codeFence = null;

    for (var i = 0; i < lines.length; i++) {
      var trimmed = lines[i].trim();
      var fenceMatch = trimmed.match(/^(```+|~~~+)/);
      if (fenceMatch) {
        var fence = fenceMatch[1];
        if (!inCodeBlock) {
          inCodeBlock = true;
          codeFence = fence;
        } else if (fence.startsWith(codeFence) || codeFence.startsWith(fence)) {
          inCodeBlock = false;
          codeFence = null;
        }
        continue;
      }
      if (inCodeBlock) continue;

      var match = trimmed.match(/^(#{1,6})\s+(.+)$/);
      if (match) {
        var level = match[1].length;
        var text = match[2].replace(/\s*\{[^}]*\}\s*$/, '').trim();
        var id = text.toLowerCase().replace(/[^\w\u4e00-\u9fa5\s-]/g, '').replace(/\s+/g, '-');
        headings.push({ level: level, text: text, id: id });
      }
    }

    if (headings.length === 0) return html;

    var tocHtml = '<div class="toc"><div class="callout-title">📑 目录导航</div><ul>';
    headings.forEach(function (h) {
      tocHtml += '<li style="margin-left:' + ((h.level - 1) * 16) + 'px"><a href="#' + h.id + '">' + MMWikiEditor.escapeHtml(h.text) + '</a></li>';
    });
    tocHtml += '</ul></div>';

    return html.replace(/<p>\[TOC\]<\/p>/gi, tocHtml).replace(/<p>\[toc\]<\/p>/gi, tocHtml);
  };
  MMWikiEditor.prototype.processTOC = MMWikiEditor.processTOC;

  // ===== 全局 Marked 初始化 =====
  MMWikiEditor._markedInitialized = false;
  MMWikiEditor.ensureMarked = function () {
    if (MMWikiEditor._markedInitialized) return;
    if (typeof marked === 'undefined') {
      console.error('Marked 库未加载');
      return;
    }

    var renderer = new marked.Renderer();
    var mermaidLangs = ['mermaid','graph','flowchart','sequencediagram','classdiagram','state','er','journey','gitgraph','pie','requirementdiagram','c4context','mindmap','timeline','sankey','xychart','block','packet','kanban'];

    renderer.code = function (code, language) {
      var lang = (language || '').toLowerCase();
      if (mermaidLangs.indexOf(lang) !== -1 || lang === 'seq' || lang === 'gantt') {
        var mermaidCode = code;
        if (lang === 'gantt') mermaidCode = 'gantt\n' + code;
        else if (lang === 'seq') mermaidCode = MMWikiEditor.convertSeqToMermaid(code);
        return '<pre><code class="hljs language-mermaid">' + MMWikiEditor.escapeHtml(mermaidCode) + '</code><button type="button" class="hljs-copy-btn">复制</button></pre>';
      }

      var validLang = language && typeof hljs !== 'undefined' && hljs.getLanguage(language) ? language : 'plaintext';
      var highlighted = '';
      if (typeof hljs !== 'undefined') {
        highlighted = validLang !== 'plaintext' ? hljs.highlight(code, { language: validLang }).value : hljs.highlightAuto(code).value;
      } else {
        highlighted = MMWikiEditor.escapeHtml(code);
      }
      return '<pre><code class="hljs language-' + validLang + '">' + highlighted + '</code><button type="button" class="hljs-copy-btn">复制</button></pre>';
    };

    renderer.codespan = function (code) {
      return '<code>' + MMWikiEditor.escapeHtml(code) + '</code>';
    };

    renderer.table = function (header, body) {
      return '<div class="table-container"><table class="table table-bordered table-striped"><thead>' + header + '</thead><tbody>' + body + '</tbody></table></div>';
    };

    renderer.heading = function (text, level) {
      var id = text.toLowerCase().replace(/[^\w\u4e00-\u9fa5\s-]/g, '').replace(/\s+/g, '-');
      return '<h' + level + ' id="' + id + '">' + text + '</h' + level + '>';
    };

    renderer.blockquote = function (quote) {
      var calloutMatch = quote.match(/^\s*<p>\[!([A-Z]+)\](?:\s+([^<]*?))?(?:<br\s*\/?>\s*|\s*)/i);
      if (calloutMatch) {
        var type = calloutMatch[1].toLowerCase();
        var inlineTitle = calloutMatch[2];
        var title = inlineTitle || type.toUpperCase();
        var content = quote.replace(/^\s*<p>\[!([A-Z]+)\](?:\s+[^<]*?)?(?:<br\s*\/?>\s*|\s*)/i, '<p>');
        var icons = {
          info: 'ℹ️', warning: '⚠️', danger: '🔴', success: '✅',
          note: '📝', tip: '💡', important: '❗'
        };
        var icon = icons[type] || '📌';
        return '<div class="callout callout-' + type + '"><div class="callout-title">' + icon + ' ' + MMWikiEditor.escapeHtml(title) + '</div>' + content + '</div>';
      }
      return '<blockquote>' + quote + '</blockquote>';
    };

    marked.use({
      extensions: [
        {
          name: 'mathBlock',
          level: 'block',
          start: function (src) { return src.indexOf('$$'); },
          tokenizer: function (src) {
            var match = /^\$\$[\r\n]+([\s\S]*?)[\r\n]+\$\$(?:\n+|$)/.exec(src);
            if (!match) {
              match = /^\$\$([^\n]+?)\$\$(?:\n+|$)/.exec(src);
            }
            if (match) {
              return {
                type: 'mathBlock',
                raw: match[0],
                text: match[1].trim()
              };
            }
          },
          renderer: function (token) {
            if (typeof katex !== 'undefined') {
              try {
                return '<div class="katex-block">' + katex.renderToString(token.text, { displayMode: true, throwOnError: false }) + '</div>\n';
              } catch (e) {}
            }
            return '<div class="katex-block">$$' + MMWikiEditor.escapeHtml(token.text) + '$$</div>\n';
          }
        },
        {
          name: 'mathInline',
          level: 'inline',
          start: function (src) { return src.indexOf('$'); },
          tokenizer: function (src) {
            var matchBlock = /^\$\$([^\$]+?)\$\$/.exec(src);
            if (matchBlock && matchBlock[1].trim().length > 0) {
              return { type: 'mathBlockInline', raw: matchBlock[0], text: matchBlock[1].trim() };
            }
            var match = /^\$([^\$\n\r]+?)\$/.exec(src);
            if (match && match[1].trim().length > 0) {
              return { type: 'mathInline', raw: match[0], text: match[1].trim() };
            }
          },
          renderer: function (token) {
            var isBlock = token.type === 'mathBlockInline';
            if (typeof katex !== 'undefined') {
              try {
                return katex.renderToString(token.text, { displayMode: isBlock, throwOnError: false });
              } catch (e) {}
            }
            return (isBlock ? '$$' : '$') + MMWikiEditor.escapeHtml(token.text) + (isBlock ? '$$' : '$');
          }
        },
        {
          name: 'highlight',
          level: 'inline',
          start: function (src) { return src.indexOf('=='); },
          tokenizer: function (src) {
            var match = /^==(.+?)==/.exec(src);
            if (match) {
              return { type: 'highlight', raw: match[0], text: match[1] };
            }
          },
          renderer: function (token) {
            return '<mark>' + MMWikiEditor.escapeHtml(token.text) + '</mark>';
          }
        }
      ]
    });

    marked.use({
      renderer: renderer,
      gfm: true,
      breaks: true
    });

    MMWikiEditor._markedInitialized = true;
  };
  MMWikiEditor.prototype.initMarked = function () {
    MMWikiEditor.ensureMarked();
  };

  // ===== 静态独立 Markdown 渲染管线 (可供浏览页/展示页直接调用) =====
  MMWikiEditor.render = function (raw, targetEl, options) {
    if (!targetEl) return;
    options = options || {};
    MMWikiEditor.ensureMarked();

    var cleanRaw = MMWikiEditor.stripFrontMatter(raw || '');
    var fnData = MMWikiEditor.preprocessFootnotes(cleanRaw);
    var html = marked.parse(fnData.processed);
    html = MMWikiEditor.processTOC(html, cleanRaw);
    html = MMWikiEditor.renderFootnotes(html, fnData.defs, fnData.indexMap);

    if (typeof DOMPurify !== 'undefined' && typeof DOMPurify.sanitize === 'function') {
      html = DOMPurify.sanitize(html, {
        ADD_TAGS: ['mjx-container', 'math', 'mrow', 'mi', 'mo', 'mn', 'msup', 'msub', 'mfrac', 'msqrt', 'mtable', 'mtr', 'mtd', 'mtext', 'mstyle', 'annotation', 'semantics', 'input'],
        ADD_ATTR: ['xmlns', 'display', 'mathvariant', 'accent', 'fence', 'stretchy', 'lspace', 'rspace', 'displaystyle', 'scriptlevel', 'type', 'checked', 'disabled']
      });
    }

    targetEl.innerHTML = html;
    if (!targetEl.classList.contains('preview-content')) {
      targetEl.classList.add('preview-content');
    }

    // 任务列表样式
    targetEl.querySelectorAll('li').forEach(function (li) {
      if (li.querySelector(':scope > input[type="checkbox"]')) {
        li.classList.add('task-list-item');
      }
    });

    // 代码块一键复制按钮
    targetEl.querySelectorAll('.hljs-copy-btn').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var codeEl = this.parentElement.querySelector('code');
        if (codeEl && navigator.clipboard) {
          navigator.clipboard.writeText(codeEl.textContent).then(function () {
            btn.textContent = '已复制';
            setTimeout(function () { btn.textContent = '复制'; }, 1500);
          });
        }
      });
    });

    // KaTeX 数学公式渲染
    if (typeof renderMathInElement !== 'undefined') {
      renderMathInElement(targetEl, {
        delimiters: [
          { left: '$$', right: '$$', display: true },
          { left: '$', right: '$', display: false }
        ],
        throwOnError: false
      });
    }

    // Mermaid 图表渲染
    var mermaidBlocks = targetEl.querySelectorAll('.language-mermaid');
    if (mermaidBlocks.length > 0) {
      mermaidBlocks.forEach(function (el) {
        var code = el.textContent;
        var container = document.createElement('div');
        container.className = 'mermaid';
        container.textContent = code;
        el.closest('pre').replaceWith(container);
      });

      MMWikiEditor.getMermaid().then(function (m) {
        m.initialize({ startOnLoad: false, theme: options.isDark ? 'dark' : 'default' });
        return m.run({ querySelector: '.mermaid' });
      }).then(function () {
        targetEl.querySelectorAll('.mermaid svg').forEach(function (svg) {
          svg.style.cursor = 'zoom-in';
          svg.addEventListener('click', function (e) { MMWikiEditor.showImageOverlay(e); });
        });
      }).catch(function (err) {
        console.warn('Mermaid 渲染异常:', err);
      });
    }

    // 图片点击放大全屏查看
    targetEl.querySelectorAll('img').forEach(function (img) {
      img.style.cursor = 'zoom-in';
      img.addEventListener('click', function (e) { MMWikiEditor.showImageOverlay(e); });
    });
  };

  // ===== 大图全屏查看 =====
  MMWikiEditor.showImageOverlay = function (e) {
    var overlay = document.getElementById('img-preview');
    var overlayImg = document.getElementById('img-preview-img');
    if (!overlay || !overlayImg) return;
    var el = e.currentTarget;
    if (el.tagName.toLowerCase() === 'svg') {
      var serializer = new XMLSerializer();
      var svgStr = serializer.serializeToString(el);
      overlayImg.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svgStr);
      overlayImg.alt = '图表预览';
    } else {
      overlayImg.src = el.src;
      overlayImg.alt = el.alt || '图片预览';
    }
    overlay.classList.add('active');
  };
  MMWikiEditor.prototype.showImagePreview = MMWikiEditor.showImageOverlay;

  // ===== 编辑器实例渲染管线 =====
  MMWikiEditor.prototype.renderPreview = function () {
    if (!this.editor || !this.preview) return;
    var raw = this.editor.value;
    MMWikiEditor.render(raw, this.preview, { isDark: this.isDark });

    if (this.hiddenInput) {
      this.hiddenInput.value = raw;
    }

    this.updateStats();
    this.scheduleSave();
  };

  // ===== 本地草稿与内容加载 =====
  MMWikiEditor.prototype.initContent = function () {
    var initial = this.options.initialContent || '';
    var draft = localStorage.getItem(this.storageKey);

    if (draft && draft !== initial) {
      if (confirm('【MM-Wiki 提示】检测到本地存在上次未提交的文档草稿，是否恢复？')) {
        this.editor.value = draft;
      } else {
        this.editor.value = initial;
      }
    } else {
      this.editor.value = initial;
    }

    if (this.hiddenInput) {
      this.hiddenInput.value = this.editor.value;
    }
  };

  MMWikiEditor.prototype.scheduleSave = function () {
    var self = this;
    if (this.saveStatus) {
      this.saveStatus.textContent = '草稿暂存中...';
      this.saveStatus.classList.add('saving');
    }
    clearTimeout(this.saveTimer);
    this.saveTimer = setTimeout(function () {
      localStorage.setItem(self.storageKey, self.editor.value);
      if (self.saveStatus) {
        self.saveStatus.textContent = '草稿已暂存 ' + new Date().toLocaleTimeString();
        self.saveStatus.classList.remove('saving');
      }
    }, 800);
  };

  // ===== 行号计算与同步 =====
  MMWikiEditor.prototype.initLineNumbers = function () {
    var self = this;
    if (!this.lineNumbersEl) return;
    var saved = localStorage.getItem(this.lineNumbersKey);
    if (saved !== 'off') {
      this.lineNumbersEl.classList.add('show');
      if (this.lineNumbersToggle) this.lineNumbersToggle.classList.add('active');
    }
    this.editor.addEventListener('input', function () { self.updateLineNumbers(); });
    this.editor.addEventListener('scroll', function () {
      self.lineNumbersEl.scrollTop = self.editor.scrollTop;
    });
  };

  MMWikiEditor.prototype.updateLineNumbers = function () {
    if (!this.lineNumbersEl || !this.lineNumbersEl.classList.contains('show')) return;
    var lines = this.editor.value.split('\n').length;
    var html = '';
    for (var i = 1; i <= lines; i++) {
      html += '<div>' + i + '</div>';
    }
    this.lineNumbersEl.innerHTML = html;
  };

  MMWikiEditor.prototype.toggleLineNumbers = function () {
    if (!this.lineNumbersEl) return;
    var show = !this.lineNumbersEl.classList.contains('show');
    this.lineNumbersEl.classList.toggle('show', show);
    if (this.lineNumbersToggle) this.lineNumbersToggle.classList.toggle('active', show);
    localStorage.setItem(this.lineNumbersKey, show ? 'on' : 'off');
    if (show) {
      this.updateLineNumbers();
      this.lineNumbersEl.scrollTop = this.editor.scrollTop;
    }
  };

  // ===== 统计与光标 =====
  MMWikiEditor.prototype.updateStats = function () {
    if (!this.editor) return;
    var text = this.editor.value;
    var count = text.replace(/\s/g, '').length;
    if (this.wordCountEl) {
      this.wordCountEl.textContent = count + ' 字';
    }

    if (this.cursorPos) {
      var pos = this.editor.selectionStart;
      var sub = text.substring(0, pos);
      var lines = sub.split('\n');
      var line = lines.length;
      var col = lines[lines.length - 1].length + 1;
      this.cursorPos.textContent = '行 ' + line + ', 列 ' + col;
    }
  };

  // ===== 双向同步滚动 (基于虚拟 Heading 插值算法) =====
  MMWikiEditor.prototype.initSyncScroll = function () {
    var self = this;
    if (!this.editor || !this.preview) return;

    var saved = localStorage.getItem(this.syncScrollKey);
    if (saved === 'off') {
      this.syncScroll = false;
    }
    if (this.syncScrollCheck) {
      this.syncScrollCheck.checked = this.syncScroll;
      this.syncScrollCheck.addEventListener('change', function () {
        self.syncScroll = this.checked;
        localStorage.setItem(self.syncScrollKey, self.syncScroll ? 'on' : 'off');
      });
    }

    var activePane = 'editor';
    this.editor.addEventListener('mouseenter', function () { activePane = 'editor'; });
    this.editor.addEventListener('wheel', function () { activePane = 'editor'; }, { passive: true });
    this.preview.addEventListener('mouseenter', function () { activePane = 'preview'; });
    this.preview.addEventListener('wheel', function () { activePane = 'preview'; }, { passive: true });

    var isSyncing = false;

    function syncScrollFrom(source, target) {
      if (isSyncing) return;
      var srcMax = source.scrollHeight - source.clientHeight;
      var tgtMax = target.scrollHeight - target.clientHeight;
      if (srcMax <= 0 || tgtMax <= 0) return;

      if (source === self.editor) {
        // 编辑器 -> 预览：基于 heading 锚点线性插值
        var headings = target.querySelectorAll('h1, h2, h3, h4, h5, h6');
        if (headings.length === 0) {
          var targetScroll = (source.scrollTop / srcMax) * tgtMax;
          if (Math.abs(target.scrollTop - targetScroll) > 0.5) {
            isSyncing = true;
            target.scrollTop = targetScroll;
            requestAnimationFrame(function () { isSyncing = false; });
          }
          return;
        }

        var ratio = source.scrollTop / srcMax;
        var virtualHeadings = [{ offsetTop: 0 }];
        headings.forEach(function (h) { virtualHeadings.push({ offsetTop: h.offsetTop }); });
        virtualHeadings.push({ offsetTop: tgtMax });

        var segments = virtualHeadings.length - 1;
        var floatIndex = ratio * segments;
        var index = Math.floor(floatIndex);
        var frac = floatIndex - index;

        var curr = virtualHeadings[index];
        var next = virtualHeadings[Math.min(index + 1, virtualHeadings.length - 1)];
        var targetScroll = curr.offsetTop + frac * (next.offsetTop - curr.offsetTop);

        if (Math.abs(target.scrollTop - targetScroll) > 0.5) {
          isSyncing = true;
          target.scrollTop = Math.max(0, Math.min(targetScroll, tgtMax));
          requestAnimationFrame(function () { isSyncing = false; });
        }
      } else {
        var ratio = source.scrollTop / srcMax;
        var targetScroll = ratio * tgtMax;
        if (Math.abs(target.scrollTop - targetScroll) > 0.5) {
          isSyncing = true;
          target.scrollTop = targetScroll;
          requestAnimationFrame(function () { isSyncing = false; });
        }
      }
    }

    this.editor.addEventListener('scroll', function () {
      if (!self.syncScroll || activePane !== 'editor') return;
      syncScrollFrom(self.editor, self.preview);
    }, { passive: true });

    this.preview.addEventListener('scroll', function () {
      if (!self.syncScroll || activePane !== 'preview') return;
      syncScrollFrom(self.preview, self.editor);
    }, { passive: true });
  };

  // ===== 分栏调整宽度 =====
  MMWikiEditor.prototype.initResizer = function () {
    var self = this;
    if (!this.resizer || !this.editor || !this.preview) return;

    var isResizing = false;
    var startX = 0;
    var startWidth = 0;
    var editorPane = this.editor.closest('.editor-pane');
    var container = this.editor.closest('.editor-container');

    this.resizer.addEventListener('mousedown', function (e) {
      isResizing = true;
      startX = e.clientX;
      startWidth = editorPane.offsetWidth;
      self.resizer.classList.add('dragging');
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', function (e) {
      if (!isResizing || !container) return;
      var dx = e.clientX - startX;
      var containerWidth = container.offsetWidth;
      var newWidth = Math.max(150, Math.min(startWidth + dx, containerWidth - 150));
      var percent = (newWidth / containerWidth) * 100;
      editorPane.style.flex = '0 0 ' + percent + '%';
    });

    document.addEventListener('mouseup', function () {
      if (isResizing) {
        isResizing = false;
        self.resizer.classList.remove('dragging');
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
      }
    });
  };

  // ===== 语法插入 =====
  MMWikiEditor.prototype.insertMarkdown = function (action) {
    var cfg = this.actions[action];
    if (!cfg) return;

    var start = this.editor.selectionStart;
    var end = this.editor.selectionEnd;
    var selected = this.editor.value.substring(start, end);
    var insertText = selected || cfg.placeholder;
    var replacement = cfg.before + insertText + cfg.after;

    this.editor.setRangeText(replacement, start, end, 'end');
    this.editor.focus();

    if (!selected && cfg.placeholder) {
      var newPos = start + cfg.before.length;
      this.editor.setSelectionRange(newPos, newPos + cfg.placeholder.length);
    }

    this.renderPreview();
    this.updateLineNumbers();
  };

  // ===== 导入与导出 =====
  MMWikiEditor.prototype.downloadFile = function (content, filename, type) {
    var blob = new Blob([content], { type: type });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  MMWikiEditor.prototype.exportMarkdown = function () {
    var title = this.editor.value.match(/^#\s+(.+)$/m);
    var filename = (title ? title[1].trim() : 'document') + '.md';
    this.downloadFile(this.editor.value, filename, 'text/markdown');
  };

  MMWikiEditor.prototype.exportHTML = async function () {
    var self = this;
    var title = this.editor.value.match(/^#\s+(.+)$/m);
    var docTitle = title ? title[1].trim() : 'Document';

    var clone = this.preview.cloneNode(true);
    clone.querySelectorAll('.hljs-copy-btn').forEach(function (btn) { btn.remove(); });
    clone.querySelectorAll('[id]').forEach(function (el) {
      if (!el.closest('svg')) el.removeAttribute('id');
    });

    var htmlContent = '<!DOCTYPE html>\n<html lang="zh-CN">\n<head>\n<meta charset="UTF-8">\n<title>' + self.escapeHtml(docTitle) + '</title>\n' +
      '<link rel="stylesheet" href="https://fastly.jsdelivr.net/npm/@highlightjs/cdn-assets@11.9.0/styles/github.min.css">\n' +
      '<link rel="stylesheet" href="https://fastly.jsdelivr.net/npm/katex@0.16.9/dist/katex.min.css">\n' +
      '<style>\n' +
      'body { font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif; max-width: 860px; margin: 40px auto; padding: 0 24px; line-height: 1.8; color: #333; }\n' +
      'table { width: 100%; border-collapse: collapse; margin: 16px 0; }\n' +
      'th, td { border: 1px solid #ddd; padding: 10px 14px; text-align: left; }\n' +
      'pre { background: #f6f8fa; padding: 16px; border-radius: 8px; overflow-x: auto; }\n' +
      'blockquote { border-left: 4px solid #2563eb; background: #f8f9fa; padding: 12px 16px; margin: 16px 0; }\n' +
      'mark { background: #fef08a; padding: 1px 4px; border-radius: 3px; }\n' +
      '</style>\n</head>\n<body>\n' + clone.innerHTML + '\n</body>\n</html>';

    this.downloadFile(htmlContent, docTitle + '.html', 'text/html');
  };

  MMWikiEditor.prototype.importMarkdown = function (file) {
    var self = this;
    var reader = new FileReader();
    reader.onload = function (e) {
      self.editor.value = e.target.result;
      self.renderPreview();
      self.updateLineNumbers();
    };
    reader.readAsText(file);
  };

  // ===== 注释切换与全屏预览 =====
  MMWikiEditor.prototype.toggleComment = function () {
    var start = this.editor.selectionStart;
    var end = this.editor.selectionEnd;
    var selected = this.editor.value.substring(start, end);

    if (selected) {
      if (selected.startsWith('<!-- ') && selected.endsWith(' -->')) {
        this.editor.setRangeText(selected.slice(5, -4), start, end, 'end');
      } else {
        this.editor.setRangeText('<!-- ' + selected + ' -->', start, end, 'end');
      }
    } else {
      this.editor.setRangeText('<!--  -->', start, end, 'end');
      this.editor.setSelectionRange(start + 5, start + 5);
    }
    this.renderPreview();
  };

  MMWikiEditor.prototype.togglePreviewOnly = function () {
    this.previewOnly = !this.previewOnly;
    document.body.classList.toggle('preview-only', this.previewOnly);
    if (this.previewToggle) {
      this.previewToggle.style.color = this.previewOnly ? 'var(--accent)' : '';
    }
  };

  MMWikiEditor.prototype.clearEditor = function () {
    if (confirm('确定要清空编辑器全部内容吗？')) {
      this.editor.value = '';
      this.renderPreview();
      this.updateLineNumbers();
    }
  };

  MMWikiEditor.prototype.showHelp = function () {
    if (this.helpModal) this.helpModal.classList.add('active');
  };

  MMWikiEditor.prototype.hideHelp = function () {
    if (this.helpModal) this.helpModal.classList.remove('active');
  };

  // ===== 图片上传处理 (粘贴与拖拽) =====
  MMWikiEditor.prototype.handlePaste = function (e) {
    var self = this;
    var items = (e.clipboardData || window.clipboardData).items;
    if (!items) return;

    for (var i = 0; i < items.length; i++) {
      var item = items[i];
      if (item.kind === 'file' && item.type.match(/^image\//i)) {
        e.preventDefault();
        var file = item.getAsFile();
        this.uploadImageFile(file);
        break;
      }
    }
  };

  MMWikiEditor.prototype.handleDrop = function (e) {
    var self = this;
    e.preventDefault();
    var files = e.dataTransfer ? e.dataTransfer.files : null;
    if (!files) return;

    for (var i = 0; i < files.length; i++) {
      var file = files[i];
      if (file.type.match(/^image\//i)) {
        self.uploadImageFile(file);
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
    if (this.saveStatus) this.saveStatus.textContent = '正在上传图片...';

    var xhr = new XMLHttpRequest();
    xhr.open('POST', uploadUrl, true);
    xhr.onload = function () {
      try {
        var res = JSON.parse(xhr.responseText);
        if (res.success === 1) {
          var imgMd = '\n![' + file.name + '](' + res.url + ')\n';
          self.insertTextAtCursor(imgMd);
          if (self.saveStatus) self.saveStatus.textContent = '图片已上传';
        } else {
          alert('图片上传失败: ' + (res.message || '服务端错误'));
        }
      } catch (err) {
        alert('解析上传响应失败: ' + err.message);
      }
    };
    xhr.onerror = function () { alert('图片上传网络请求异常'); };
    xhr.send(formData);
  };

  MMWikiEditor.prototype.insertTextAtCursor = function (text) {
    var ed = this.editor;
    var start = ed.selectionStart;
    var end = ed.selectionEnd;
    ed.setRangeText(text, start, end, 'end');
    this.renderPreview();
    this.updateLineNumbers();
  };

  // ===== 浮动 Tooltip =====
  MMWikiEditor.prototype.initTooltip = function () {
    var self = this;
    if (!this.tooltipEl) return;
    var targets = document.querySelectorAll('[title]');
    targets.forEach(function (el) {
      if (el.dataset.tooltipBound) return;
      var title = el.getAttribute('title');
      if (!title) return;
      el.dataset.tooltip = title;
      el.removeAttribute('title');
      el.dataset.tooltipBound = '1';

      el.addEventListener('mouseenter', function () {
        self.tooltipEl.textContent = el.dataset.tooltip;
        self.tooltipEl.classList.add('show');
        var rect = el.getBoundingClientRect();
        var tipRect = self.tooltipEl.getBoundingClientRect();
        var left = rect.left + rect.width / 2 - tipRect.width / 2;
        var top = rect.bottom + 6;
        if (left < 4) left = 4;
        if (left + tipRect.width > window.innerWidth - 4) {
          left = window.innerWidth - tipRect.width - 4;
        }
        self.tooltipEl.style.left = left + 'px';
        self.tooltipEl.style.top = top + 'px';
      });

      el.addEventListener('mouseleave', function () {
        self.tooltipEl.classList.remove('show');
      });
    });
  };

  // ===== 事件监听绑定 =====
  MMWikiEditor.prototype.bindEvents = function () {
    var self = this;

    this.editor.addEventListener('input', function () {
      self.renderPreview();
      self.updateStats();
    });

    this.editor.addEventListener('keyup', function () { self.updateStats(); });
    this.editor.addEventListener('click', function () { self.updateStats(); });
    this.editor.addEventListener('paste', function (e) { self.handlePaste(e); });
    this.editor.addEventListener('drop', function (e) { self.handleDrop(e); });
    this.editor.addEventListener('dragover', function (e) { e.preventDefault(); });

    // 键盘快捷键监听
    this.editor.addEventListener('keydown', function (e) {
      // Tab 缩进 (2空格)
      if (e.key === 'Tab') {
        e.preventDefault();
        var start = self.editor.selectionStart;
        var end = self.editor.selectionEnd;
        if (e.shiftKey) {
          var before = self.editor.value.substring(0, start);
          var lineStart = before.lastIndexOf('\n') + 1;
          var lineText = before.substring(lineStart);
          if (lineText.startsWith('  ')) {
            self.editor.setRangeText('', lineStart, lineStart + 2, 'end');
          } else if (lineText.startsWith('\t')) {
            self.editor.setRangeText('', lineStart, lineStart + 1, 'end');
          }
        } else {
          self.editor.setRangeText('  ', start, end, 'end');
        }
        self.renderPreview();
        return;
      }

      var isCtrl = e.ctrlKey || e.metaKey;
      if (isCtrl && e.key.toLowerCase() === 'b') {
        e.preventDefault();
        self.insertMarkdown('bold');
      } else if (isCtrl && e.key.toLowerCase() === 'i') {
        e.preventDefault();
        self.insertMarkdown('italic');
      } else if (isCtrl && e.key.toLowerCase() === 'u') {
        e.preventDefault();
        self.insertMarkdown('underline');
      } else if (isCtrl && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        self.insertMarkdown('link');
      } else if (isCtrl && e.key === '/') {
        e.preventDefault();
        self.toggleComment();
      } else if (isCtrl && e.key.toLowerCase() === 's') {
        e.preventDefault();
        if (typeof self.options.onSave === 'function') {
          self.options.onSave();
        }
      }
    });

    // 工具栏操作按钮委托
    document.querySelectorAll('.tool-btn').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var action = btn.dataset.action;
        if (!action) return;
        switch (action) {
          case 'theme': self.toggleTheme(); break;
          case 'line-numbers': self.toggleLineNumbers(); break;
          case 'preview-only': self.togglePreviewOnly(); break;
          case 'import-md':
            var input = document.getElementById('import-file');
            if (input) input.click();
            break;
          case 'download-md': self.exportMarkdown(); break;
          case 'download-html': self.exportHTML(); break;
          case 'clear': self.clearEditor(); break;
          case 'help': self.showHelp(); break;
          case 'toggle-sidebar':
            if (parent && parent.layoutToggle) parent.layoutToggle();
            break;
          default:
            self.insertMarkdown(action);
        }
      });
    });

    // 标题下拉菜单
    document.querySelectorAll('.tool-dropdown-item').forEach(function (item) {
      item.addEventListener('click', function () {
        var action = item.dataset.action;
        if (action) self.insertMarkdown(action);
      });
    });

    // 导入文件事件
    var importFile = document.getElementById('import-file');
    if (importFile) {
      importFile.addEventListener('change', function (e) {
        var file = e.target.files[0];
        if (file) self.importMarkdown(file);
        e.target.value = '';
      });
    }

    // 弹窗关闭监听
    document.querySelectorAll('.modal-close, .modal-overlay').forEach(function (el) {
      el.addEventListener('click', function (e) {
        if (e.target === el) self.hideHelp();
      });
    });

    if (this.imgPreview) {
      this.imgPreview.addEventListener('click', function () {
        self.imgPreview.classList.remove('active');
      });
    }

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape') {
        self.hideHelp();
        if (self.imgPreview) self.imgPreview.classList.remove('active');
      }
    });

    this.initTooltip();
  };

  MMWikiEditor.prototype.getValue = function () {
    return this.editor ? this.editor.value : '';
  };

  MMWikiEditor.prototype.setValue = function (val) {
    if (this.editor) {
      this.editor.value = val;
      this.renderPreview();
      this.updateLineNumbers();
    }
  };

  window.MMWikiEditor = MMWikiEditor;
  window.MMWikiMarkdown = {
    render: MMWikiEditor.render
  };

})(window);
