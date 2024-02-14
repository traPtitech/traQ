package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// healthcheckCommand ヘルスチェックコマンド
func healthcheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "healthcheck",
		Short: "Run healthcheck",
		Run: func(_ *cobra.Command, _ []string) {
			logger := getCLILogger()
			defer logger.Sync()

			resp, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d/api/ping", c.Port))
			if err != nil {
				logger.Fatal("HTTP Client Error", zap.Error(err))
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				logger.Fatal("Unexpected status", zap.Int("status", resp.StatusCode))
			}
		},
	}
}
