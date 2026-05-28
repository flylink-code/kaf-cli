(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))v(a);new MutationObserver(a=>{for(const i of a)if(i.type==="childList")for(const u of i.addedNodes)u.tagName==="LINK"&&u.rel==="modulepreload"&&v(u)}).observe(document,{childList:!0,subtree:!0});function d(a){const i={};return a.integrity&&(i.integrity=a.integrity),a.referrerPolicy&&(i.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?i.credentials="include":a.crossOrigin==="anonymous"?i.credentials="omit":i.credentials="same-origin",i}function v(a){if(a.ep)return;a.ep=!0;const i=d(a);fetch(a.href,i)}})();function g(e){return window.go.main.App.Convert(e)}function y(){return window.go.main.App.GetConfig()}function w(e){return window.go.main.App.InspectSource(e)}function k(){return window.go.main.App.OpenLastOutputDir()}function E(){return window.go.main.App.PickCover()}function x(){return window.go.main.App.PickTXT()}function T(e){return window.go.main.App.SaveConfig(e)}function C(e,n,d){return window.runtime.EventsOnMultiple(e,n,d)}function h(e,n){return C(e,n,-1)}const r={converting:!1,log:"",moreOpen:!1},I=document.querySelector("#app");I.innerHTML=`
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
          <label class="field field-compact">
            <span>输出格式</span>
            <select id="format">
              <option value="all">all</option>
              <option value="epub">epub</option>
              <option value="mobi">mobi</option>
              <option value="azw3">azw3</option>
            </select>
          </label>

          <div class="field field-compact">
            <span>高级选项</span>
            <button id="moreToggle" class="more-btn" type="button" aria-expanded="false">
              <span>更多</span>
              <span id="moreSummary" class="more-summary">已启用 1 项</span>
            </button>
          </div>
        </div>

        <div id="morePanel" class="more-panel" hidden>
          <div class="more-panel-head">
            <div>
              <p class="eyebrow">高级选项</p>
              <h3>按书源启用</h3>
            </div>
            <button id="moreClose" class="more-close" type="button" aria-label="关闭高级选项">×</button>
          </div>

          <div class="more-checks">
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
        </div>

        <div class="rules-grid">
          <label class="field field-wide">
            <span>章节匹配规则</span>
            <input id="match" placeholder="可选：自定义章节匹配正则；留空时自动识别" />
          </label>

          <label class="field field-wide">
            <span>卷匹配规则</span>
            <input id="volumeMatch" placeholder="可选：自定义卷匹配正则；填 false 可禁用卷识别" />
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
`;const t={txtFile:document.getElementById("txtFile"),coverFile:document.getElementById("coverFile"),author:document.getElementById("author"),format:document.getElementById("format"),match:document.getElementById("match"),volumeMatch:document.getElementById("volumeMatch"),dedup:document.getElementById("dedup"),tips:document.getElementById("tips"),quotes:document.getElementById("quotes"),moreToggle:document.getElementById("moreToggle"),morePanel:document.getElementById("morePanel"),moreClose:document.getElementById("moreClose"),moreSummary:document.getElementById("moreSummary"),bookname:document.getElementById("bookname"),statusChip:document.getElementById("statusChip"),log:document.getElementById("log"),pickTxt:document.getElementById("pickTxt"),pickCover:document.getElementById("pickCover"),convert:document.getElementById("convert"),openDir:document.getElementById("openDir")};function B(e){return["all","epub","mobi","azw3"][e]||"all"}function L(e){return["all","epub","mobi","azw3"].indexOf(e)}function l(e){t.statusChip.textContent=e}function c(){const e=[t.dedup.checked,t.tips.checked,t.quotes.checked].filter(Boolean).length;t.moreSummary.textContent=e>0?`已启用 ${e} 项`:"未启用"}function p(e){r.moreOpen=e,t.morePanel.hidden=!e,t.moreToggle.setAttribute("aria-expanded",e?"true":"false")}function f(e){t.bookname.textContent=e||"书名将在选择 TXT 后显示"}function b(e){r.log+=e,t.log.textContent=r.log||"等待开始转换…",t.log.scrollTop=t.log.scrollHeight}function s(e){r.converting=e,t.convert.disabled=e||!t.txtFile.value.trim(),t.pickTxt.disabled=e,t.pickCover.disabled=e}async function o(){await T({txt_file:t.txtFile.value.trim(),cover_file:t.coverFile.value.trim(),author:t.author.value.trim(),format_index:L(t.format.value),match:t.match.value.trim(),volume_match:t.volumeMatch.value.trim(),dedup:t.dedup.checked,tips:t.tips.checked,quotes:t.quotes.checked})}async function m(){const e=t.txtFile.value.trim();if(s(r.converting),!e){f(""),l("准备就绪");return}l("已选择 TXT");const n=await w(e);f(n==null?void 0:n.bookname),!t.author.value.trim()&&(n!=null&&n.author)&&(t.author.value=n.author),!t.coverFile.value.trim()&&(n!=null&&n.cover)&&(t.coverFile.value=n.cover)}async function F(){const e=await x();e&&(t.txtFile.value=e,await m(),await o())}async function O(){const e=await E();e&&(t.coverFile.value=e,await o())}async function P(){try{await k()}catch(e){l(String(e||"无法打开输出目录"))}}async function M(){r.log="",t.log.textContent="正在启动转换…",s(!0),t.openDir.disabled=!0,l("正在转换"),await o();try{await g({txtFile:t.txtFile.value.trim(),coverFile:t.coverFile.value.trim(),author:t.author.value.trim(),format:t.format.value,match:t.match.value.trim(),volumeMatch:t.volumeMatch.value.trim(),dedup:t.dedup.checked,tips:t.tips.checked,quotes:t.quotes.checked})}catch(e){b(`
${String(e||"转换失败")}
`),l("转换失败"),s(!1)}}async function S(){const e=await y();t.txtFile.value=(e==null?void 0:e.txt_file)||"",t.coverFile.value=(e==null?void 0:e.cover_file)||"",t.author.value=(e==null?void 0:e.author)||"",t.format.value=B((e==null?void 0:e.format_index)??0),t.match.value=(e==null?void 0:e.match)||"",t.volumeMatch.value=(e==null?void 0:e.volume_match)||"",t.dedup.checked=(e==null?void 0:e.dedup)??!0,t.tips.checked=(e==null?void 0:e.tips)??!0,t.quotes.checked=(e==null?void 0:e.quotes)??!1,c(),p(!1),await m(),s(!1),h("convert:log",b),h("convert:state",n=>{if(n==="running"){l("正在转换");return}if(n==="done"){l("转换完成"),t.openDir.disabled=!1,s(!1);return}n==="error"&&(l("转换失败"),s(!1))})}t.pickTxt.addEventListener("click",F);t.pickCover.addEventListener("click",O);t.convert.addEventListener("click",M);t.openDir.addEventListener("click",P);t.moreToggle.addEventListener("click",()=>{p(!r.moreOpen)});t.moreClose.addEventListener("click",()=>{p(!1)});t.txtFile.addEventListener("change",async()=>{await m(),await o()});t.coverFile.addEventListener("change",o);t.author.addEventListener("change",o);t.format.addEventListener("change",o);t.match.addEventListener("change",o);t.volumeMatch.addEventListener("change",o);t.dedup.addEventListener("change",async()=>{c(),await o()});t.tips.addEventListener("change",async()=>{c(),await o()});t.quotes.addEventListener("change",async()=>{c(),await o()});S();
