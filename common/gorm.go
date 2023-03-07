package common

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	log "github.com/sirupsen/logrus"
	"time"
)

var (
	dbGorm  *gorm.DB //db read/write connect pool
	rdbGorm *gorm.DB //db read-only connect pool
)

//初始化数据库连接池
func initGormDbPool() {
	initWriteGromDbPool()
	initReadGormDbPool()
}

//init write/read db connect pool
func initWriteGromDbPool() {
	var err error
	dbGorm, err = gorm.Open("mysql", DBCONF.ToDSN())
	if err != nil || dbGorm == nil {
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initWriteGromDbPool",
			"error":  err,
		}).Fatal("[initWriteGromDbPool]db driver or dsn config wrong")
	}
	if CURMODE == ENV_DEV {
		dbGorm.LogMode(true)
	}
	dbGorm.SingularTable(true)
	//mygorm默认空闲超时断开时间为8hour，所以我设置为链接的时间小于8hour
	dbGorm.DB().SetConnMaxLifetime(time.Duration(DBCONF.MaxLifeTime) * time.Hour)
	dbGorm.DB().SetMaxOpenConns(DBCONF.MaxCon)
	dbGorm.DB().SetMaxIdleConns(DBCONF.IdleCon)
}

//init read-only db connect pool
func initReadGormDbPool() {
	var err error
	rdbGorm, err = gorm.Open("mysql", RDBCONF.ToDSN())
	if err != nil || rdbGorm == nil {
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initReadGormDbPool",
			"error":  err,
		}).Fatal("[initReadGormDbPool]db driver or dsn config wrong")
	}
	if CURMODE == ENV_DEV {
		rdbGorm.LogMode(true)
	}
	rdbGorm.SingularTable(true)
	//mygorm默认空闲超时断开时间为8hour，所以我设置为链接的时间小于8hour
	rdbGorm.DB().SetConnMaxLifetime(time.Duration(DBCONF.MaxLifeTime) * time.Hour)
	rdbGorm.DB().SetMaxOpenConns(RDBCONF.MaxCon)
	rdbGorm.DB().SetMaxIdleConns(RDBCONF.IdleCon)
}

//获取mygorm(write)的连接
func GetDbGormConn() *gorm.DB {
	if dbGorm == nil {
		initWriteGromDbPool()
	}
	return dbGorm
}

//获取mygorm(read-only)的连接
func GetRDbGormConn() *gorm.DB {
	if rdbGorm == nil {
		initReadGormDbPool()
	}

	return rdbGorm
}
