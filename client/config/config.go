package config

import (
	"gopkg.in/ini.v1"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"task/pkg/logger"
)

var config *ini.File

func init() {
	iniFile := "./client.ini"
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

func NodeAddr() string {
	return GetSection("APP").Key("NODE_ADDR").String()
}

func NodeName() string {
	return GetSection("APP").Key("NODE_NAME").String()
}

func HeartBeatInterval() uint {
	t, _ := GetSection("APP").Key("HEARTBEAT_INTERVAL").Uint()
	return t
}

func RpcListenAddr() string {
	return GetSection("APP").Key("RPC_LISTEN_ADDR").String()
}

func ManageListenAddr() string {
	return GetSection("APP").Key("MANAGE_LISTEN_ADDR").String()
}

func DaemonLogPath(ID uint, d string) string {
	return filepath.Join("runtime/log/daemon", d, strconv.Itoa(int(ID))+".log")
}

func GetLogger() *logger.ZapLogger {
	maxSize, _ := GetSection("ZAP").Key("MAX_SIZE").Int()
	maxBackups, _ := GetSection("ZAP").Key("MAX_BACKUPS").Int()
	maxAge, _ := GetSection("ZAP").Key("MAX_AGE").Int()
	localTime, _ := GetSection("ZAP").Key("LOCAL_TIME").Bool()
	compress, _ := GetSection("ZAP").Key("COMPRESS").Bool()
	return logger.NewZapLogger(&logger.ZapConfig{
		DebugFile:  GetSection("ZAP").Key("DEBUG_FILE").String(),
		InfoFile:   GetSection("ZAP").Key("INFO_FILE").String(),
		ErrorFile:  GetSection("ZAP").Key("ERROR_FILE").String(),
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		LocalTime:  localTime,
		Compress:   compress,
	})
}
