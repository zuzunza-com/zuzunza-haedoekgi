package cmd

import (
	"fmt"
	"os"
)

var outputDir string

func Execute() error {
	if len(os.Args) < 2 {
		printUsage()
		return fmt.Errorf("하위 명령이 필요합니다")
	}
	switch os.Args[1] {
	case "search":
		return runSearch(os.Args[2:])
	case "download":
		return runDownload(os.Args[2:])
	case "decrypt":
		return runDecrypt(os.Args[2:])
	case "quota":
		return runQuota(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("알 수 없는 명령: %s", os.Args[1])
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `zuzunza-haedoekgi — 플래시 아카이브·추출 CLI

환경 변수:
  ZUZUNZA_SERVGATE_URL  servgate 베이스 URL (예: https://www.zuzunza.com/xpi)

명령:
  haedoekgi search [--title T] [--author-id ID] [--nickname N] [-q QUERY]
  haedoekgi download <id> [-o ./out/]
  haedoekgi decrypt <id> [-o ./out/]
  haedoekgi quota

`)
}
