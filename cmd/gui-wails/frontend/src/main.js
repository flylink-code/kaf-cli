import "./style.css";
import {
  Convert,
  CheckForUpdate,
  GetConfig,
  GetAIConfig,
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
import { EventsOn } from "./wailsjs/runtime/runtime";

const state = {
  converting: false,
  log: "",
};

const app = document.querySelector("#app");

app.innerHTML = `
  <div class="shell">
    <header class="topbar">
      <div class="topbar-brand">
        <span class="hero-badge">kaf-cli</span>
        <h1>电子书转换</h1>
      </div>
      <div class="topbar-actions">
        <button id="openSettings" class="ghost-btn" type="button">
          设置
          <span id="aiStatusChip" class="inline-chip">AI关</span>
        </button>
        <div class="help-menu">
          <button id="helpMenuButton" class="icon-btn" type="button" aria-label="帮助和更新" title="帮助和更新">?</button>
          <div id="helpMenu" class="help-menu-popup" hidden>
            <button id="aboutButton" class="menu-item" type="button">关于 kaf-cli <span id="versionLabel"></span></button>
            <button id="checkUpdate" class="menu-item" type="button">检查更新</button>
          </div>
        </div>
        <div id="statusChip" class="status-chip">准备就绪</div>
      </div>
    </header>

    <main class="workspace">
      <section class="panel panel-main">
        <div class="form-grid">
          <label class="field field-wide">
            <span>TXT 文件</span>
            <div class="input-row">
              <input id="txtFile" placeholder="选择小说 TXT 文件" />
              <button id="pickTxt" class="ghost-btn" type="button">选择</button>
            </div>
          </label>

          <label class="field">
            <span>封面图片</span>
            <div class="input-row">
              <input id="coverFile" placeholder="可选，留空自动匹配" />
              <button id="pickCover" class="ghost-btn" type="button">选择</button>
            </div>
          </label>

          <label class="field">
            <span>作者</span>
            <input id="author" placeholder="可留空，自动从文件名提取" />
          </label>

          <div class="field meta-card">
            <span>书名</span>
            <strong id="bookname">选择 TXT 后显示</strong>
          </div>
        </div>

        <div class="action-row action-row-main">
          <label class="field field-inline">
            <span>输出格式</span>
            <select id="format">
              <option value="all">all</option>
              <option value="epub">epub</option>
              <option value="mobi">mobi</option>
              <option value="azw3">azw3</option>
            </select>
          </label>
          <button id="convert" class="primary-btn" type="button">开始转换</button>
          <button id="openDir" class="ghost-btn" type="button" disabled>打开输出目录</button>
        </div>
      </section>

      <section class="panel panel-log">
        <div class="panel-head panel-head-compact">
          <h2>转换日志</h2>
        </div>
        <pre id="log" class="log-box">等待开始转换…</pre>
      </section>
    </main>
  </div>

  <div id="updateModal" class="modal" hidden>
    <div class="modal-backdrop" data-close-update></div>
    <div class="modal-card update-modal-card" role="dialog" aria-modal="true" aria-labelledby="updateTitle">
      <div class="modal-header">
        <div>
          <p class="eyebrow">软件更新</p>
          <h2 id="updateTitle">检查更新</h2>
        </div>
        <button id="updateClose" class="modal-close" type="button" aria-label="关闭">×</button>
      </div>
      <div class="modal-body update-body">
        <p id="updateMessage">正在检查 GitHub Release…</p>
      </div>
      <div class="modal-footer">
        <button id="updateInstall" class="primary-btn" type="button" hidden>下载并安装</button>
        <button id="updateCancel" class="ghost-btn" type="button">关闭</button>
      </div>
    </div>
  </div>

  <div id="settingsModal" class="modal" hidden>
    <div class="modal-backdrop" data-close-modal></div>
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="settingsTitle">
      <div class="modal-header">
        <div>
          <p class="eyebrow">配置</p>
          <h2 id="settingsTitle">设置</h2>
        </div>
        <button id="settingsClose" class="modal-close" type="button" aria-label="关闭">×</button>
      </div>

      <div class="modal-body">
        <div class="modal-pane">
          <h3 class="settings-section-title">转换选项</h3>
          <div class="more-checks more-checks-modal">
            <label class="check-row">
              <input id="dedup" type="checkbox" />
              <span>合并重复目录行</span>
            </label>
            <label class="check-row">
              <input id="tips" type="checkbox" />
              <span>添加制作说明</span>
            </label>
            <label class="check-row">
              <input id="quotes" type="checkbox" />
              <span>对话引号优化</span>
            </label>
          </div>
          <label class="field field-wide">
            <span>章节匹配规则</span>
            <input id="match" placeholder="留空自动识别；例：第.{1,8}章" />
          </label>
          <label class="field field-wide">
            <span>卷匹配规则</span>
            <input id="volumeMatch" placeholder="留空自动识别；填 false 禁用卷识别" />
          </label>

          <h3 class="settings-section-title">AI 优化</h3>
          <label class="check-row check-row-wide">
            <input id="aiEnabled" type="checkbox" />
            <span>启用 AI 后处理</span>
          </label>

          <div class="options-grid options-grid-modal">
            <label class="field">
              <span>Base URL</span>
              <input id="aiBaseURL" placeholder="https://api.deepseek.com/v1" />
            </label>
            <label class="field">
              <span>Model</span>
              <input id="aiModel" placeholder="deepseek-chat" />
            </label>
          </div>

          <label class="field field-wide">
            <span>API Key</span>
            <div class="input-row">
              <input id="aiAPIKey" type="password" placeholder="sk-...（仅本地保存）" />
              <button id="aiToggleKey" class="ghost-btn" type="button">显示</button>
            </div>
          </label>

          <div class="more-checks more-checks-modal">
            <label class="check-row">
              <input id="aiStructure" type="checkbox" />
              <span>章节结构（疑点标题）</span>
            </label>
            <label class="check-row">
              <input id="aiTypography" type="checkbox" />
              <span>排版修正</span>
            </label>
            <label class="check-row">
              <input id="aiNoise" type="checkbox" />
              <span>噪音清理</span>
            </label>
            <label class="check-row">
              <input id="aiMetadata" type="checkbox" />
              <span>书籍简介</span>
            </label>
          </div>

          <label class="field field-compact">
            <span>正文抽样上限（0=不上传正文）</span>
            <input id="aiSampleChars" type="number" min="0" step="500" placeholder="0" />
          </label>

          <div class="action-row action-row-modal">
            <button id="aiTest" class="ghost-btn" type="button">测试连接</button>
            <span id="aiTestResult" class="ai-test-result"></span>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button id="settingsSave" class="primary-btn" type="button">保存</button>
        <button id="settingsCancel" class="ghost-btn" type="button">取消</button>
      </div>
    </div>
  </div>
`;

const els = {
  txtFile: document.getElementById("txtFile"),
  coverFile: document.getElementById("coverFile"),
  author: document.getElementById("author"),
  format: document.getElementById("format"),
  match: document.getElementById("match"),
  volumeMatch: document.getElementById("volumeMatch"),
  dedup: document.getElementById("dedup"),
  tips: document.getElementById("tips"),
  quotes: document.getElementById("quotes"),
  bookname: document.getElementById("bookname"),
  statusChip: document.getElementById("statusChip"),
  aiStatusChip: document.getElementById("aiStatusChip"),
  log: document.getElementById("log"),
  pickTxt: document.getElementById("pickTxt"),
  pickCover: document.getElementById("pickCover"),
  convert: document.getElementById("convert"),
  openDir: document.getElementById("openDir"),
  openSettings: document.getElementById("openSettings"),
  settingsModal: document.getElementById("settingsModal"),
  settingsClose: document.getElementById("settingsClose"),
  settingsCancel: document.getElementById("settingsCancel"),
  settingsSave: document.getElementById("settingsSave"),
  aiEnabled: document.getElementById("aiEnabled"),
  aiBaseURL: document.getElementById("aiBaseURL"),
  aiModel: document.getElementById("aiModel"),
  aiAPIKey: document.getElementById("aiAPIKey"),
  aiToggleKey: document.getElementById("aiToggleKey"),
  aiStructure: document.getElementById("aiStructure"),
  aiTypography: document.getElementById("aiTypography"),
  aiNoise: document.getElementById("aiNoise"),
  aiMetadata: document.getElementById("aiMetadata"),
  aiSampleChars: document.getElementById("aiSampleChars"),
  aiTest: document.getElementById("aiTest"),
  aiTestResult: document.getElementById("aiTestResult"),
  helpMenuButton: document.getElementById("helpMenuButton"),
  helpMenu: document.getElementById("helpMenu"),
  aboutButton: document.getElementById("aboutButton"),
  checkUpdate: document.getElementById("checkUpdate"),
  versionLabel: document.getElementById("versionLabel"),
  updateModal: document.getElementById("updateModal"),
  updateClose: document.getElementById("updateClose"),
  updateCancel: document.getElementById("updateCancel"),
  updateInstall: document.getElementById("updateInstall"),
  updateMessage: document.getElementById("updateMessage"),
};

let pendingUpdate;

function formatIndexToValue(index) {
  return ["all", "epub", "mobi", "azw3"][index] || "all";
}

function formatValueToIndex(value) {
  return ["all", "epub", "mobi", "azw3"].indexOf(value);
}

function renderStatus(text) {
  els.statusChip.textContent = text;
}

function renderAIStatus() {
  const on = els.aiEnabled.checked;
  els.aiStatusChip.textContent = on ? "AI开" : "AI关";
  els.aiStatusChip.classList.toggle("is-on", on);
}

function setBookname(text) {
  els.bookname.textContent = text || "选择 TXT 后显示";
}

function appendLog(chunk) {
  state.log += chunk;
  els.log.textContent = state.log || "等待开始转换…";
  els.log.scrollTop = els.log.scrollHeight;
}

function setConverting(converting) {
  state.converting = converting;
  els.convert.disabled = converting || !els.txtFile.value.trim();
  els.pickTxt.disabled = converting;
  els.pickCover.disabled = converting;
}

function openSettingsModal() {
  els.settingsModal.hidden = false;
  els.aiTestResult.textContent = "";
}

function closeSettingsModal() {
  els.settingsModal.hidden = true;
}

function toggleHelpMenu() {
  els.helpMenu.hidden = !els.helpMenu.hidden;
}

function closeHelpMenu() {
  els.helpMenu.hidden = true;
}

function openUpdateModal(message) {
  els.updateMessage.textContent = message;
  els.updateInstall.hidden = true;
  els.updateModal.hidden = false;
}

function closeUpdateModal() {
  els.updateModal.hidden = true;
}

async function checkForUpdate() {
  closeHelpMenu();
  openUpdateModal("正在检查 GitHub Release…");
  try {
    const update = await CheckForUpdate();
    pendingUpdate = update;
    if (update?.available) {
      els.updateMessage.textContent = `发现新版本 ${update.latest}（当前 ${update.current}）。将从 GitHub 下载安装程序，安装后自动替换当前版本。`;
      els.updateInstall.hidden = false;
      return;
    }
    els.updateMessage.textContent = `当前已是最新版本（${update?.current || "未知"}）。`;
  } catch (err) {
    els.updateMessage.textContent = `检查更新失败：${String(err || "未知错误")}`;
  }
}

async function installUpdate() {
  if (!pendingUpdate?.downloadURL) return;
  els.updateInstall.disabled = true;
  els.updateCancel.disabled = true;
  els.updateMessage.textContent = "正在从 GitHub 下载更新包，完成后将启动 Windows Installer…";
  try {
    await InstallUpdate(pendingUpdate.downloadURL);
  } catch (err) {
    els.updateInstall.disabled = false;
    els.updateCancel.disabled = false;
    els.updateMessage.textContent = `启动更新失败：${String(err || "未知错误")}`;
  }
}

function defaultAIConfig() {
  return {
    enabled: false,
    base_url: "",
    api_key: "",
    model: "",
    sample_chars: 0,
    tasks: { structure: true, typography: false, noise: false, metadata: false },
  };
}

function collectAIConfig() {
  return {
    enabled: els.aiEnabled.checked,
    base_url: els.aiBaseURL.value.trim(),
    api_key: els.aiAPIKey.value.trim(),
    model: els.aiModel.value.trim(),
    sample_chars: Number(els.aiSampleChars.value) || 0,
    tasks: {
      structure: els.aiStructure.checked,
      typography: els.aiTypography.checked,
      noise: els.aiNoise.checked,
      metadata: els.aiMetadata.checked,
    },
  };
}

function fillAIConfig(cfg) {
  const ai = cfg?.ai || {};
  els.aiEnabled.checked = !!ai.enabled;
  els.aiBaseURL.value = ai.base_url || "";
  els.aiModel.value = ai.model || "";
  els.aiAPIKey.value = ai.api_key || "";
  els.aiSampleChars.value = ai.sample_chars || 0;
  const tasks = ai.tasks || {};
  els.aiStructure.checked = tasks.structure ?? true;
  els.aiTypography.checked = !!tasks.typography;
  els.aiNoise.checked = !!tasks.noise;
  els.aiMetadata.checked = !!tasks.metadata;
  renderAIStatus();
}

function collectAIRequest() {
  const cfg = collectAIConfig();
  return {
    enabled: cfg.enabled,
    structure: cfg.tasks.structure,
    typography: cfg.tasks.typography,
    noise: cfg.tasks.noise,
    metadata: cfg.tasks.metadata,
    sampleChars: cfg.sample_chars,
  };
}

async function saveAIConfig() {
  await SaveAIConfig(collectAIConfig());
  renderAIStatus();
}

async function saveConfig() {
  await SaveConfig({
    txt_file: els.txtFile.value.trim(),
    cover_file: els.coverFile.value.trim(),
    author: els.author.value.trim(),
    format_index: formatValueToIndex(els.format.value),
    match: els.match.value.trim(),
    volume_match: els.volumeMatch.value.trim(),
    dedup: els.dedup.checked,
    tips: els.tips.checked,
    quotes: els.quotes.checked,
  });
}

async function saveAllSettings() {
  await saveAIConfig();
  await saveConfig();
}

async function testAI() {
  els.aiTestResult.textContent = "测试中…";
  els.aiTest.disabled = true;
  try {
    const result = await TestAIConnection(collectAIConfig());
    els.aiTestResult.textContent = result?.ok
      ? `✓ ${result.message || "连接成功"}`
      : `✗ ${result?.message || "连接失败"}`;
  } catch (err) {
    els.aiTestResult.textContent = `✗ ${String(err || "连接失败")}`;
  } finally {
    els.aiTest.disabled = false;
  }
}

async function inspectSource() {
  const txtPath = els.txtFile.value.trim();
  setConverting(state.converting);

  if (!txtPath) {
    setBookname("");
    renderStatus("准备就绪");
    return;
  }

  renderStatus("已选择 TXT");
  const info = await InspectSource(txtPath);
  setBookname(info?.bookname);
  if (!els.author.value.trim() && info?.author) {
    els.author.value = info.author;
  }
  if (!els.coverFile.value.trim() && info?.cover) {
    els.coverFile.value = info.cover;
  }
}

async function pickTxt() {
  const path = await PickTXT();
  if (!path) return;
  els.txtFile.value = path;
  await inspectSource();
  await saveConfig();
}

async function pickCover() {
  const path = await PickCover();
  if (!path) return;
  els.coverFile.value = path;
  await saveConfig();
}

async function openOutputDir() {
  try {
    await OpenLastOutputDir();
  } catch (err) {
    renderStatus(String(err || "无法打开输出目录"));
  }
}

async function convert() {
  state.log = "";
  els.log.textContent = "正在启动转换…";
  setConverting(true);
  els.openDir.disabled = true;
  renderStatus("正在转换");
  await saveAllSettings();

  try {
    await Convert({
      txtFile: els.txtFile.value.trim(),
      coverFile: els.coverFile.value.trim(),
      author: els.author.value.trim(),
      format: els.format.value,
      match: els.match.value.trim(),
      volumeMatch: els.volumeMatch.value.trim(),
      dedup: els.dedup.checked,
      tips: els.tips.checked,
      quotes: els.quotes.checked,
      ai: collectAIRequest(),
    });
  } catch (err) {
    appendLog(`\n${String(err || "转换失败")}\n`);
    renderStatus("转换失败");
    setConverting(false);
  }
}

async function bootstrap() {
  const currentVersion = await GetVersion();
  els.versionLabel.textContent = currentVersion || "dev";
  const cfg = await GetConfig();
  els.txtFile.value = cfg?.txt_file || "";
  els.coverFile.value = cfg?.cover_file || "";
  els.author.value = cfg?.author || "";
  els.format.value = formatIndexToValue(cfg?.format_index ?? 0);
  els.match.value = cfg?.match || "";
  els.volumeMatch.value = cfg?.volume_match || "";
  els.dedup.checked = cfg?.dedup ?? true;
  els.tips.checked = cfg?.tips ?? true;
  els.quotes.checked = cfg?.quotes ?? false;

  try {
    const ai = await GetAIConfig();
    fillAIConfig({ ai });
  } catch {
    fillAIConfig({ ai: defaultAIConfig() });
  }

  await inspectSource();
  setConverting(false);

  EventsOn("convert:log", appendLog);
  EventsOn("convert:state", (payload) => {
    if (payload === "running") {
      renderStatus("正在转换");
      return;
    }
    if (payload === "done") {
      renderStatus("转换完成");
      els.openDir.disabled = false;
      setConverting(false);
      return;
    }
    if (payload === "error") {
      renderStatus("转换失败");
      setConverting(false);
    }
  });
}

els.pickTxt.addEventListener("click", pickTxt);
els.pickCover.addEventListener("click", pickCover);
els.convert.addEventListener("click", convert);
els.openDir.addEventListener("click", openOutputDir);

els.openSettings.addEventListener("click", openSettingsModal);
els.settingsClose.addEventListener("click", closeSettingsModal);
els.settingsCancel.addEventListener("click", closeSettingsModal);
els.settingsModal.querySelectorAll("[data-close-modal]").forEach((node) => {
  node.addEventListener("click", closeSettingsModal);
});

els.settingsSave.addEventListener("click", async () => {
  try {
    await saveAllSettings();
    closeSettingsModal();
    renderStatus("设置已保存");
  } catch (err) {
    renderStatus(String(err || "保存设置失败"));
  }
});

els.txtFile.addEventListener("change", async () => {
  await inspectSource();
  await saveConfig();
});
els.coverFile.addEventListener("change", saveConfig);
els.author.addEventListener("change", saveConfig);
els.format.addEventListener("change", saveConfig);

els.aiEnabled.addEventListener("change", renderAIStatus);
els.aiToggleKey.addEventListener("click", () => {
  const isPwd = els.aiAPIKey.type === "password";
  els.aiAPIKey.type = isPwd ? "text" : "password";
  els.aiToggleKey.textContent = isPwd ? "隐藏" : "显示";
});
els.aiTest.addEventListener("click", testAI);

els.helpMenuButton.addEventListener("click", toggleHelpMenu);
els.aboutButton.addEventListener("click", () => {
  closeHelpMenu();
  openUpdateModal(`kaf-cli 电子书转换\n版本 ${els.versionLabel.textContent}`);
});
els.checkUpdate.addEventListener("click", checkForUpdate);
els.updateClose.addEventListener("click", closeUpdateModal);
els.updateCancel.addEventListener("click", closeUpdateModal);
els.updateInstall.addEventListener("click", installUpdate);
els.updateModal.querySelectorAll("[data-close-update]").forEach((node) => {
  node.addEventListener("click", closeUpdateModal);
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && !els.settingsModal.hidden) {
    closeSettingsModal();
  }
  if (e.key === "Escape" && !els.updateModal.hidden) {
    closeUpdateModal();
  }
});

document.addEventListener("click", (e) => {
  if (!e.target.closest(".help-menu")) closeHelpMenu();
});

bootstrap();
