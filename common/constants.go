package common

import (
	"errors"
	"math/big"
)

// errors
var (
	ErrTimeInvalid  = errors.New("Time invalid")
	ErrLockAccount  = errors.New("Failed to lock account")
	ErrParamInvalid = errors.New("Param invalid")
	ErrServerError  = errors.New("Server error")
)

//运行环境变量
const (
	ENV_DEV        = "dev"        //开发环境
	ENV_TEST       = "test"       //测试环境
	ENV_ONLINE     = "online"     //生产环境
	ENV_PRE_ONLINE = "pre_online" //生产环境
)

//日期格式
const (
	LOGTIME_FORMAT  = "2006-01-02 15:04:05.000"
	DATETIME_FORMAT = "2006-01-02 15:04:05"
	DATE_FORMAT     = "2006-01-02"
	DAY_SECONDS     = 60 * 60 * 24
)

//订单类型
const (
	ORDER_TYPE_SELL = 2 //卖出类型
	ORDER_TYPE_BUY  = 1 //买入类型
)

// redis key
const (
	REDIS_KEY_PREFIX = "bc:exg" //redis缓存前缀
)

//分页设置
const (
	DEFAULT_PAGE_NO   = 1  //默认页号
	DEFAULT_PAGE_SIZE = 20 //默认页大小
)

// float64精确到小数点后10位的零食
const (
	FLOAT64_ZERO_VALUE        = 0
	FLOAT64_PRECISION_DEFAULT = 10 //float64保留小数点后十位
	FLOAT64_MAX_LENGTH        = 15 //浮点数字符最大长度
	INT64_MAX_LENGTH          = 12 //整形最大长度
)

var (
	BASE_SHIFT_BIGINT, _                    = big.NewInt(0).SetString("1000000000000000000", 10) //10^18
	ECOIN_EXCHANGE_SERVICE_FEE_PERCENT, _   = big.NewInt(0).SetString("1000000000000000", 10)    //平台撮合交易手续费 0.001 * 10^18
	ECOIN_EXCHANGE_INVITE_REWARD_PERCENT, _ = big.NewInt(0).SetString("100000000000000000", 10)  //用户推广奖励占总手续费比率 0.1 * 10^18
	BIG_INT_ZERO                            = big.NewInt(0)
)

const (
	FLOAT_STR_SHIFT_NUM = 18                    //float字符串字符串小数点向右偏移的位数
	FLOAT_ZEAR_VALUE    = "0.00000000000000000" // 0值
)

//代币的最小委托量
var (
	ETH_MIN_EXG_NUM, _ = big.NewInt(0).SetString("20000000000000000", 10)      //eth最小委托量
	BTC_MIN_EXG_NUM, _ = big.NewInt(0).SetString("1000000000000000", 10)       //btc最小委托量
	BCT_MIN_EXG_NUM, _ = big.NewInt(0).SetString("1000000000000000000000", 10) //bct最小委托量
)
var MIN_EXG_NUM_MAP = map[string]*big.Int{
	"ETH": ETH_MIN_EXG_NUM,
	"BTC": BTC_MIN_EXG_NUM,
	"BCT": BCT_MIN_EXG_NUM,
}
