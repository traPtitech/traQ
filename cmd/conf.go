package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"log"
)

// confコマンド
// 設定確認・ベース設定ファイル出力用
var confCommand = &cobra.Command{
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
