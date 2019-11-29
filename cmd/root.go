package cmd

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // mysql driver
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"net/http"
	_ "net/http/pprof" // pprof init
	"strings"
	"time"
)

var (
	Version  string
	Revision string
)

var rootCommand = &cobra.Command{
	Use: "traQ",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// enable pprof http handler
		if viper.GetBool("pprof") {
			go func() { _ = http.ListenAndServe("0.0.0.0:6060", nil) }()
		}
	},
}

func Execute() error {
	return rootCommand.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	viper.SetDefault("origin", "http://localhost:3000")
	viper.SetDefault("port", 3000)
	viper.SetDefault("gzip", true)
	viper.SetDefault("accessLog.enabled", true)
	viper.SetDefault("accessLog.excludesHeartbeat", true)

	viper.SetDefault("pprof", false)
	viper.SetDefault("gormLogMode", false)

	viper.SetDefault("externalAuthentication.enabled", false)

	viper.SetDefault("mariadb.host", "127.0.0.1")
	viper.SetDefault("mariadb.port", 3306)
	viper.SetDefault("mariadb.username", "root")
	viper.SetDefault("mariadb.password", "password")
	viper.SetDefault("mariadb.database", "traq")
	viper.SetDefault("mariadb.connection.maxOpen", 0)
	viper.SetDefault("mariadb.connection.maxIdle", 2)
	viper.SetDefault("mariadb.connection.lifetime", 0)

	viper.SetDefault("storage.type", "local")
	viper.SetDefault("storage.local.dir", "./storage")

	viper.SetDefault("gcp.stackdriver.profiler.enabled", false)

	viper.SetDefault("oauth2.isRefreshEnabled", false)
	viper.SetDefault("oauth2.accessTokenExp", 60*60*24*365)

	viper.SetDefault("jwt.keys.public", "./keys/ec_pub.pem")
	viper.SetDefault("jwt.keys.private", "./keys/ec.pem")

	viper.SetDefault("skyway.secretKey", "")

	rootCommand.AddCommand(serveCommand)
	rootCommand.AddCommand(migrateCommand)
	rootCommand.AddCommand(versionCommand)
}

func initConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TRAQ")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Fatalf("failed to read config file: %v", err)
		}
	}
}

func getDatabase() (*gorm.DB, error) {
	engine, err := gorm.Open("mysql", fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_general_ci&parseTime=true",
		viper.GetString("mariadb.username"),
		viper.GetString("mariadb.password"),
		viper.GetString("mariadb.host"),
		viper.GetInt("mariadb.port"),
		viper.GetString("mariadb.database"),
	))
	if err != nil {
		return nil, err
	}
	engine.DB().SetMaxOpenConns(viper.GetInt("mariadb.connection.maxOpen"))
	engine.DB().SetMaxIdleConns(viper.GetInt("mariadb.connection.maxIdle"))
	engine.DB().SetConnMaxLifetime(time.Duration(viper.GetInt("mariadb.connection.lifetime")) * time.Second)
	engine.LogMode(viper.GetBool("gormLogMode"))
	return engine, nil
}
