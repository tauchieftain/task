package model

import (
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"strconv"
	"time"
)

var dbConnections = make(map[string]*gorm.DB)

func InitDB(dbConfig map[string]map[string]string, options map[string]interface{}) {
	for k, v := range dbConfig {
		switch v["dialect"] {
		case "mysql":
			l, ok := options["logger"]
			if !ok {
				log.Fatalf("[%s]This dialect %s not config logger", k, v["dialect"])
			}
			dbConnections[k] = connectMysql(v, l.(logger.Interface))
			log.Println("Mysql connect success")
		case "sqlite":
			l, ok := options["logger"]
			if !ok {
				log.Fatalf("[%s]This dialect %s not config logger", k, v["dialect"])
			}
			dbConnections[k] = connectSqlite(v, l.(logger.Interface))
			log.Println("Sqlite connect success")
		default:
			log.Fatalf("[%s]This dialect %s not support", k, v["dialect"])
		}
	}
}

func connectMysql(conf map[string]string, l logger.Interface) *gorm.DB {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       conf["dsn"],
		SkipInitializeWithVersion: false,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		//NamingStrategy: schema.NamingStrategy{
		//	TablePrefix: conf["prefix"],
		//},
		FullSaveAssociations:                     false,
		Logger:                                   l,
		NowFunc:                                  time.Now().Local,
		DryRun:                                   false,
		PrepareStmt:                              true,
		DisableAutomaticPing:                     true,
		DisableForeignKeyConstraintWhenMigrating: true,
		DisableNestedTransaction:                 true,
		AllowGlobalUpdate:                        true,
		QueryFields:                              false,
		CreateBatchSize:                          1000,
	})
	if err != nil {
		log.Fatalf("Mysql connect failed, error: %s", err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Mysql pool init failed , error: %s", err.Error())
	}
	maxIdleConn, _ := strconv.Atoi(conf["max_idle_conn"])
	maxOpenConn, _ := strconv.Atoi(conf["max_open_conn"])
	sqlDB.SetMaxIdleConns(maxIdleConn)
	sqlDB.SetMaxOpenConns(maxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db
}

func connectSqlite(conf map[string]string, l logger.Interface) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(conf["dsn"]), &gorm.Config{
		SkipDefaultTransaction: true,
		//NamingStrategy: schema.NamingStrategy{
		//	TablePrefix: conf["prefix"],
		//},
		FullSaveAssociations:                     false,
		Logger:                                   l,
		NowFunc:                                  time.Now().Local,
		DryRun:                                   false,
		PrepareStmt:                              true,
		DisableAutomaticPing:                     true,
		DisableForeignKeyConstraintWhenMigrating: true,
		DisableNestedTransaction:                 true,
		AllowGlobalUpdate:                        true,
		QueryFields:                              false,
		CreateBatchSize:                          1000,
	})
	if err != nil {
		log.Fatalf("Sqlite connect failed, error: %s", err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Sqlite pool init failed , error: %s", err.Error())
	}
	maxIdleConn, _ := strconv.Atoi(conf["max_idle_conn"])
	maxOpenConn, _ := strconv.Atoi(conf["max_open_conn"])
	sqlDB.SetMaxIdleConns(maxIdleConn)
	sqlDB.SetMaxOpenConns(maxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db
}

func GetConnection(name string) *gorm.DB {
	connection, ok := dbConnections[name]
	if !ok {
		return nil
	}
	return connection
}

func Task() *gorm.DB {
	return GetConnection("task")
}
