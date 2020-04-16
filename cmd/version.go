package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// versionCommand バージョンプリントコマンド
func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("traQ %s (revision %s)\n", Version, Revision)
		},
	}
}
