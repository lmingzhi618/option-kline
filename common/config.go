package common

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widuu/goini"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type OptionSetting struct {
	Type       string `gorm:"column:type" json:"type"`
	KeyName    string `gorm:"column:keyName" json:"keyName"`
	Value      string `gorm:"column:value" json:"value"`
	Remarks    string `gorm:"column:remarks" json:"remarks"`
	UpdateTime int64  `gorm:"column:updateTime" json:"updateTime"`
}

//exgservice 相关配置
type ExgServiceConfig struct {
	GrpcServerHost string
	MaxConn        int
	InitConn       int
}

//system
var (
	APPNAME        string  //应用名称
	CURMODE        string  //当前系统的运行环境(dev/test/online)
	DATAENCRPTYKEY string  //数据对称加密秘钥
	LISTENPORT     string  //grpc 服务监听端口
	GINLISTENPORT  string  //gin 服务监听端口
	WHITE_UIDS     []int64 //用户id API调用频率限速白名单
	CacheCapacity  int     //K线缓存数量
)

//db
var (
	DBCONF  *DbConfig //db 相关配置
	RDBCONF *DbConfig //db 读库配置
)

//cache
var (
	REDISCONF *RedisConfig //redis 相关配置
)

//logger
var (
	LOGDIR       string //日志目录
	LOGFILENAME  string //日志文件名
	LOGKEEPDAYS  int64  //日志文件留存时间，单位：天
	LOGRATEHOURS int64  //日志文件切割时间间隔，单位：小时
	DEBUG        string //调试标志
)

// web
var (
	AllowOrigins = []string{}
)

// rabbitmq
var (
	RabbitMqUrl        = "amqp://guest:guest@localhost:5672"
	PushExchange       = "gateway-ws"
	PushRoutineKeyList = []string{"push_option"}
)

// kline
var (
	ForexAddr = "127.0.0.1:2000"

	PricePointsRange = 50
	SelectRange      = 100
	SelectStep       = 50

	//CoinSupported = []string{"GT", "USDT"}
	//KlineSampleNum   = 200
	//PriceAmplitude   = 100
	//OrderRate        = 5.0
	CoinSupported  atomic.Value // 支持货币对
	KlineSampleNum atomic.Value // 报价样本数量
	PriceAmplitude atomic.Value // 震幅
	OrderRate      atomic.Value // 倍率
)

func init() {
	initMode()
	loadConfig()
	initGormDbPool()
	go func() {
		for {
			LoadConfigFromDB()
			time.Sleep(5 * time.Second)
		}
	}()
}

func GetPwd() string {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "./"
	}
	path, err := filepath.Abs(file)
	if err != nil {
		return "./"
	}
	return filepath.Dir(path)
}

// 加载配置文件
func loadConfig() {

	var fileName = fmt.Sprintf("%s/conf/%s.ini", GetPwd(), CURMODE)
	_, err := os.Stat(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.WithFields(log.Fields{
				"app":    APPNAME,
				"action": "loadConfig",
				"error":  err,
			}).Fatal("configuration file is not exist!")
		} else {
			log.WithFields(log.Fields{
				"app":    APPNAME,
				"action": "loadConfig",
				"error":  err,
			}).Fatal("configuration file is privilge mode is not right!")
		}
	}
	conf := goini.SetConfig(fileName)

	APPNAME = conf.GetValue("service", "app_name")
	LISTENPORT = conf.GetValue("service", "listen_port")
	GINLISTENPORT = conf.GetValue("service", "gin_listen_port")
	DATAENCRPTYKEY = conf.GetValue("service", "data_encrpty_key")
	CacheCapacity, err = strconv.Atoi(conf.GetValue("service", "cache_capacity"))
	if CacheCapacity <= 0 || err != nil {
		CacheCapacity = 100000
	}

	whiteUidsStr := conf.GetValue("service", "white_uids")
	WHITE_UIDS = func(idsStr string) []int64 {
		ret := make([]int64, 0, 5)
		idStrArr := strings.Split(idsStr, ",")
		for _, idStr := range idStrArr {
			int64Id, _ := strconv.ParseInt(idStr, 10, 64)
			ret = append(ret, int64Id)
		}
		return ret
	}(whiteUidsStr)

	LOGDIR = conf.GetValue("log", "dir")
	LOGFILENAME = conf.GetValue("log", "file_name")
	LOGKEEPDAYS, _ = strconv.ParseInt(conf.GetValue("log", "keep_days"), 10, 64)
	LOGRATEHOURS, _ = strconv.ParseInt(conf.GetValue("log", "rotate_period"), 10, 64)
	DEBUG = conf.GetValue("log", "debug")
	//redis 配置加载
	REDISCONF = &RedisConfig{
		Host:   conf.GetValue("redis", "host"),
		Port:   conf.GetValue("redis", "port"),
		Auth:   conf.GetValue("redis", "auth"),
		DbName: conf.GetValue("redis", "db_name"),
	}
	REDISCONF.MaxIdle, _ = strconv.ParseInt(conf.GetValue("redis", "max_idle"), 10, 64)
	REDISCONF.MaxActive, _ = strconv.ParseInt(conf.GetValue("redis", "max_active"), 10, 64)
	REDISCONF.IdleTimeout, _ = strconv.ParseInt(conf.GetValue("redis", "idle_timeout"), 10, 64)
	REDISCONF.Wait, _ = strconv.ParseBool(conf.GetValue("redis", "wait"))
	REDISCONF.ConnTimeout, _ = strconv.ParseInt(conf.GetValue("redis", "conn_timeout"), 10, 64)
	REDISCONF.WriteTimeout, _ = strconv.ParseInt(conf.GetValue("redis", "write_timeout"), 10, 64)
	REDISCONF.ReadTimeout, _ = strconv.ParseInt(conf.GetValue("redis", "read_timeout"), 10, 64)
	//mysql配置读写库加载
	DBCONF = &DbConfig{
		Host:     conf.GetValue("db", "host"),
		Port:     conf.GetValue("db", "port"),
		DbName:   conf.GetValue("db", "dbname"),
		UserName: conf.GetValue("db", "user_name"),
		PassWord: conf.GetValue("db", "password"),
		Charset:  conf.GetValue("db", "charset"),
	}
	DBCONF.MaxCon, _ = strconv.Atoi(conf.GetValue("db", "maxConn"))
	DBCONF.IdleCon, _ = strconv.Atoi(conf.GetValue("db", "idleConn"))
	DBCONF.MaxLifeTime, _ = strconv.ParseInt(conf.GetValue("db", "maxLifeTime"), 10, 64)
	//mysql配置读库配置
	RDBCONF = &DbConfig{
		Host:     conf.GetValue("rdb", "host"),
		Port:     conf.GetValue("rdb", "port"),
		DbName:   conf.GetValue("rdb", "dbname"),
		UserName: conf.GetValue("rdb", "user_name"),
		PassWord: conf.GetValue("rdb", "password"),
		Charset:  conf.GetValue("rdb", "charset"),
	}
	RDBCONF.MaxCon, _ = strconv.Atoi(conf.GetValue("rdb", "maxConn"))
	RDBCONF.IdleCon, _ = strconv.Atoi(conf.GetValue("rdb", "idleConn"))
	RDBCONF.MaxLifeTime, _ = strconv.ParseInt(conf.GetValue("rdb", "maxLifeTime"), 10, 64)

	if val := conf.GetValue("rabbitmq", "RabbitMqUrl"); val != "" {
		RabbitMqUrl = val
	}
	if val := conf.GetValue("rabbitmq", "PushExchange"); val != "" {
		PushExchange = val
	}
	if val := conf.GetValue("rabbitmq", "PushRoutineKeyList"); val != "" {
		val = strings.Replace(val, " ", "", -1)
		valList := strings.Split(val, ",")
		if len(valList) > 0 {
			PushRoutineKeyList = valList
		}
	}
	fmt.Println("RabbitMqUrl: ", RabbitMqUrl)
	fmt.Println("PushExchange: ", PushExchange)
	fmt.Println("PushRoutineKeyList: ", PushRoutineKeyList)

	// kline
	if val := conf.GetValue("kline", "coin_supported"); val != "" {
		val = strings.Replace(val, " ", "", -1)
		valList := strings.Split(val, ",")
		if len(valList) > 0 {
			CoinSupported.Store(valList)
		} else {
			CoinSupported.Store([]string{"GT", "USDT", "BTC"})
		}
	}
	if val := conf.GetValue("kline", "forex_addr"); val != "" {
		ForexAddr = val
	}
	if val := conf.GetValue("kline", "order_rate"); val != "" {
		orderRate, err := strconv.ParseFloat(val, 64)
		if err != nil {
			orderRate = 5
		}
		OrderRate.Store(orderRate)
	}
	if val := conf.GetValue("kline", "price_point_range"); val != "" {
		PricePointsRange, err = strconv.Atoi(val)
		if err != nil {
			PricePointsRange = 50
		}
	}
	if val := conf.GetValue("kline", "price_amplitude"); val != "" {
		amplitude, err := strconv.Atoi(val)
		if err != nil {
			amplitude = 100
		}
		PriceAmplitude.Store(amplitude)
	}
	if val := conf.GetValue("kline", "kline_sample_number"); val != "" {
		sampleNum, err := strconv.Atoi(val)
		if err != nil {
			sampleNum = 200
		}
		KlineSampleNum.Store(sampleNum)
	}
	if val := conf.GetValue("kline", "select_range"); val != "" {
		SelectRange, err = strconv.Atoi(val)
		if err != nil {
			SelectRange = 100
		}
	}
	if val := conf.GetValue("kline", "select_step"); val != "" {
		SelectStep, err = strconv.Atoi(val)
		if err != nil {
			SelectStep = 10
		}
	}
}

// 初始换运行环境
func initMode() {
	exchangeMode := os.Getenv("SERVERMODE")
	if exchangeMode == "" {
		log.WithFields(log.Fields{
			"app":    APPNAME,
			"action": "initMode",
			"error":  "env variable SERVERMODE is not set",
		}).Fatal("environment variable SERVERMODE is not set!")
	}
	for _, mode := range []string{ENV_DEV, ENV_TEST, ENV_ONLINE, ENV_PRE_ONLINE} {
		if mode == exchangeMode {
			CURMODE = mode
			return
		}
	}
	log.WithFields(log.Fields{
		"app":    APPNAME,
		"action": "initMode",
		"error":  "env variable SERVERMODE is invalid",
	}).Fatal("environment variable SERVERMODE is invalid!")
}

func LoadConfigFromDB() {
	fn := "LoadConfigFromDB"
	settings := []*OptionSetting{}
	db := GetRDbGormConn()
	if err := db.Find(&settings).Error; err != nil {
		log.Errorf("[%s]failed to query db: %s", fn, err)
		return
	}

	for _, s := range settings {
		switch s.KeyName {
		case "coin_supported":
			val := strings.Replace(s.Value, " ", "", -1)
			valList := strings.Split(val, ",")
			if len(valList) > 0 {
				CoinSupported.Store(valList)
			}
			log.Debugf("[%s]coin supported: %v", fn, CoinSupported)
		case "kline_sample_number":
			if sampleNum, err := strconv.Atoi(s.Value); err == nil && sampleNum >= 60 {
				KlineSampleNum.Store(sampleNum)
			}
			log.Debugf("[%s]kline sample number: %v", fn, KlineSampleNum)
		case "price_amplitude":
			if amplitude, err := strconv.Atoi(s.Value); err == nil {
				PriceAmplitude.Store(amplitude)
			}
			log.Debugf("[%s]price amplitude: %v", fn, PriceAmplitude)
		case "order_rate":
			if orderRate, err := strconv.ParseFloat(s.Value, 64); err == nil {
				OrderRate.Store(orderRate)
			}
			log.Debugf("[%s]order rate: %v", fn, OrderRate)
		default:
		}
	}
}
