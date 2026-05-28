(function(){const n=document.createElement("link").relList;if(n&&n.supports&&n.supports("modulepreload"))return;for(const i of document.querySelectorAll('link[rel="modulepreload"]'))p(i);new MutationObserver(i=>{for(const o of i)if(o.type==="childList")for(const d of o.addedNodes)d.tagName==="LINK"&&d.rel==="modulepreload"&&p(d)}).observe(document,{childList:!0,subtree:!0});function c(i){const o={};return i.integrity&&(o.integrity=i.integrity),i.referrerPolicy&&(o.referrerPolicy=i.referrerPolicy),i.crossOrigin==="use-credentials"?o.credentials="include":i.crossOrigin==="anonymous"?o.credentials="omit":o.credentials="same-origin",o}function p(i){if(i.ep)return;i.ep=!0;const o=c(i);fetch(i.href,o)}})();function f(e){return window.go.main.App.Convert(e)}function b(){return window.go.main.App.GetConfig()}function g(e){return window.go.main.App.InspectSource(e)}function w(){return window.go.main.App.OpenLastOutputDir()}function y(){return window.go.main.App.PickCover()}function k(){return window.go.main.App.PickTXT()}function x(e){return window.go.main.App.SaveConfig(e)}function E(e,n,c){return window.runtime.EventsOnMultiple(e,n,c)}function v(e,n){return E(e,n,-1)}const s={converting:!1,log:""},T=document.querySelector("#app");T.innerHTML=`
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
`;const t={txtFile:document.getElementById("txtFile"),coverFile:document.getElementById("coverFile"),author:document.getElementById("author"),format:document.getElementById("format"),dedup:document.getElementById("dedup"),tips:document.getElementById("tips"),quotes:document.getElementById("quotes"),bookname:document.getElementById("bookname"),statusChip:document.getElementById("statusChip"),log:document.getElementById("log"),pickTxt:document.getElementById("pickTxt"),pickCover:document.getElementById("pickCover"),convert:document.getElementById("convert"),openDir:document.getElementById("openDir")};function C(e){return["all","epub","mobi","azw3"][e]||"all"}function F(e){return["all","epub","mobi","azw3"].indexOf(e)}function r(e){t.statusChip.textContent=e}function m(e){t.bookname.textContent=e||"书名将在选择 TXT 后显示"}function h(e){s.log+=e,t.log.textContent=s.log||"等待开始转换…",t.log.scrollTop=t.log.scrollHeight}function l(e){s.converting=e,t.convert.disabled=e||!t.txtFile.value.trim(),t.pickTxt.disabled=e,t.pickCover.disabled=e}async function a(){await x({txt_file:t.txtFile.value.trim(),cover_file:t.coverFile.value.trim(),author:t.author.value.trim(),format_index:F(t.format.value),dedup:t.dedup.checked,tips:t.tips.checked,quotes:t.quotes.checked})}async function u(){const e=t.txtFile.value.trim();if(l(s.converting),!e){m(""),r("准备就绪");return}r("已选择 TXT");const n=await g(e);m(n==null?void 0:n.bookname),!t.author.value.trim()&&(n!=null&&n.author)&&(t.author.value=n.author),!t.coverFile.value.trim()&&(n!=null&&n.cover)&&(t.coverFile.value=n.cover)}async function I(){const e=await k();e&&(t.txtFile.value=e,await u(),await a())}async function L(){const e=await y();e&&(t.coverFile.value=e,await a())}async function B(){try{await w()}catch(e){r(String(e||"无法打开输出目录"))}}async function O(){s.log="",t.log.textContent="正在启动转换…",l(!0),t.openDir.disabled=!0,r("正在转换"),await a();try{await f({txtFile:t.txtFile.value.trim(),coverFile:t.coverFile.value.trim(),author:t.author.value.trim(),format:t.format.value,dedup:t.dedup.checked,tips:t.tips.checked,quotes:t.quotes.checked})}catch(e){h(`
${String(e||"转换失败")}
`),r("转换失败"),l(!1)}}async function P(){const e=await b();t.txtFile.value=(e==null?void 0:e.txt_file)||"",t.coverFile.value=(e==null?void 0:e.cover_file)||"",t.author.value=(e==null?void 0:e.author)||"",t.format.value=C((e==null?void 0:e.format_index)??0),t.dedup.checked=(e==null?void 0:e.dedup)??!0,t.tips.checked=(e==null?void 0:e.tips)??!0,t.quotes.checked=(e==null?void 0:e.quotes)??!1,await u(),l(!1),v("convert:log",h),v("convert:state",n=>{if(n==="running"){r("正在转换");return}if(n==="done"){r("转换完成"),t.openDir.disabled=!1,l(!1);return}n==="error"&&(r("转换失败"),l(!1))})}t.pickTxt.addEventListener("click",I);t.pickCover.addEventListener("click",L);t.convert.addEventListener("click",O);t.openDir.addEventListener("click",B);t.txtFile.addEventListener("change",async()=>{await u(),await a()});t.coverFile.addEventListener("change",a);t.author.addEventListener("change",a);t.format.addEventListener("change",a);t.dedup.addEventListener("change",a);t.tips.addEventListener("change",a);t.quotes.addEventListener("change",a);P();
