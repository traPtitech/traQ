package cmd

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof init
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/blendle/zapdriver"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/traPtitech/traQ/utils/message"
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
	cobra.OnInitialize(func() {
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
		message.SetOrigin(c.Origin)
	})

	rootCommand.AddCommand(
		serveCommand(),
		migrateCommand(),
		migrateV2ToV3Command(),
		confCommand(),
		fileCommand(),
		stampCommand(),
		versionCommand(),
		healthcheckCommand(),
	)

	flags := rootCommand.PersistentFlags()
	flags.StringVarP(&configFile, "config", "c", "", "config file path")

	flags.Bool("dev", false, "development mode")
	bindPFlag(flags, "dev")
	flags.Bool("pprof", false, "expose pprof http interface")
	bindPFlag(flags, "pprof")
}

func Execute() error {
	return rootCommand.Execute()
}

func getLogger() (logger *zap.Logger) {
	if c.DevMode {
		return getCLILogger()
	}
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Encoding:         "json",
		EncoderConfig:    zapdriver.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, _ = cfg.Build(zapdriver.WrapCore(zapdriver.ServiceName("traq", fmt.Sprintf("%s.%s", Version, Revision))))
	return
}

func getCLILogger() (logger *zap.Logger) {
	level := zap.NewAtomicLevel()
	if c.DevMode {
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	cfg := zap.Config{
		Level:       level,
		Development: c.DevMode,
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
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, _ = cfg.Build()
	return
}

func bindPFlag(flags *pflag.FlagSet, key string, flag ...string) {
	if len(flag) == 0 {
		flag = []string{key}
	}
	if err := viper.BindPFlag(key, flags.Lookup(flag[0])); err != nil {
		panic(err)
	}
}

func waitSIGINT() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	signal.Stop(quit)
	close(quit)
	for range quit {
		continue
	}
}
