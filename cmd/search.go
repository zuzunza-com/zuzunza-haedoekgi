package cmd

import (
	"fmt"
	"strconv"

	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/client"
	"github.com/zuzunza-com/zuzunza-haedoekgi/internal/config"
)

func runSearch(args []string) error {
	var q, title, makerID, nickname string
	limit, offset := 20, 0
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-q":
			i++
			if i >= len(args) {
				return fmt.Errorf("-q 값 필요")
			}
			q = args[i]
		case "--title":
			i++
			if i >= len(args) {
				return fmt.Errorf("--title 값 필요")
			}
			title = args[i]
		case "--author-id":
			i++
			if i >= len(args) {
				return fmt.Errorf("--author-id 값 필요")
			}
			makerID = args[i]
		case "--nickname":
			i++
			if i >= len(args) {
				return fmt.Errorf("--nickname 값 필요")
			}
			nickname = args[i]
		case "--limit":
			i++
			if i >= len(args) {
				return fmt.Errorf("--limit 값 필요")
			}
			v, err := strconv.Atoi(args[i])
			if err != nil {
				return err
			}
			limit = v
		case "--offset":
			i++
			if i >= len(args) {
				return fmt.Errorf("--offset 값 필요")
			}
			v, err := strconv.Atoi(args[i])
			if err != nil {
				return err
			}
			offset = v
		default:
			return fmt.Errorf("알 수 없는 옵션: %s", args[i])
		}
	}

	cfg, err := config.Load(".")
	if err != nil {
		return err
	}
	c := client.New(cfg.ServgateURL)
	resp, err := c.Search(q, title, makerID, nickname, limit, offset)
	if err != nil {
		return err
	}
	fmt.Printf("총 %d건 (표시 %d–%d)\n", resp.Total, resp.Offset+1, resp.Offset+len(resp.Items))
	for _, item := range resp.Items {
		author := item.MakerNickname
		if author == "" {
			author = item.MakerDisplayName
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", item.ID, item.Title, author, item.CreatedAt)
	}
	return nil
}
