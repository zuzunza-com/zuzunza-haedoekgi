use futures_util::StreamExt;
use reqwest::header::{HeaderMap, HeaderValue};
use serde::{Deserialize, Serialize};
use std::path::Path;
use tauri::{AppHandle, Emitter};

pub const DEFAULT_SERVGATE_URL: &str = "https://www.zuzunza.com/xpi";
const CLIENT_HEADER: &str = "X-Zuzunza-Decoder-Client";
const CLIENT_VALUE: &str = "haedoekgi-gui/0.2";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchItem {
    pub id: String,
    pub title: String,
    #[serde(rename = "makerId")]
    pub maker_id: String,
    #[serde(rename = "makerDisplayName")]
    pub maker_display_name: String,
    #[serde(rename = "makerNickname")]
    pub maker_nickname: String,
    #[serde(rename = "createdAt")]
    pub created_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchResponse {
    pub items: Vec<SearchItem>,
    pub total: i64,
    pub limit: i64,
    pub offset: i64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuotaResponse {
    #[serde(rename = "downloadCount")]
    pub download_count: i64,
    #[serde(rename = "downloadLimit")]
    pub download_limit: i64,
    #[serde(rename = "decryptCount")]
    pub decrypt_count: i64,
    #[serde(rename = "decryptLimit")]
    pub decrypt_limit: i64,
    #[serde(rename = "bandwidthLimited")]
    pub bandwidth_limited: bool,
}

#[derive(Debug, Clone, Serialize)]
pub struct DownloadProgress {
    pub id: String,
    pub written: i64,
    pub total: i64,
    pub limited: bool,
}

fn servgate_url() -> String {
    std::env::var("ZUZUNZA_SERVGATE_URL")
        .ok()
        .filter(|s| !s.trim().is_empty())
        .unwrap_or_else(|| DEFAULT_SERVGATE_URL.to_string())
        .trim_end_matches('/')
        .to_string()
}

fn http_client() -> Result<reqwest::Client, String> {
    let mut headers = HeaderMap::new();
    headers.insert("Origin", HeaderValue::from_static("haedoekgi"));
    headers.insert(
        CLIENT_HEADER,
        HeaderValue::from_static(CLIENT_VALUE),
    );
    reqwest::Client::builder()
        .default_headers(headers)
        .timeout(std::time::Duration::from_secs(600))
        .build()
        .map_err(|e| e.to_string())
}

async fn check_response(resp: reqwest::Response) -> Result<reqwest::Response, String> {
    let status = resp.status();
    if status.is_success() {
        return Ok(resp);
    }
    let body = resp.text().await.unwrap_or_default();
    let msg = body.trim();
    match status.as_u16() {
        403 => Err("접근 거부: 해독기 클라이언트 헤더가 필요합니다".into()),
        429 => Err(format!("일일 해독 할당량 초과: {msg}")),
        _ => {
            if msg.is_empty() {
                Err(format!("API 오류 ({status})"))
            } else {
                Err(format!("API 오류 ({status}): {msg}"))
            }
        }
    }
}

pub async fn search(
    q: Option<String>,
    title: Option<String>,
    maker_id: Option<String>,
    nickname: Option<String>,
    limit: Option<i64>,
    offset: Option<i64>,
) -> Result<SearchResponse, String> {
    let client = http_client()?;
    let base = servgate_url();
    let mut query: Vec<(&str, String)> = Vec::new();
    if let Some(v) = q.filter(|s| !s.trim().is_empty()) {
        query.push(("q", v));
    }
    if let Some(v) = title.filter(|s| !s.trim().is_empty()) {
        query.push(("title", v));
    }
    if let Some(v) = maker_id.filter(|s| !s.trim().is_empty()) {
        query.push(("maker_id", v));
    }
    if let Some(v) = nickname.filter(|s| !s.trim().is_empty()) {
        query.push(("nickname", v));
    }
    if let Some(v) = limit {
        query.push(("limit", v.to_string()));
    }
    if let Some(v) = offset {
        query.push(("offset", v.to_string()));
    }

    let resp = client
        .get(format!("{base}/api/decoder/v1/search"))
        .query(&query)
        .send()
        .await
        .map_err(|e| e.to_string())?;
    let resp = check_response(resp).await?;
    resp.json::<SearchResponse>().await.map_err(|e| e.to_string())
}

pub async fn quota() -> Result<QuotaResponse, String> {
    let client = http_client()?;
    let base = servgate_url();
    let resp = client
        .get(format!("{base}/api/decoder/v1/quota"))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    let resp = check_response(resp).await?;
    resp.json::<QuotaResponse>().await.map_err(|e| e.to_string())
}

pub async fn download(
    app: &AppHandle,
    id: &str,
    output_dir: &str,
) -> Result<String, String> {
    let client = http_client()?;
    let base = servgate_url();
    let dest = Path::new(output_dir).join(format!("{id}.zetswf"));
    tokio::fs::create_dir_all(output_dir)
        .await
        .map_err(|e| e.to_string())?;

    let resp = client
        .get(format!("{base}/api/decoder/v1/flash/{id}/download"))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    let resp = check_response(resp).await?;
    let limited = resp
        .headers()
        .get("X-Decoder-Bandwidth-Limited")
        .and_then(|v| v.to_str().ok())
        .map(|v| v.eq_ignore_ascii_case("true"))
        .unwrap_or(false);
    let total = resp.content_length().map(|v| v as i64).unwrap_or(-1);

    let mut file = tokio::fs::File::create(&dest)
        .await
        .map_err(|e| e.to_string())?;
    let mut stream = resp.bytes_stream();
    let mut written: i64 = 0;
    let mut last_emit = std::time::Instant::now();

    while let Some(chunk) = stream.next().await {
        let chunk = chunk.map_err(|e| e.to_string())?;
        use tokio::io::AsyncWriteExt;
        file.write_all(&chunk).await.map_err(|e| e.to_string())?;
        written += chunk.len() as i64;
        if last_emit.elapsed().as_millis() >= 500 {
            let _ = app.emit(
                "download-progress",
                DownloadProgress {
                    id: id.to_string(),
                    written,
                    total,
                    limited,
                },
            );
            last_emit = std::time::Instant::now();
        }
    }

    let _ = app.emit(
        "download-progress",
        DownloadProgress {
            id: id.to_string(),
            written,
            total,
            limited,
        },
    );

    Ok(dest.to_string_lossy().into_owned())
}

pub async fn decrypt(id: &str, output_dir: &str) -> Result<String, String> {
    let client = http_client()?;
    let base = servgate_url();
    let dest = Path::new(output_dir).join(format!("{id}-decrypt.zip"));
    tokio::fs::create_dir_all(output_dir)
        .await
        .map_err(|e| e.to_string())?;

    let resp = client
        .get(format!("{base}/api/decoder/v1/flash/{id}/decrypt"))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    let resp = check_response(resp).await?;
    let bytes = resp.bytes().await.map_err(|e| e.to_string())?;
    tokio::fs::write(&dest, &bytes)
        .await
        .map_err(|e| e.to_string())?;
    Ok(dest.to_string_lossy().into_owned())
}
