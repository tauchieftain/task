package config

import "task/pkg/logger"

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
