package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/client"
	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/config"
)

func runDecrypt(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("decrypt <id> [-o dir] 필요")
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
	dest := filepath.Join(cfg.OutputDir, id+"-decrypt.zip")
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}
	c := client.New(cfg.ServgateURL)
	fmt.Fprintf(os.Stderr, "해독 %s → %s\n", id, dest)
	if err := c.Decrypt(id, dest); err != nil {
		return err
	}
	fmt.Println(dest)
	return nil
}
