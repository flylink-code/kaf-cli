import "./style.css";
import {
  Convert,
  CheckForUpdate,
  GetConfig,
  GetAIConfig,
  GetCoverPreview,
  GetVersion,
  InstallUpdate,
  InspectSource,
  OpenLastOutputDir,
  PickCover,
  PickTXT,
  SaveAIConfig,
  SaveConfig,
  TestAIConnection,
} from "./wailsjs/go/main/App";
import { EventsOn, OnFileDrop } from "./wailsjs/runtime/runtime";

// 运行时全局状态
const state = {
  converting: false,
  version: "dev",
  txtFile: "",
  coverFile: "",
  coverDataURL: "",
  bookname: "",
  author: "",
  fileSize: "",
  estimatedWords: "",
  format: "all",
  logLines: [],
  autoScroll: true,
  pendingUpdate: null,
};

const app = document.querySelector("#app");

app.innerHTML = `
  <div class="shell">
    <!-- 顶部状态与导航栏 -->
    <header class="topbar">
      <div class="topbar-brand">
        <div class="brand-icon">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1-2.5-2.5Z"/>
            <path d="M6 6h10"/>
            <path d="M6 10h10"/>
            <path d="M6 14h7"/>
          </svg>
        </div>
        <div class="brand-text">
          <div class="brand-title">
            <span class="brand-name">kaf-cli</span>
            <span class="brand-badge">EBOOK</span>
          </div>
          <span class="brand-desc">现代小说电子书工坊</span>
        </div>
      </div>

      <div class="topbar-actions">
        <div id="statusIndicator" class="status-indicator ready">
          <span class="status-dot"></span>
          <span id="statusText">准备就绪</span>
        </div>

        <button id="quickAIToggle" class="pill-btn ai-pill" type="button" title="点击配置 AI 智能处理">
          <span class="ai-spark-icon">✨</span>
          <span id="aiPillLabel">AI 智能分析</span>
          <span id="aiPillBadge" class="pill-badge off">OFF</span>
        </button>

        <button id="btnCheckUpdate" class="icon-action-btn" type="button" title="检查软件更新" aria-label="检查更新">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
            <path d="M3 3v5h5"/>
            <path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16"/>
            <path d="M16 21h5v-5"/>
          </svg>
        </button>

        <button id="openSettings" class="action-btn" type="button">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
            <circle cx="12" cy="12" r="3"/>
          </svg>
          <span>设置</span>
        </button>
      </div>
    </header>

    <!-- 主工作区：现代双栏布局 -->
    <main class="workspace">
      <!-- 左栏：书籍管理与核心参数控制 -->
      <section class="panel panel-workbench">
        <!-- 智能拖拽 / 电子书信息卡片容器 -->
        <div id="dropzoneContainer" class="dropzone-box">
          <!-- 未选文件时的拖拽引导 -->
          <div id="emptyDropzone" class="empty-dropzone">
            <div class="drop-icon-wrapper">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"/>
                <polyline points="14 2 14 8 20 8"/>
                <path d="M12 12v6"/>
                <path d="m9 15 3-3 3 3"/>
              </svg>
            </div>
            <div class="drop-text-group">
              <span class="drop-main-text">将 TXT 小说拖拽至此处</span>
              <span class="drop-sub-text">支持自动识别书名、作者与同目录封面图片</span>
            </div>
            <button id="btnBrowseTxt" class="browse-btn" type="button">点击选择文件</button>
          </div>

          <!-- 已选文件后的精装书预览卡片 -->
          <div id="activeBookCard" class="active-book-card" hidden>
            <!-- 3D 书封展示与替换触发 -->
            <div class="book-cover-stage" id="coverClickTarget" title="点击更换封面图片">
              <div class="book-cover-3d" id="coverDisplay">
                <div class="book-spine"></div>
                <img id="coverImg" class="cover-image" alt="书籍封面" hidden />
                <div id="coverPlaceholder" class="cover-placeholder">
                  <span id="coverBadgeChar" class="cover-char">书</span>
                  <span id="coverBadgeTitle" class="cover-mini-title">电子书</span>
                  <span class="cover-tag">KAF</span>
                </div>
              </div>
              <span class="cover-tip-text">点击换封面</span>
            </div>

            <!-- 书籍元数据与编辑 -->
            <div class="book-meta-info">
              <div class="meta-field">
                <label class="meta-label">书名</label>
                <input id="inputBookname" class="meta-input meta-input-title" placeholder="小说书名" />
              </div>

              <div class="meta-field">
                <label class="meta-label">作者</label>
                <input id="inputAuthor" class="meta-input" placeholder="作者名（留空自动识别）" />
              </div>

              <div class="meta-tags-row">
                <span id="badgeSize" class="meta-badge" title="文件体积">-- MB</span>
                <span id="badgeWords" class="meta-badge" title="估算字数">-- 万字</span>
                <button id="btnRemoveCover" class="text-link-btn" type="button" title="重置回默认封面">重置封面</button>
              </div>

              <div class="meta-file-path" id="displayTxtPath" title="点击重新选择">
                未选择文件
              </div>
            </div>
          </div>
        </div>

        <!-- 输出格式选择器：现代药丸分段控件 -->
        <div class="control-group">
          <div class="group-header">
            <span class="group-title">输出格式</span>
            <span class="group-hint">全选将同时打包 EPUB、MOBI 及 AZW3</span>
          </div>
          <div class="format-segmented-pill" id="formatSegmented">
            <button class="seg-item active" type="button" data-format="all">ALL (全部)</button>
            <button class="seg-item" type="button" data-format="epub">EPUB</button>
            <button class="seg-item" type="button" data-format="mobi">MOBI</button>
            <button class="seg-item" type="button" data-format="azw3">AZW3</button>
          </div>
        </div>

        <!-- 转换核心行动区 -->
        <div class="workbench-action-row">
          <button id="btnConvert" class="hero-convert-btn" type="button" disabled>
            <span class="btn-spinner" hidden></span>
            <span id="convertBtnLabel">开始转换</span>
          </button>
          <button id="btnOpenDir" class="open-dir-btn" type="button" disabled title="打开生成书籍所在的文件夹">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>
            </svg>
            <span>打开输出目录</span>
          </button>
        </div>
      </section>

      <!-- 右栏：AI 智能诊断指示器与现代终端控制台 -->
      <section class="panel panel-monitor">
        <!-- AI 智能流水线状态卡片（AI 开启时显示） -->
        <div id="aiPipelineCard" class="ai-pipeline-card" hidden>
          <div class="pipeline-header">
            <div class="pipeline-title">
              <span class="ai-sparkle">✨</span>
              <span>AI 语义深度重构流水线</span>
            </div>
            <span class="pipeline-tag" id="pipelineStatusTag">待机中</span>
          </div>
          <div class="pipeline-steps">
            <div class="step-item" id="stepStructure">
              <span class="step-dot"></span>
              <span class="step-label">章节目录诊断</span>
            </div>
            <div class="step-item" id="stepTypography">
              <span class="step-dot"></span>
              <span class="step-label">正文排版规范</span>
            </div>
            <div class="step-item" id="stepMetadata">
              <span class="step-dot"></span>
              <span class="step-label">精炼简介生成</span>
            </div>
          </div>
        </div>

        <!-- 现代终端代码控制台 -->
        <div class="terminal-card">
          <div class="terminal-topbar">
            <div class="terminal-dots">
              <span class="dot dot-red"></span>
              <span class="dot dot-yellow"></span>
              <span class="dot dot-green"></span>
            </div>
            <div class="terminal-title">执行输出控制台</div>
            <div class="terminal-actions">
              <button id="btnClearLog" class="terminal-action-btn" type="button" title="清空日志">清空</button>
              <button id="btnCopyLog" class="terminal-action-btn" type="button" title="复制日志到剪贴板">复制</button>
            </div>
          </div>

          <div id="terminalBody" class="terminal-body">
            <div id="logOutput" class="terminal-output">
              <div class="terminal-welcome-line">kaf-cli 现代电子书工坊已就绪。请选择或拖入 TXT 小说开始转换…</div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>

  <!-- 偏好与 AI 设置模态框 -->
  <div id="settingsModal" class="modal-overlay" hidden>
    <div class="modal-backdrop" data-close-settings></div>
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="settingsTitle">
      <div class="modal-header">
        <div>
          <h2 id="settingsTitle" class="modal-title">转换偏好与 AI 增强</h2>
          <p class="modal-subtitle">自定义排版细节、AI 大模型接入与高级分章规则</p>
        </div>
        <button id="settingsClose" class="modal-close-btn" type="button" aria-label="关闭">×</button>
      </div>

      <div class="modal-body">
        <!-- 基础排版选项 -->
        <div class="settings-section">
          <div class="section-header-row">
            <div>
              <h3 class="section-title">基础偏好</h3>
              <p class="section-subtitle">引擎已内置章节重复合并与「」直角引号规范化，由底层自适应全自动接管</p>
            </div>
          </div>
          <div class="switch-grid">
            <label class="switch-card">
              <div class="switch-text">
                <span class="switch-label">附带制作说明</span>
                <span class="switch-hint">在生成的电子书首尾附带制作教程与转换说明信息</span>
              </div>
              <input id="cfgTips" type="checkbox" />
            </label>
          </div>
        </div>

        <!-- AI 大模型增强设置 -->
        <div class="settings-section">
          <div class="section-header-row">
            <div>
              <h3 class="section-title">AI 全自动语义智能托管</h3>
              <p class="section-subtitle">启用后全自动深度诊断目录疑点、优化细节排版、清理采集水印与生成书籍简介</p>
            </div>
            <label class="toggle-switch">
              <input id="cfgAIEnabled" type="checkbox" />
              <span class="toggle-slider"></span>
            </label>
          </div>

          <div id="aiConfigPanel" class="ai-config-panel">
            <!-- 预设快速填入 -->
            <div class="provider-preset-row">
              <span class="preset-label">服务商一键配置：</span>
              <button class="preset-pill" type="button" data-provider="deepseek">DeepSeek (官方推荐)</button>
              <button class="preset-pill" type="button" data-provider="openai">OpenAI (gpt-4o-mini)</button>
              <button class="preset-pill" type="button" data-provider="siliconflow">SiliconFlow (硅基流动)</button>
              <button class="preset-pill" type="button" data-provider="ollama">本地 Ollama (免Key)</button>
            </div>

            <div class="input-grid">
              <div class="input-field">
                <label for="cfgAIBaseURL">API 接口地址 (Base URL)</label>
                <input id="cfgAIBaseURL" placeholder="https://api.deepseek.com/v1" />
              </div>
              <div class="input-field">
                <label for="cfgAIModel">模型名称 (Model)</label>
                <input id="cfgAIModel" placeholder="deepseek-chat" />
              </div>
            </div>

            <div class="input-field">
              <label for="cfgAIAPIKey">API Key (密钥仅在本地安全加密保存)</label>
              <div class="input-with-action">
                <input id="cfgAIAPIKey" type="password" placeholder="sk-..." />
                <button id="btnToggleKey" class="action-suffix-btn" type="button">显示</button>
              </div>
            </div>

            <div class="input-field">
              <div class="field-header">
                <label for="cfgAISampleChars">正文智能抽样深度（字数）</label>
                <span class="field-value-tip" id="sampleCharsTip">0 表示仅分析目录，不上传正文</span>
              </div>
              <div class="range-row">
                <input id="cfgAISampleChars" type="range" min="0" max="8000" step="500" value="0" />
                <span id="sampleCharsDisplay" class="range-display">0 字</span>
              </div>
            </div>

            <div class="test-connection-row">
              <button id="btnTestAI" class="secondary-btn" type="button">测试模型连通性</button>
              <span id="aiTestFeedback" class="test-feedback"></span>
            </div>
          </div>
        </div>

        <!-- 高级匹配规则（折叠手风琴，去冗余） -->
        <details class="advanced-details">
          <summary class="advanced-summary">
            <span>高级分章正则规则 (通常保持留空即可)</span>
            <span class="summary-arrow">▾</span>
          </summary>
          <div class="advanced-body">
            <p class="advanced-tip">底层转换引擎已内置全能自适应分章与大写数字识别算法，绝大多数小说无需填写任何规则。</p>
            <div class="input-grid">
              <div class="input-field">
                <label for="cfgMatch">章节匹配正则表达式</label>
                <input id="cfgMatch" placeholder="留空自动识别；例：第.{1,8}章" />
              </div>
              <div class="input-field">
                <label for="cfgVolumeMatch">分卷匹配正则表达式</label>
                <input id="cfgVolumeMatch" placeholder="留空自动识别；填 false 禁用卷识别" />
              </div>
            </div>
          </div>
        </details>
      </div>

      <div class="modal-footer">
        <span class="footer-ver" id="footerVersion">vdev</span>
        <div class="modal-footer-btns">
          <button id="btnSettingsCancel" class="ghost-btn" type="button">取消</button>
          <button id="btnSettingsSave" class="primary-btn" type="button">保存设置</button>
        </div>
      </div>
    </div>
  </div>

  <!-- 软件更新提示模态框 -->
  <div id="updateModal" class="modal-overlay" hidden>
    <div class="modal-backdrop" data-close-update></div>
    <div class="modal-card modal-card-sm" role="dialog" aria-modal="true" aria-labelledby="updateModalTitle">
      <div class="modal-header">
        <div>
          <h2 id="updateModalTitle" class="modal-title">软件更新</h2>
          <p class="modal-subtitle">从 GitHub 自动同步最新版本</p>
        </div>
        <button id="updateClose" class="modal-close-btn" type="button" aria-label="关闭">×</button>
      </div>
      <div class="modal-body">
        <div id="updateContent" class="update-content">
          <p id="updateMessage"></p>
        </div>
      </div>
      <div class="modal-footer">
        <button id="btnUpdateCancel" class="ghost-btn" type="button">关闭</button>
        <button id="btnUpdateInstall" class="primary-btn" type="button" hidden>立即下载并安装</button>
      </div>
    </div>
  </div>
`;

// 获取核心 DOM 元素引用
const els = {
  // 顶栏与状态
  statusIndicator: document.getElementById("statusIndicator"),
  statusText: document.getElementById("statusText"),
  quickAIToggle: document.getElementById("quickAIToggle"),
  aiPillBadge: document.getElementById("aiPillBadge"),
  btnCheckUpdate: document.getElementById("btnCheckUpdate"),
  openSettings: document.getElementById("openSettings"),

  // 工作区与书籍卡片
  dropzoneContainer: document.getElementById("dropzoneContainer"),
  emptyDropzone: document.getElementById("emptyDropzone"),
  activeBookCard: document.getElementById("activeBookCard"),
  btnBrowseTxt: document.getElementById("btnBrowseTxt"),
  coverClickTarget: document.getElementById("coverClickTarget"),
  coverImg: document.getElementById("coverImg"),
  coverPlaceholder: document.getElementById("coverPlaceholder"),
  coverBadgeChar: document.getElementById("coverBadgeChar"),
  coverBadgeTitle: document.getElementById("coverBadgeTitle"),
  inputBookname: document.getElementById("inputBookname"),
  inputAuthor: document.getElementById("inputAuthor"),
  badgeSize: document.getElementById("badgeSize"),
  badgeWords: document.getElementById("badgeWords"),
  btnRemoveCover: document.getElementById("btnRemoveCover"),
  displayTxtPath: document.getElementById("displayTxtPath"),

  // 格式与操作
  formatSegmented: document.getElementById("formatSegmented"),
  btnConvert: document.getElementById("btnConvert"),
  convertBtnLabel: document.getElementById("convertBtnLabel"),
  btnSpinner: document.querySelector(".btn-spinner"),
  btnOpenDir: document.getElementById("btnOpenDir"),

  // AI 流水线指示
  aiPipelineCard: document.getElementById("aiPipelineCard"),
  pipelineStatusTag: document.getElementById("pipelineStatusTag"),
  stepStructure: document.getElementById("stepStructure"),
  stepTypography: document.getElementById("stepTypography"),
  stepMetadata: document.getElementById("stepMetadata"),

  // 控制台终端
  terminalBody: document.getElementById("terminalBody"),
  logOutput: document.getElementById("logOutput"),
  btnClearLog: document.getElementById("btnClearLog"),
  btnCopyLog: document.getElementById("btnCopyLog"),

  // 设置模态框
  settingsModal: document.getElementById("settingsModal"),
  settingsClose: document.getElementById("settingsClose"),
  btnSettingsCancel: document.getElementById("btnSettingsCancel"),
  btnSettingsSave: document.getElementById("btnSettingsSave"),
  footerVersion: document.getElementById("footerVersion"),

  cfgTips: document.getElementById("cfgTips"),
  cfgAIEnabled: document.getElementById("cfgAIEnabled"),
  aiConfigPanel: document.getElementById("aiConfigPanel"),
  cfgAIBaseURL: document.getElementById("cfgAIBaseURL"),
  cfgAIModel: document.getElementById("cfgAIModel"),
  cfgAIAPIKey: document.getElementById("cfgAIAPIKey"),
  btnToggleKey: document.getElementById("btnToggleKey"),
  cfgAISampleChars: document.getElementById("cfgAISampleChars"),
  sampleCharsDisplay: document.getElementById("sampleCharsDisplay"),
  btnTestAI: document.getElementById("btnTestAI"),
  aiTestFeedback: document.getElementById("aiTestFeedback"),
  cfgMatch: document.getElementById("cfgMatch"),
  cfgVolumeMatch: document.getElementById("cfgVolumeMatch"),

  // 更新模态框
  updateModal: document.getElementById("updateModal"),
  updateClose: document.getElementById("updateClose"),
  btnUpdateCancel: document.getElementById("btnUpdateCancel"),
  btnUpdateInstall: document.getElementById("btnUpdateInstall"),
  updateMessage: document.getElementById("updateMessage"),
};

// ----------------- UI 状态渲染逻辑 -----------------

function updateStatus(type, message) {
  els.statusIndicator.className = `status-indicator ${type}`;
  els.statusText.textContent = message;
}

function updateAIRefreshDisplay() {
  const enabled = els.cfgAIEnabled.checked;
  els.aiPillBadge.textContent = enabled ? "ON" : "OFF";
  els.aiPillBadge.className = `pill-badge ${enabled ? "on" : "off"}`;
  els.aiConfigPanel.classList.toggle("disabled", !enabled);
  els.aiPipelineCard.hidden = !enabled;
}

function renderBookPreview() {
  if (!state.txtFile) {
    els.emptyDropzone.hidden = false;
    els.activeBookCard.hidden = true;
    els.btnConvert.disabled = true;
    els.displayTxtPath.textContent = "未选择文件";
    return;
  }

  els.emptyDropzone.hidden = true;
  els.activeBookCard.hidden = false;
  els.btnConvert.disabled = state.converting;

  els.inputBookname.value = state.bookname;
  els.inputAuthor.value = state.author;
  els.badgeSize.textContent = state.fileSize || "-- MB";
  els.badgeWords.textContent = state.estimatedWords || "-- 万字";
  els.displayTxtPath.textContent = state.txtFile;

  // 封面渲染
  if (state.coverDataURL) {
    els.coverImg.src = state.coverDataURL;
    els.coverImg.hidden = false;
    els.coverPlaceholder.hidden = true;
  } else {
    els.coverImg.hidden = true;
    els.coverPlaceholder.hidden = false;
    const firstChar = (state.bookname || "书").trim().charAt(0);
    els.coverBadgeChar.textContent = firstChar;
    els.coverBadgeTitle.textContent = state.bookname || "电子书";
  }
}

function setConvertingState(isConverting) {
  state.converting = isConverting;
  els.btnConvert.disabled = isConverting || !state.txtFile;
  els.btnSpinner.hidden = !isConverting;
  els.convertBtnLabel.textContent = isConverting ? "正在打包转换…" : "开始转换";
  if (isConverting) {
    els.btnOpenDir.disabled = true;
    updateStatus("running", "正在转换…");
  }
}

// ----------------- 日志终端高亮与渲染 -----------------

function appendLog(rawText) {
  if (!rawText) return;
  const lines = rawText.split("\n");

  lines.forEach((line) => {
    if (!line && lines.length > 1) return;
    // 过滤底层第三方 mobi 库输出的纯数字调试行（如 Offset: 25）
    if (/^Offset:\s*\d+$/i.test(line.trim())) return;

    state.logLines.push(line);

    // AI 流水线步骤动态感知
    if (line.includes("AI: 正在分析章节结构")) {
      setPipelineStep("structure", "running");
    } else if (line.includes("AI: 结构分析完成")) {
      setPipelineStep("structure", "done");
    } else if (line.includes("AI: 正在分析排版问题")) {
      setPipelineStep("typography", "running");
    } else if (line.includes("未发现可机械修正") || line.includes("排版细节优化完成")) {
      setPipelineStep("typography", "done");
    } else if (line.includes("AI: 正在生成书籍简介")) {
      setPipelineStep("metadata", "running");
    } else if (line.includes("AI: 简介生成完成")) {
      setPipelineStep("metadata", "done");
      els.pipelineStatusTag.textContent = "分析完成";
    } else if (line.includes("排版/噪音/简介任务")) {
      setPipelineStep("typography", "skipped");
      setPipelineStep("metadata", "skipped");
    }

    const row = document.createElement("div");
    row.className = getLogLineClass(line);
    row.textContent = line;
    els.logOutput.appendChild(row);
  });

  if (state.autoScroll) {
    els.terminalBody.scrollTop = els.terminalBody.scrollHeight;
  }
}

function getLogLineClass(line) {
  const lower = line.toLowerCase();
  if (line.startsWith("AI:") || line.includes("AI/智能排版") || line.includes("智能目录")) return "term-line term-ai";
  if (line.includes("转换完成") || line.includes("PASS") || line.includes("生成EPUB电子书耗时")) return "term-line term-success";
  if (line.includes("匹配章节") || line.includes("读取文件耗时")) return "term-line term-info";
  if (line.includes("错误") || line.includes("失败") || lower.includes("error")) return "term-line term-error";
  if (line.includes("跳过") || line.includes("警告")) return "term-line term-warn";
  return "term-line";
}

function setPipelineStep(step, status) {
  const map = {
    structure: els.stepStructure,
    typography: els.stepTypography,
    metadata: els.stepMetadata,
  };
  const el = map[step];
  if (!el) return;
  el.className = `step-item step-${status}`;
}

function resetPipeline() {
  ["stepStructure", "stepTypography", "stepMetadata"].forEach((id) => {
    els[id].className = "step-item";
  });
  els.pipelineStatusTag.textContent = "待机中";
}

// ----------------- 文件智能解析与封面查找 -----------------

async function inspectAndLoadFile(filePath) {
  if (!filePath) return;
  updateStatus("loading", "正在解析书籍元数据…");

  try {
    const insight = await InspectSource(filePath);
    state.txtFile = filePath;
    state.bookname = insight?.bookname || "";
    state.author = insight?.author || "";
    state.coverFile = insight?.cover || "";
    state.coverDataURL = insight?.coverDataURL || "";
    state.fileSize = insight?.fileSize || "";
    state.estimatedWords = insight?.estimatedWords || "";

    renderBookPreview();
    updateStatus("ready", "书籍信息已解析");
    await persistConfig();
  } catch (err) {
    updateStatus("error", "解析失败");
    appendLog(`[错误] 解析小说文件失败: ${String(err)}`);
  }
}

async function handleCoverPick() {
  try {
    const path = await PickCover();
    if (!path) return;
    state.coverFile = path;
    const preview = await GetCoverPreview(path);
    state.coverDataURL = preview || "";
    renderBookPreview();
    await persistConfig();
  } catch (err) {
    appendLog(`[错误] 选择封面失败: ${String(err)}`);
  }
}

// ----------------- 配置加载与持久化 -----------------

async function loadSettings() {
  try {
    const v = await GetVersion();
    state.version = v || "dev";
    els.footerVersion.textContent = `v${state.version}`;

    const cfg = await GetConfig();
    if (cfg?.txt_file) {
      await inspectAndLoadFile(cfg.txt_file);
    }
    if (cfg?.author && !state.author) {
      state.author = cfg.author;
      els.inputAuthor.value = state.author;
    }
    els.cfgTips.checked = cfg?.tips ?? true;
    els.cfgMatch.value = cfg?.match || "";
    els.cfgVolumeMatch.value = cfg?.volume_match || "";

    // 格式恢复
    const formatValues = ["all", "epub", "mobi", "azw3"];
    state.format = formatValues[cfg?.format_index ?? 0] || "all";
    updateSegmentedUI(state.format);

    // AI 配置
    try {
      const ai = await GetAIConfig();
      els.cfgAIEnabled.checked = !!ai.enabled;
      els.cfgAIBaseURL.value = ai.base_url || "";
      els.cfgAIModel.value = ai.model || "";
      els.cfgAIAPIKey.value = ai.api_key || "";
      els.cfgAISampleChars.value = ai.sample_chars || 0;
      els.sampleCharsDisplay.textContent = `${ai.sample_chars || 0} 字`;
    } catch {
      // 默认缺省值
      els.cfgAIEnabled.checked = false;
    }
    updateAIRefreshDisplay();
  } catch (err) {
    console.error("加载配置出错:", err);
  }
}

async function persistConfig() {
  const formatIndex = ["all", "epub", "mobi", "azw3"].indexOf(state.format);
  await SaveConfig({
    txt_file: state.txtFile,
    cover_file: state.coverFile,
    author: els.inputAuthor.value.trim(),
    format_index: formatIndex >= 0 ? formatIndex : 0,
    match: els.cfgMatch.value.trim(),
    volume_match: els.cfgVolumeMatch.value.trim(),
    dedup: true,
    tips: els.cfgTips.checked,
    quotes: true,
  });
}

async function persistAIConfig() {
  await SaveAIConfig({
    enabled: els.cfgAIEnabled.checked,
    base_url: els.cfgAIBaseURL.value.trim(),
    api_key: els.cfgAIAPIKey.value.trim(),
    model: els.cfgAIModel.value.trim(),
    sample_chars: Number(els.cfgAISampleChars.value) || 0,
    tasks: {
      structure: true,
      typography: true,
      noise: true,
      metadata: true,
    },
  });
  updateAIRefreshDisplay();
}

function updateSegmentedUI(format) {
  state.format = format;
  els.formatSegmented.querySelectorAll(".seg-item").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.format === format);
  });
}

// ----------------- 转换调度逻辑 -----------------

async function triggerConvert() {
  if (!state.txtFile || state.converting) return;

  state.logLines = [];
  els.logOutput.innerHTML = "";
  appendLog(`正在初始化电子书转换工程: ${state.bookname || state.txtFile}`);
  setConvertingState(true);
  resetPipeline();
  if (els.cfgAIEnabled.checked) {
    els.pipelineStatusTag.textContent = "AI 介入分析中";
  }

  // 同步保存当前界面编辑的书名作者
  state.bookname = els.inputBookname.value.trim();
  state.author = els.inputAuthor.value.trim();
  await persistConfig();
  await persistAIConfig();

  const req = {
    txtFile: state.txtFile,
    coverFile: state.coverFile,
    author: state.author,
    format: state.format,
    match: els.cfgMatch.value.trim(),
    volumeMatch: els.cfgVolumeMatch.value.trim(),
    dedup: true,
    tips: els.cfgTips.checked,
    quotes: true,
    ai: {
      enabled: els.cfgAIEnabled.checked,
      structure: true,
      typography: true,
      noise: true,
      metadata: true,
      sampleChars: Number(els.cfgAISampleChars.value) || 0,
    },
  };

  try {
    await Convert(req);
  } catch (err) {
    appendLog(`\n[错误] 转换过程异常中断: ${String(err)}\n`);
    updateStatus("error", "转换失败");
    setConvertingState(false);
  }
}

// ----------------- 服务商快速预设 -----------------

const PROVIDER_PRESETS = {
  deepseek: {
    url: "https://api.deepseek.com/v1",
    model: "deepseek-chat",
  },
  openai: {
    url: "https://api.openai.com/v1",
    model: "gpt-4o-mini",
  },
  siliconflow: {
    url: "https://api.siliconflow.cn/v1",
    model: "deepseek-ai/DeepSeek-V3",
  },
  ollama: {
    url: "http://localhost:11434/v1",
    model: "qwen2.5:7b",
  },
};

function applyProviderPreset(providerKey) {
  const p = PROVIDER_PRESETS[providerKey];
  if (!p) return;
  els.cfgAIBaseURL.value = p.url;
  els.cfgAIModel.value = p.model;
  els.aiTestFeedback.textContent = `已自动应用 ${providerKey} 配置预设`;
  els.aiTestFeedback.className = "test-feedback success";
}

// ----------------- 事件监听绑定 -----------------

function setupEventListeners() {
  // 文件选择与拖放
  els.btnBrowseTxt.addEventListener("click", async () => {
    const path = await PickTXT();
    if (path) await inspectAndLoadFile(path);
  });

  els.displayTxtPath.addEventListener("click", async () => {
    const path = await PickTXT();
    if (path) await inspectAndLoadFile(path);
  });

  els.coverClickTarget.addEventListener("click", handleCoverPick);

  els.btnRemoveCover.addEventListener("click", async (e) => {
    e.stopPropagation();
    state.coverFile = "";
    state.coverDataURL = "";
    renderBookPreview();
    await persistConfig();
  });

  // 格式胶囊单选
  els.formatSegmented.addEventListener("click", (e) => {
    const btn = e.target.closest(".seg-item");
    if (!btn) return;
    updateSegmentedUI(btn.dataset.format);
    persistConfig();
  });

  // 主转换按钮
  els.btnConvert.addEventListener("click", triggerConvert);

  els.btnOpenDir.addEventListener("click", async () => {
    try {
      await OpenLastOutputDir();
    } catch (err) {
      appendLog(`[错误] 无法打开输出目录: ${String(err)}`);
    }
  });

  // 终端操作
  els.terminalBody.addEventListener("scroll", () => {
    const threshold = 32;
    const isBottom = els.terminalBody.scrollHeight - els.terminalBody.scrollTop - els.terminalBody.clientHeight <= threshold;
    state.autoScroll = isBottom;
  });

  els.btnClearLog.addEventListener("click", () => {
    state.logLines = [];
    state.autoScroll = true;
    els.logOutput.innerHTML = '<div class="terminal-welcome-line">控制台日志已清空。</div>';
  });

  els.btnCopyLog.addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText(state.logLines.join("\n"));
      const orig = els.btnCopyLog.textContent;
      els.btnCopyLog.textContent = "已复制";
      setTimeout(() => (els.btnCopyLog.textContent = orig), 1500);
    } catch (err) {
      appendLog(`[提示] 复制失败: ${err}`);
    }
  });

  // 设置弹窗
  els.openSettings.addEventListener("click", () => {
    els.settingsModal.hidden = false;
    els.aiTestFeedback.textContent = "";
  });

  els.quickAIToggle.addEventListener("click", () => {
    els.settingsModal.hidden = false;
    els.cfgAIEnabled.checked = !els.cfgAIEnabled.checked;
    updateAIRefreshDisplay();
  });

  const closeSettings = () => (els.settingsModal.hidden = true);
  els.settingsClose.addEventListener("click", closeSettings);
  els.btnSettingsCancel.addEventListener("click", closeSettings);
  els.settingsModal.querySelectorAll("[data-close-settings]").forEach((n) => n.addEventListener("click", closeSettings));

  els.btnSettingsSave.addEventListener("click", async () => {
    await persistConfig();
    await persistAIConfig();
    closeSettings();
    updateStatus("ready", "设置已保存");
  });

  els.cfgAIEnabled.addEventListener("change", updateAIRefreshDisplay);

  els.cfgAISampleChars.addEventListener("input", (e) => {
    const val = e.target.value;
    els.sampleCharsDisplay.textContent = `${val} 字`;
  });

  els.btnToggleKey.addEventListener("click", () => {
    const isPwd = els.cfgAIAPIKey.type === "password";
    els.cfgAIAPIKey.type = isPwd ? "text" : "password";
    els.btnToggleKey.textContent = isPwd ? "隐藏" : "显示";
  });

  // AI 预设按钮
  document.querySelectorAll(".preset-pill").forEach((btn) => {
    btn.addEventListener("click", () => applyProviderPreset(btn.dataset.provider));
  });

  // 测试 AI
  els.btnTestAI.addEventListener("click", async () => {
    els.aiTestFeedback.textContent = "正在发起模型连通测试…";
    els.aiTestFeedback.className = "test-feedback";
    els.btnTestAI.disabled = true;

    try {
      const res = await TestAIConnection({
        enabled: true,
        base_url: els.cfgAIBaseURL.value.trim(),
        api_key: els.cfgAIAPIKey.value.trim(),
        model: els.cfgAIModel.value.trim(),
        sample_chars: 0,
        tasks: { structure: true, typography: false, noise: false, metadata: false },
      });
      if (res?.ok) {
        els.aiTestFeedback.textContent = `✓ 连通成功: ${res.message || "响应正常"}`;
        els.aiTestFeedback.className = "test-feedback success";
      } else {
        els.aiTestFeedback.textContent = `✗ 连通失败: ${res?.message || "无回显"}`;
        els.aiTestFeedback.className = "test-feedback error";
      }
    } catch (err) {
      els.aiTestFeedback.textContent = `✗ 测试异常: ${String(err)}`;
      els.aiTestFeedback.className = "test-feedback error";
    } finally {
      els.btnTestAI.disabled = false;
    }
  });

  // 更新弹窗
  els.btnCheckUpdate.addEventListener("click", async () => {
    els.updateModal.hidden = false;
    els.btnUpdateInstall.hidden = true;
    els.updateMessage.textContent = "正在查询 GitHub 最新 Release…";

    try {
      const up = await CheckForUpdate();
      state.pendingUpdate = up;
      if (up?.available) {
        els.updateMessage.textContent = "发现新版本 " + up.latest + " (当前版本 v" + up.current + ")。\n点击下方按钮即可一键下载并静默执行安装更新。";
        els.btnUpdateInstall.hidden = false;
      } else {
        els.updateMessage.textContent = "当前已是最新版本 (v" + (up?.current || state.version) + ")，无需更新。";
      }
    } catch (err) {
      els.updateMessage.textContent = `检查更新失败: ${String(err)}`;
    }
  });

  const closeUpdate = () => (els.updateModal.hidden = true);
  els.updateClose.addEventListener("click", closeUpdate);
  els.btnUpdateCancel.addEventListener("click", closeUpdate);
  els.updateModal.querySelectorAll("[data-close-update]").forEach((n) => n.addEventListener("click", closeUpdate));

  els.btnUpdateInstall.addEventListener("click", async () => {
    if (!state.pendingUpdate?.downloadURL) return;
    els.btnUpdateInstall.disabled = true;
    els.btnUpdateCancel.disabled = true;
    els.updateMessage.textContent = "正在下载更新包并准备执行安装…";
    try {
      await InstallUpdate(state.pendingUpdate.downloadURL);
    } catch (err) {
      els.btnUpdateInstall.disabled = false;
      els.btnUpdateCancel.disabled = false;
      els.updateMessage.textContent = `启动安装程序失败: ${String(err)}`;
    }
  });

  // Wails 原生全局拖拽 (OnFileDrop)
  if (typeof OnFileDrop === "function") {
    OnFileDrop(async (x, y, paths) => {
      if (!paths || !paths.length) return;
      for (const p of paths) {
        const lower = p.toLowerCase();
        if (lower.endsWith(".txt")) {
          await inspectAndLoadFile(p);
        } else if (lower.endsWith(".png") || lower.endsWith(".jpg") || lower.endsWith(".jpeg") || lower.endsWith(".webp")) {
          state.coverFile = p;
          const preview = await GetCoverPreview(p);
          state.coverDataURL = preview || "";
          renderBookPreview();
          await persistConfig();
        }
      }
    }, true);
  }

  // HTML5 拖拽体验加持
  window.addEventListener("dragover", (e) => {
    e.preventDefault();
    els.dropzoneContainer.classList.add("drag-hover");
  });
  window.addEventListener("dragleave", (e) => {
    if (!e.relatedTarget) {
      els.dropzoneContainer.classList.remove("drag-hover");
    }
  });
  window.addEventListener("drop", (e) => {
    e.preventDefault();
    els.dropzoneContainer.classList.remove("drag-hover");
  });

  // 键盘快捷键
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      if (!els.settingsModal.hidden) closeSettings();
      if (!els.updateModal.hidden) closeUpdate();
    }
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      triggerConvert();
    }
  });

  // 监听后端 Wails 转换事件
  EventsOn("convert:log", appendLog);
  EventsOn("convert:state", (status) => {
    if (status === "running") {
      setConvertingState(true);
      updateStatus("running", "正在转换中…");
    } else if (status === "done") {
      setConvertingState(false);
      updateStatus("ready", "转换完成");
      els.btnOpenDir.disabled = false;
      resetPipeline();
      els.pipelineStatusTag.textContent = "已就绪";
    } else if (status === "error") {
      setConvertingState(false);
      updateStatus("error", "转换发生错误");
    }
  });
}

// 启动入口
(async function bootstrap() {
  setupEventListeners();
  await loadSettings();
})();
