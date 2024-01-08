package config

var dbConfig map[string]map[string]string

func setDbConfig() {
	mysqlTask := GetSection("MYSQL_TASK")
	dbConfig = map[string]map[string]string{
		"task": {
			"dialect":       mysqlTask.Key("DIALECT").String(),
			"dsn":           mysqlTask.Key("DSN").String(),
			"prefix":        mysqlTask.Key("PREFIX").String(),
			"max_idle_conn": mysqlTask.Key("MAX_IDLE_CONN").String(),
			"max_open_conn": mysqlTask.Key("MAX_OPEN_CONN").String(),
		},
	}
}

func GetDbConfig() map[string]map[string]string {
	return dbConfig
}
