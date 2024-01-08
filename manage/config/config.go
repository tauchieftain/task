package config

import (
	"gopkg.in/ini.v1"
	"log"
	"os"
	"strings"
)

var config *ini.File

func init() {
	iniFile := "./manage.ini"
	var err error
	if _, err = os.Stat(iniFile); os.IsNotExist(err) {
		log.Fatalf("Config file [%s] not found", iniFile)
	}
	config, err = ini.Load(iniFile)
	if err != nil {
		log.Fatalf("Parse config file [%s] failed, err: %s", iniFile, err.Error())
	}
	setDbConfig()
}

func GetSection(section string) *ini.Section {
	return config.Section(section)
}

func IsDebug() bool {
	debug := GetSection("APP").Key("DEBUG").String()
	if strings.ToLower(debug) == "true" {
		return true
	}
	return false
}

func CasAddress() string {
	return GetSection("APP").Key("CAS_ADDRESS").String()
}

func CasAppId() int {
	appId, _ := GetSection("APP").Key("CAS_APP_ID").Int()
	return appId
}

func RpcListenAddr() string {
	return GetSection("APP").Key("RPC_LISTEN_ADDR").String()
}

func HttpListenAddr() string {
	return GetSection("APP").Key("HTTP_LISTEN_ADDR").String()
}
