package utils

import (
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	DBDriver            string        `mapstructure:"DB_DRIVER"`
	DBSource            string        `mapstructure:"DB_SOURCE"`
	ServerAddress       string        `mapstructure:"SERVER_ADDRESS"`
	TokenSymmetricKey   string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuartion time.Duration `mapstructure:"ACCESS_TOKEN_DUARTION"`
}

// LoadConig reads configuration from config file or environment variables.
func LoadConig(path string) (config Config, err error) {
	viper.AddConfigPath(path)  // 配置文件所在目录
	viper.SetConfigName("app") // 配置文件名称（不包含后缀）
	viper.SetConfigType("env") // 配置文件类型（后缀）

	viper.AutomaticEnv() // 尝试加载环境变量中的配置信息

	err = viper.ReadInConfig() // 读取配置信息
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
