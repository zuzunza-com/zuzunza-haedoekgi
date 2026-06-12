package cmd

import (
	"fmt"

	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/client"
	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/config"
)

func runQuota(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("quota 는 인자가 없습니다")
	}
	cfg, err := config.Load(".")
	if err != nil {
		return err
	}
	c := client.New(cfg.ServgateURL)
	q, err := c.Quota()
	if err != nil {
		return err
	}
	fmt.Println(client.FormatQuota(q))
	return nil
}
