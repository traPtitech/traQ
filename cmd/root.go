package cmd

import (
	"fmt"
	"github.com/blendle/zapdriver"
	_ "github.com/jinzhu/gorm/dialects/mysql" // mysql driver
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof init
	"strings"
)

var (
	Version  string
	Revision string
)

var (
	// configFile 設定ファイルyamlのパス
	configFile string
	// c 設定
	c Config
)

// rootコマンドはダミー。コマンドとしては使用しない
var rootCommand = &cobra.Command{
	Use: "traQ",
	// 全コマンド共通の前処理
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// enable pprof http handler
		if c.Pprof {
			go func() { _ = http.ListenAndServe("0.0.0.0:6060", nil) }()
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCommand.AddCommand(serveCommand)
	rootCommand.AddCommand(migrateCommand)
	rootCommand.AddCommand(confCommand)
	rootCommand.AddCommand(versionCommand)

	flags := rootCommand.PersistentFlags()
	flags.StringVarP(&configFile, "config", "c", "", "config file path")

	flags.Bool("dev", false, "development mode")
	bindPFlag(flags, "dev")
	flags.Bool("pprof", false, "expose pprof http interface")
	bindPFlag(flags, "pprof")
}

func initConfig() {
	if len(configFile) > 0 {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TRAQ")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("failed to read config file: %v", err)
		}
	}
	if err := viper.Unmarshal(&c); err != nil {
		log.Fatal(err)
	}
}

func Execute() error {
	return rootCommand.Execute()
}

func getLogger() (logger *zap.Logger) {
	if c.DevMode {
		cfg := zap.Config{
			Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
			Development: true,
			Encoding:    "console",
			EncoderConfig: zapcore.EncoderConfig{
				TimeKey:        "T",
				LevelKey:       "L",
				NameKey:        "N",
				CallerKey:      "C",
				MessageKey:     "M",
				StacktraceKey:  "S",
				LineEnding:     zapcore.DefaultLineEnding,
				EncodeLevel:    zapcore.CapitalColorLevelEncoder,
				EncodeTime:     zapcore.ISO8601TimeEncoder,
				EncodeDuration: zapcore.StringDurationEncoder,
				EncodeCaller:   zapcore.ShortCallerEncoder,
			},
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
		logger, _ = cfg.Build()
	} else {
		cfg := zap.Config{
			Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
			Encoding:         "json",
			EncoderConfig:    zapdriver.NewProductionEncoderConfig(),
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}
		logger, _ = cfg.Build(zapdriver.WrapCore(zapdriver.ServiceName("traq", fmt.Sprintf("%s.%s", Version, Revision))))
	}
	return
}

func bindPFlag(flags *pflag.FlagSet, key string) {
	if err := viper.BindPFlag(key, flags.Lookup(key)); err != nil {
		panic(err)
	}
}
