package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// confCommand 設定確認・ベース設定プリントコマンド
func confCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "conf",
		Short: "Print loaded config variables",
		Run: func(cmd *cobra.Command, args []string) {
			bs, err := yaml.Marshal(c)
			if err != nil {
				log.Fatalf("unable to marshal config to YAML: %v", err)
			}
			fmt.Print(string(bs))
		},
	}
}
