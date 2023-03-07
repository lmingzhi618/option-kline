package common

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"time"
)

// 数据库相关配置
type DbConfig struct {
	Host        string
	Port        string
	UserName    string
	PassWord    string
	DbName      string
	Charset     string
	MaxCon      int
	MaxLifeTime int64
	IdleCon     int
	Debug       string
}

//将mysql的配置转换成dsn格式的字符串
func (dbconfig *DbConfig) ToDSN() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local", dbconfig.UserName, dbconfig.PassWord,
		dbconfig.Host, dbconfig.Port, dbconfig.DbName, dbconfig.Charset)
	return dsn
}

var (
	dbCon  *sql.DB //db read/write connect pool
	rdbCon *sql.DB //db read-only connect pool
)

//初始化数据库连接池
func initDbPool() {
	initWriteDbPool()
	initReadDbPool()
}

//init write/read db connect pool
func initWriteDbPool() {
	var err error
	dbCon, err = sql.Open("mysql", DBCONF.ToDSN())
	if err != nil || dbCon == nil {
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initWriteDbPool",
			"error":  err,
		}).Fatal("db driver or dsn config wrong")
	}
	//ping 一下db服务器判断是否可以正常连接
	if err = dbCon.Ping(); err != nil {
		dbCon.Close()
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initWriteDbPool",
			"error":  err,
		}).Fatal("db ping connect failed")
	}
	//mysql默认空闲超时断开时间为8hour，所以我设置为链接的时间小于8hour
	dbCon.SetConnMaxLifetime(4 * time.Hour)
	dbCon.SetMaxOpenConns(DBCONF.MaxCon)
	dbCon.SetMaxIdleConns(DBCONF.IdleCon)
}

//init read-only db connect pool
func initReadDbPool() {
	var err error
	rdbCon, err = sql.Open("mysql", RDBCONF.ToDSN())
	if err != nil || rdbCon == nil {
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initReadDbPool",
			"error":  err,
		}).Fatal("db driver or dsn config wrong")
	}
	//ping 一下db服务器判断是否可以正常连接
	if err = rdbCon.Ping(); err != nil {
		rdbCon.Close()
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initReadDbPool",
			"error":  err,
		}).Fatal("db ping connect failed")
	}
	//mysql默认空闲超时断开时间为8hour，所以我设置为链接的时间小于8hour
	rdbCon.SetConnMaxLifetime(4 * time.Hour)
	rdbCon.SetMaxOpenConns(RDBCONF.MaxCon)
	rdbCon.SetMaxIdleConns(RDBCONF.IdleCon)
}

//获取mysql(read/write)的连接
func GetDbConn() *sql.DB {
	if dbCon == nil {
		initWriteDbPool()
	}
	return dbCon
}

//获取mysql(read-only)的连接
func GetRdbConn() *sql.DB {
	if rdbCon == nil {
		initReadDbPool()
	}

	return rdbCon
}
