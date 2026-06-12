mod api;

use api::{decrypt, download, quota, search, QuotaResponse, SearchResponse};

#[tauri::command]
async fn cmd_search(
    q: Option<String>,
    title: Option<String>,
    maker_id: Option<String>,
    nickname: Option<String>,
    limit: Option<i64>,
    offset: Option<i64>,
) -> Result<SearchResponse, String> {
    search(q, title, maker_id, nickname, limit, offset).await
}

#[tauri::command]
async fn cmd_quota() -> Result<QuotaResponse, String> {
    quota().await
}

#[tauri::command]
async fn cmd_download(
    app: tauri::AppHandle,
    id: String,
    output_dir: String,
) -> Result<String, String> {
    download(&app, &id, &output_dir).await
}

#[tauri::command]
async fn cmd_decrypt(id: String, output_dir: String) -> Result<String, String> {
    decrypt(&id, &output_dir).await
}

#[tauri::command]
fn cmd_default_output_dir() -> Result<String, String> {
    dirs::download_dir()
        .or_else(dirs::document_dir)
        .map(|p| p.join("haedoekgi").to_string_lossy().into_owned())
        .ok_or_else(|| "기본 저장 폴더를 찾을 수 없습니다".into())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .invoke_handler(tauri::generate_handler![
            cmd_search,
            cmd_quota,
            cmd_download,
            cmd_decrypt,
            cmd_default_output_dir,
        ])
        .run(tauri::generate_context!())
        .expect("해독기 GUI 실행 오류");
}
