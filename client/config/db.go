package config

var dbConfig map[string]map[string]string

func setDbConfig() {
	sqliteTask := GetSection("SQLITE_TASK")
	dbConfig = map[string]map[string]string{
		"task": {
			"dialect":       sqliteTask.Key("DIALECT").String(),
			"dsn":           sqliteTask.Key("DSN").String(),
			"prefix":        sqliteTask.Key("PREFIX").String(),
			"max_idle_conn": sqliteTask.Key("MAX_IDLE_CONN").String(),
			"max_open_conn": sqliteTask.Key("MAX_OPEN_CONN").String(),
		},
	}
}

func GetDbConfig() map[string]map[string]string {
	return dbConfig
}
