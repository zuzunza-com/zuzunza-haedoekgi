import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { open } from "@tauri-apps/plugin-dialog";

interface SearchItem {
  id: string;
  title: string;
  makerId: string;
  makerDisplayName: string;
  makerNickname: string;
  createdAt: string;
}

interface SearchResponse {
  items: SearchItem[];
  total: number;
  limit: number;
  offset: number;
}

interface QuotaResponse {
  downloadCount: number;
  downloadLimit: number;
  decryptCount: number;
  decryptLimit: number;
  bandwidthLimited: boolean;
}

interface DownloadProgress {
  id: string;
  written: number;
  total: number;
  limited: boolean;
}

const app = document.querySelector<HTMLDivElement>("#app")!;

let selectedId: string | null = null;
let outputDir = "";
let busy = false;
let currentOffset = 0;
const pageSize = 20;

function formatBytes(n: number): string {
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MiB`;
  if (n >= 1 << 10) return `${(n / (1 << 10)).toFixed(1)} KiB`;
  return `${n} B`;
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("ko-KR");
  } catch {
    return iso;
  }
}

function authorName(item: SearchItem): string {
  return item.makerNickname || item.makerDisplayName || item.makerId || "";
}

function formatError(e: unknown): string {
  if (typeof e === "string") return e;
  if (e instanceof Error) return e.message;
  return String(e);
}

function setStatus(msg: string, kind: "normal" | "error" | "success" = "normal") {
  const el = document.getElementById("status")!;
  el.textContent = msg;
  el.className = `status${kind !== "normal" ? ` ${kind}` : ""}`;
}

function setBusy(value: boolean) {
  busy = value;
  document.querySelectorAll("button").forEach((b) => {
    if (!b.id?.startsWith("row-")) {
      (b as HTMLButtonElement).disabled = value;
    }
  });
}

async function refreshQuota() {
  try {
    const q = await invoke<QuotaResponse>("cmd_quota");
    const limited = q.bandwidthLimited ? " · 대역폭 제한" : "";
    document.getElementById("quota")!.innerHTML =
      `다운로드 <strong>${q.downloadCount}/${q.downloadLimit}</strong>` +
      ` · 해독 <strong>${q.decryptCount}/${q.decryptLimit}</strong>${limited}`;
  } catch (e) {
    document.getElementById("quota")!.textContent = "할당량 조회 실패";
  }
}

async function doSearch(offset = 0) {
  const q = (document.getElementById("q") as HTMLInputElement).value.trim();
  const title = (document.getElementById("title") as HTMLInputElement).value.trim();
  const makerId = (document.getElementById("maker-id") as HTMLInputElement).value.trim();
  const nickname = (document.getElementById("nickname") as HTMLInputElement).value.trim();

  if (!q && !title && !makerId && !nickname) {
    setStatus("검색어, 제목, 작가 ID, 닉네임 중 하나 이상 입력하세요.", "error");
    return;
  }

  setBusy(true);
  setStatus("검색 중…");
  currentOffset = offset;

  try {
    const resp = await invoke<SearchResponse>("cmd_search", {
      q: q || null,
      title: title || null,
      makerId: makerId || null,
      nickname: nickname || null,
      limit: pageSize,
      offset,
    });
    renderResults(resp);
    setStatus(`총 ${resp.total}건`);
    await refreshQuota();
  } catch (e) {
    setStatus(formatError(e), "error");
  } finally {
    setBusy(false);
  }
}

function renderResults(resp: SearchResponse) {
  const tbody = document.getElementById("results-body")!;
  const summary = document.getElementById("results-summary")!;
  const from = resp.total === 0 ? 0 : resp.offset + 1;
  const to = resp.offset + resp.items.length;
  summary.textContent = `${from}–${to} / 총 ${resp.total}건`;

  (document.getElementById("prev-btn") as HTMLButtonElement).disabled = resp.offset <= 0;
  (document.getElementById("next-btn") as HTMLButtonElement).disabled =
    resp.offset + resp.items.length >= resp.total;

  if (resp.items.length === 0) {
    tbody.innerHTML = `<tr><td colspan="5" class="empty">검색 결과가 없습니다.</td></tr>`;
    selectedId = null;
    return;
  }

  tbody.innerHTML = resp.items
    .map(
      (item) => `
    <tr data-id="${item.id}" class="${selectedId === item.id ? "selected" : ""}">
      <td>${item.id}</td>
      <td>${escapeHtml(item.title)}</td>
      <td>${escapeHtml(authorName(item))}</td>
      <td>${formatDate(item.createdAt)}</td>
      <td class="actions">
        <button class="secondary" data-action="download" data-id="${item.id}">다운로드</button>
        <button class="secondary" data-action="decrypt" data-id="${item.id}">해독</button>
      </td>
    </tr>`
    )
    .join("");

  tbody.querySelectorAll("tr").forEach((row) => {
    row.addEventListener("click", (e) => {
      const target = e.target as HTMLElement;
      if (target.closest("button")) return;
      selectedId = row.getAttribute("data-id");
      tbody.querySelectorAll("tr").forEach((r) => r.classList.remove("selected"));
      row.classList.add("selected");
    });
  });

  tbody.querySelectorAll("button[data-action]").forEach((btn) => {
    btn.addEventListener("click", async (e) => {
      e.stopPropagation();
      const id = (btn as HTMLButtonElement).dataset.id!;
      const action = (btn as HTMLButtonElement).dataset.action!;
      if (action === "download") await doDownload(id);
      else await doDecrypt(id);
    });
  });
}

function escapeHtml(s: string | null | undefined): string {
  return String(s ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

async function pickOutputDir() {
  const selected = await open({ directory: true, multiple: false });
  if (selected && typeof selected === "string") {
    outputDir = selected;
    document.getElementById("output-path")!.textContent = outputDir;
  }
}

async function doDownload(id: string) {
  if (!outputDir) {
    setStatus("저장 폴더를 먼저 선택하세요.", "error");
    return;
  }
  setBusy(true);
  setStatus(`다운로드 중: ${id}…`);
  showProgress(true, id);

  try {
    const path = await invoke<string>("cmd_download", { id, outputDir: outputDir });
    setStatus(`다운로드 완료: ${path}`, "success");
    await refreshQuota();
  } catch (e) {
    setStatus(formatError(e), "error");
  } finally {
    showProgress(false);
    setBusy(false);
  }
}

async function doDecrypt(id: string) {
  if (!outputDir) {
    setStatus("저장 폴더를 먼저 선택하세요.", "error");
    return;
  }
  setBusy(true);
  setStatus(`해독 중: ${id}…`);

  try {
    const path = await invoke<string>("cmd_decrypt", { id, outputDir: outputDir });
    setStatus(`해독 완료: ${path}`, "success");
    await refreshQuota();
  } catch (e) {
    setStatus(formatError(e), "error");
  } finally {
    setBusy(false);
  }
}

function showProgress(active: boolean, id = "") {
  const wrap = document.getElementById("progress-wrap")!;
  wrap.classList.toggle("active", active);
  if (!active) return;
  const bar = document.getElementById("progress-bar") as HTMLProgressElement;
  bar.value = 0;
  bar.max = 100;
  bar.dataset.id = id;
  document.getElementById("progress-label")!.textContent = "준비 중…";
}

app.innerHTML = `
<header>
  <div>
    <h1>해독기</h1>
    <div class="subtitle">ZUZUNZA 플래시 아카이브 검색 · 다운로드 · 해독</div>
  </div>
  <div class="quota-bar" id="quota">할당량 불러오는 중…</div>
</header>
<main>
  <section class="search-panel">
    <div class="search-grid">
      <label>통합 검색<input id="q" type="text" placeholder="ID 또는 키워드" /></label>
      <label>제목<input id="title" type="text" placeholder="제목" /></label>
      <label>작가 ID<input id="maker-id" type="text" placeholder="maker_id" /></label>
      <label>닉네임<input id="nickname" type="text" placeholder="닉네임" /></label>
      <button class="primary" id="search-btn">검색</button>
    </div>
    <div class="output-row">
      <span class="path" id="output-path">저장 폴더를 선택하세요</span>
      <button class="secondary" id="pick-dir-btn">폴더 선택</button>
      <button class="secondary" id="download-btn" disabled>선택 항목 다운로드</button>
      <button class="secondary" id="decrypt-btn" disabled>선택 항목 해독</button>
    </div>
  </section>
  <section class="results-panel">
    <div class="results-header">
      <span id="results-summary">검색어를 입력하고 검색하세요.</span>
      <div>
        <button class="secondary" id="prev-btn" disabled>이전</button>
        <button class="secondary" id="next-btn" disabled>다음</button>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>ID</th><th>제목</th><th>작가</th><th>날짜</th><th>작업</th>
          </tr>
        </thead>
        <tbody id="results-body">
          <tr><td colspan="5" class="empty">검색 결과가 여기에 표시됩니다.</td></tr>
        </tbody>
      </table>
    </div>
  </section>
</main>
<footer>
  <div class="progress-wrap" id="progress-wrap">
    <progress id="progress-bar" value="0" max="100"></progress>
    <span class="progress-label" id="progress-label"></span>
  </div>
  <div class="status" id="status"></div>
</footer>
`;

document.getElementById("search-btn")!.addEventListener("click", () => doSearch(0));
document.getElementById("pick-dir-btn")!.addEventListener("click", () => pickOutputDir());
document.getElementById("download-btn")!.addEventListener("click", () => {
  if (selectedId) doDownload(selectedId);
});
document.getElementById("decrypt-btn")!.addEventListener("click", () => {
  if (selectedId) doDecrypt(selectedId);
});
document.getElementById("prev-btn")!.addEventListener("click", () => {
  if (currentOffset >= pageSize) doSearch(currentOffset - pageSize);
});
document.getElementById("next-btn")!.addEventListener("click", () => {
  doSearch(currentOffset + pageSize);
});

["q", "title", "maker-id", "nickname"].forEach((id) => {
  document.getElementById(id)!.addEventListener("keydown", (e) => {
    if ((e as KeyboardEvent).key === "Enter") doSearch(0);
  });
});

listen<DownloadProgress>("download-progress", (event) => {
  const { id, written, total, limited } = event.payload;
  const bar = document.getElementById("progress-bar") as HTMLProgressElement;
  if (bar.dataset.id !== id) return;
  const label = document.getElementById("progress-label")!;
  if (total > 0) {
    bar.value = written;
    bar.max = total;
    const pct = ((written / total) * 100).toFixed(0);
    label.textContent = `${formatBytes(written)} / ${formatBytes(total)} (${pct}%)${limited ? " · 제한" : ""}`;
  } else {
    label.textContent = `${formatBytes(written)}${limited ? " · 제한" : ""}`;
  }
});

(async () => {
  try {
    outputDir = await invoke<string>("cmd_default_output_dir");
    document.getElementById("output-path")!.textContent = outputDir;
  } catch {
    /* user picks manually */
  }
  await refreshQuota();

  const updateSelectionButtons = () => {
    const has = !!selectedId;
    (document.getElementById("download-btn") as HTMLButtonElement).disabled = !has || busy;
    (document.getElementById("decrypt-btn") as HTMLButtonElement).disabled = !has || busy;
  };
  setInterval(updateSelectionButtons, 200);
})();
