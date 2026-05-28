import "./style.css";
import {
  Convert,
  GetConfig,
  InspectSource,
  OpenLastOutputDir,
  PickCover,
  PickTXT,
  SaveConfig,
} from "./wailsjs/go/main/App";
import { EventsOn } from "./wailsjs/runtime/runtime";

const state = {
  converting: false,
  log: "",
};

const app = document.querySelector("#app");

app.innerHTML = `
  <div class="shell">
    <aside class="hero-card">
      <div class="hero-badge">kaf-cli</div>
      <h1>电子书转换工作台</h1>
      <p class="hero-copy">
        用更轻松的方式，把 TXT 整理成 EPUB、MOBI、AZW3。自动识别书名、作者、封面，并实时输出转换日志。
      </p>
      <div class="hero-list">
        <div>拖拽 TXT 或手动选择</div>
        <div>默认封面支持 PNG / JPG / JPEG</div>
        <div>保留现有 Go 转换核心逻辑</div>
      </div>
    </aside>

    <main class="workspace">
      <section class="panel">
        <div class="panel-head">
          <div>
            <p class="eyebrow">素材</p>
            <h2>输入与识别</h2>
          </div>
          <div id="statusChip" class="status-chip">准备就绪</div>
        </div>

        <div class="form-grid">
          <label class="field field-wide">
            <span>TXT 文件</span>
            <div class="input-row">
              <input id="txtFile" placeholder="选择小说 TXT 文件，支持拖拽导入" />
              <button id="pickTxt" class="ghost-btn" type="button">选择 TXT</button>
            </div>
          </label>

          <label class="field field-wide">
            <span>封面图片</span>
            <div class="input-row">
              <input id="coverFile" placeholder="可选，留空时会尝试自动匹配同名封面" />
              <button id="pickCover" class="ghost-btn" type="button">选择封面</button>
            </div>
          </label>

          <label class="field">
            <span>作者</span>
            <input id="author" placeholder="可留空，程序会尽量从文件名自动提取" />
          </label>

          <div class="field meta-card">
            <span>自动识别书名</span>
            <strong id="bookname">书名将在选择 TXT 后显示</strong>
          </div>
        </div>
      </section>

      <section class="panel">
        <div class="panel-head">
          <div>
            <p class="eyebrow">参数</p>
            <h2>转换选项</h2>
          </div>
        </div>

        <div class="options-grid">
          <label class="field">
            <span>输出格式</span>
            <select id="format">
              <option value="all">all</option>
              <option value="epub">epub</option>
              <option value="mobi">mobi</option>
              <option value="azw3">azw3</option>
            </select>
          </label>

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

        <div class="action-row">
          <button id="convert" class="primary-btn" type="button">开始转换</button>
          <button id="openDir" class="ghost-btn" type="button" disabled>打开输出目录</button>
        </div>
      </section>

      <section class="panel panel-log">
        <div class="panel-head">
          <div>
            <p class="eyebrow">输出</p>
            <h2>转换日志</h2>
          </div>
          <div class="panel-tip">这里会实时显示解析进度、输出结果和错误信息。</div>
        </div>
        <pre id="log" class="log-box">等待开始转换…</pre>
      </section>
    </main>
  </div>
`;

const els = {
  txtFile: document.getElementById("txtFile"),
  coverFile: document.getElementById("coverFile"),
  author: document.getElementById("author"),
  format: document.getElementById("format"),
  dedup: document.getElementById("dedup"),
  tips: document.getElementById("tips"),
  quotes: document.getElementById("quotes"),
  bookname: document.getElementById("bookname"),
  statusChip: document.getElementById("statusChip"),
  log: document.getElementById("log"),
  pickTxt: document.getElementById("pickTxt"),
  pickCover: document.getElementById("pickCover"),
  convert: document.getElementById("convert"),
  openDir: document.getElementById("openDir"),
};

function formatIndexToValue(index) {
  return ["all", "epub", "mobi", "azw3"][index] || "all";
}

function formatValueToIndex(value) {
  return ["all", "epub", "mobi", "azw3"].indexOf(value);
}

function renderStatus(text) {
  els.statusChip.textContent = text;
}

function setBookname(text) {
  els.bookname.textContent = text || "书名将在选择 TXT 后显示";
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

async function saveConfig() {
  await SaveConfig({
    txt_file: els.txtFile.value.trim(),
    cover_file: els.coverFile.value.trim(),
    author: els.author.value.trim(),
    format_index: formatValueToIndex(els.format.value),
    dedup: els.dedup.checked,
    tips: els.tips.checked,
    quotes: els.quotes.checked,
  });
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
  await saveConfig();

  try {
    await Convert({
      txtFile: els.txtFile.value.trim(),
      coverFile: els.coverFile.value.trim(),
      author: els.author.value.trim(),
      format: els.format.value,
      dedup: els.dedup.checked,
      tips: els.tips.checked,
      quotes: els.quotes.checked,
    });
  } catch (err) {
    appendLog(`\n${String(err || "转换失败")}\n`);
    renderStatus("转换失败");
    setConverting(false);
  }
}

async function bootstrap() {
  const cfg = await GetConfig();
  els.txtFile.value = cfg?.txt_file || "";
  els.coverFile.value = cfg?.cover_file || "";
  els.author.value = cfg?.author || "";
  els.format.value = formatIndexToValue(cfg?.format_index ?? 0);
  els.dedup.checked = cfg?.dedup ?? true;
  els.tips.checked = cfg?.tips ?? true;
  els.quotes.checked = cfg?.quotes ?? false;
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

els.txtFile.addEventListener("change", async () => {
  await inspectSource();
  await saveConfig();
});
els.coverFile.addEventListener("change", saveConfig);
els.author.addEventListener("change", saveConfig);
els.format.addEventListener("change", saveConfig);
els.dedup.addEventListener("change", saveConfig);
els.tips.addEventListener("change", saveConfig);
els.quotes.addEventListener("change", saveConfig);

bootstrap();
