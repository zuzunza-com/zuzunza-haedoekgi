package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/client"
	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/config"
	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/progress"
)

func runDownload(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("download <id> [-o dir] 필요")
	}
	id := args[0]
	out := "./"
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" {
			i++
			if i >= len(args) {
				return fmt.Errorf("-o 값 필요")
			}
			out = args[i]
		}
	}
	cfg, err := config.Load(out)
	if err != nil {
		return err
	}
	dest := filepath.Join(cfg.OutputDir, id+".zetswf")
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}
	c := client.New(cfg.ServgateURL)
	rep := progress.New()
	fmt.Fprintf(os.Stderr, "다운로드 %s → %s\n", id, dest)
	if err := c.Download(id, dest, rep.Callback); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr)
	fmt.Println(dest)
	return nil
}
