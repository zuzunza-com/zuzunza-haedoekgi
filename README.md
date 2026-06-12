# zuzunza-haedoekgi (해독기)

ZUZUNZA 플래시 아카이브 검색·`.zetswf` 다운로드·워터마크 ZIP 해독 CLI.

## 다운로드

[Releases](https://github.com/zuzunza-com/zuzunza-haedoekgi/releases)에서 OS에 맞는 바이너리를 받으세요.

| 파일 | 플랫폼 |
|------|--------|
| `haedoekgi-windows-amd64.exe` | Windows 64-bit |
| `haedoekgi-linux-amd64` | Linux x86_64 |
| `haedoekgi-linux-arm64` | Linux ARM64 |
| `haedoekgi-darwin-amd64` | macOS Intel |
| `haedoekgi-darwin-arm64` | macOS Apple Silicon |

## 설정

```bash
# Linux / macOS
export ZUZUNZA_SERVGATE_URL=https://www.zuzunza.com/xpi

# Windows (PowerShell)
$env:ZUZUNZA_SERVGATE_URL = "https://www.zuzunza.com/xpi"
```

## 사용 예

```bash
haedoekgi search --title "고양이" --nickname "작가닉"
haedoekgi search -q "flash123"

haedoekgi download 5114051 -o ./archive/
haedoekgi decrypt 5114051 -o ./archive/
haedoekgi quota
```

Windows에서는 `haedoekgi-windows-amd64.exe`를 `haedoekgi.exe`로 이름을 바꾸거나 그대로 실행하면 됩니다.

```powershell
.\haedoekgi-windows-amd64.exe search -q "고양이"
.\haedoekgi-windows-amd64.exe download 5114051 -o .\archive\
```

## 직접 빌드

```bash
# Linux / macOS
go build -o haedoekgi .

# Windows exe (크로스 컴파일)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o haedoekgi-windows-amd64.exe .
```
