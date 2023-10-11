package bconf

import (
	"os"
	"path/filepath"
)

const (
	// EnvConfigFilePathKey (Set configuration file path export BTGO_CONFIG_FILE_PATH = xxxxxxbtgo.json)
	// (设置配置文件路径 export BTGO_CONFIG_FILE_PATH = xxx/xxx/btgo.json)
	EnvConfigFilePathKey     = "BTGO_CONFIG_FILE_PATH"
	EnvDefaultConfigFilePath = "/conf/btgo.json"
)

var env = new(zEnv)

type zEnv struct {
	configFilePath string
}

func init() {
	configFilePath := os.Getenv(EnvConfigFilePathKey)
	if configFilePath == "" {
		pwd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		configFilePath = filepath.Join(pwd, EnvDefaultConfigFilePath)
	}
	var err error
	configFilePath, err = filepath.Abs(configFilePath)
	if err != nil {
		panic(err)
	}
	env.configFilePath = configFilePath
}

func GetConfigFilePath() string {
	return env.configFilePath
}
