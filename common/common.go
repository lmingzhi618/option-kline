package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

//将int64数组按照分隔符拼接成字符串
func Int64Explode(arr []int64, division string) string {
	var res string = ""
	arrLen := len(arr)
	if arrLen > 0 {
		for index := 0; index < arrLen; index++ {
			res += strconv.Itoa(int(arr[index]))
			if index < arrLen-1 {
				res += division
			}
		}
	}
	return res
}

//将整形数组按照指定的分隔符拼接成字符串
func IntExplode(arr []int, division string) string {
	var res string = ""
	arrLen := len(arr)
	if arrLen > 0 {
		for index := 0; index < arrLen; index++ {
			res += strconv.Itoa(arr[index])
			if index < arrLen-1 {
				res += division
			}
		}
	}
	return res
}

//结构体对象转成json 字符串
func StructToJsonStr(structObj interface{}) (str string, err error) {
	if structObj == nil {
		err = errors.New("func=[StructToJsonStr] struct obj is nil")
		return
	}

	byteArr, err := json.Marshal(structObj)
	if err != nil {
		return
	}
	str = string(byteArr)
	return
}

//取多个整数中最大的一个
func FindMaxNumInt64(nums ...int64) (max int64, err error) {
	if len(nums) <= 0 {
		err = errors.New("参数个数不能为空")
		return
	}
	max = nums[0]
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return
}

//取多个整数中最小的一个
func FindMinNumInt64(nums ...int64) (min int64, err error) {
	if len(nums) <= 0 {
		err = errors.New("参数个数不能为空")
		return
	}
	min = nums[0]
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return
}

//取多个浮点数中最大的一个
func FindMaxNumFloat64(nums ...float64) (max float64, err error) {
	if len(nums) <= 0 {
		err = errors.New("参数个数不能为空")
		return
	}
	max = nums[0]
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return
}

//取多个浮点数中最小的一个
func FindMinNumFloat64(nums ...float64) (min float64, err error) {
	if len(nums) <= 0 {
		err = errors.New("参数个数不能为空")
		return
	}
	min = nums[0]
	for _, num := range nums {
		if num < min {
			min = num
		}
	}
	return
}

//判断一个元素是否在slice,数组，map中
func Contain(target interface{}, obj interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}

	return false
}

//将浮点数原封不动的转换成可视化的字符串
func Float64ToString(val float64) string {
	//float64小数点后面最多保留15位小数
	parseStr := fmt.Sprintf("%.10f", val)
	return strings.TrimRight(strings.TrimRight(parseStr, "0"), ".")
}

//将64位浮点数进行精度换算，最多保留小数点后{precision} 位
func Float64PrecisionDeal(val float64, precision int) float64 {
	var precisionFormat = "%." + strconv.Itoa(precision) + "f"
	valStr := fmt.Sprintf(precisionFormat, val)
	valStr = strings.TrimRight(strings.TrimRight(valStr, "0"), ".")
	res, _ := strconv.ParseFloat(valStr, 10)
	return res
}

//获取本机的ip地址
//return : ipv4的地址字符串
func GetLocalOrProxyAddr() (ipstr string) {
	ipCheckUrl := "http://ip.cn"
	resp, err := http.Get(ipCheckUrl)
	defer resp.Body.Close()
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	regx, err := regexp.Compile(`((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)`)
	if err != nil {
		return
	}
	return string(regx.Find(body))
}

// Http Get请求基础函数
// strUrl: 请求的URL
// strParams: string类型的请求参数, user=lxz&pwd=lxz
// headArr : http头相关参数数组
// transport : http代理访问配置
// return: 请求结果
func HttpGetRequest(strUrl string, mapParams map[string]string, headArr map[string]string, transport *http.Transport) []byte {
	defer func() {
		if err := recover(); err != nil {
			log.Error("funcName=[HttpGetRequest], err=", err)
		}
	}()
	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 10000
	httpClient := &http.Client{}
	if transport != nil {
		httpClient.Transport = transport
	}
	var strRequestUrl string
	if nil == mapParams {
		strRequestUrl = strUrl
	} else {
		strParams := Map2UrlQuery(mapParams)
		strRequestUrl = strUrl + "?" + strParams
	}
	// 构建Request, 并且按官方要求添加Http Header
	request, err := http.NewRequest(http.MethodGet, strRequestUrl, nil)
	if nil != err {
		return []byte(err.Error())
	}

	if headArr != nil && len(headArr) > 0 {
		for k, v := range headArr {
			request.Header.Add(k, v)
		}
	}

	// 发出请求
	response, err := httpClient.Do(request)
	if nil != err {
		return []byte(err.Error())
	}
	defer response.Body.Close()
	// 解析响应内容
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return []byte(err.Error())
	}
	io.Copy(ioutil.Discard, response.Body)

	return body
}

// Http POST请求基础函数
// strUrl: 请求的URL
// mapParams: map类型的请求参数
// headArr : http头相关参数数组
// transport : http代理访问配置
// return: 请求结果
func HttpPostRequest(strUrl string, mapParams map[string]string, headArr map[string]string, transport *http.Transport) []byte {
	httpClient := http.DefaultClient

	values := url.Values{}
	for k, v := range mapParams {
		values.Add(k, v)
	}
	request, err := http.NewRequest(http.MethodPost, strUrl, strings.NewReader(values.Encode()))
	if nil != err {
		return []byte(err.Error())
	}

	if headArr != nil && len(headArr) > 0 {
		for k, v := range headArr {
			request.Header.Add(k, v)
		}
	}

	response, err := httpClient.Do(request)
	if nil != err {
		return []byte(err.Error())
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if nil != err {
		return []byte(err.Error())
	}

	return body
}

// 将map格式的请求参数转换为字符串格式的
// mapParams: map格式的参数键值对
// return: 查询字符串
func Map2UrlQuery(mapParams map[string]string) string {
	var strParams string
	for key, value := range mapParams {
		strParams += (key + "=" + value + "&")
	}

	if 0 < len(strParams) {
		strParams = string([]rune(strParams)[:len(strParams)-1])
	}

	return strParams
}

//将整形字符串转换成小数点向左偏移18位的浮点字符串
func IntStrToFloatStr(intStr string) (floatStr string) {
	tempZeroStr := "000000000000000000" //18位的0串
	var buffer bytes.Buffer
	if intStr[0:1] == "-" {
		intStr = intStr[1:]
		buffer.WriteByte('-')
	}
	intStrLen := len(intStr)
	if intStrLen > FLOAT_STR_SHIFT_NUM {
		intpartLen := intStrLen - FLOAT_STR_SHIFT_NUM
		buffer.WriteString(intStr[0:intpartLen])
		buffer.WriteByte('.')
		buffer.WriteString(intStr[intpartLen:])
	} else {
		buffer.WriteString("0.")
		fillZeroNum := FLOAT_STR_SHIFT_NUM - intStrLen
		if fillZeroNum > 0 {
			buffer.WriteString(tempZeroStr[0:fillZeroNum])
		}
		buffer.WriteString(intStr)
	}

	floatStr = buffer.String()
	return
}

//将整形字符串转换成其整数值除以10^18得到的浮点数或者整形的原生字符串
//eg: 1800000000000000000 =>1.8000000000000000000	=> 1.8
func IntStrToFloatOriginStr(intStr string) (floatStr string) {
	floatStr = IntStrToFloatStr(intStr)
	var i int
	for i = len(floatStr) - 1; i > 0; i-- {
		if floatStr[i] != '0' {
			if floatStr[i] != '.' {
				i++
			}
			break
		}
	}
	return floatStr[:i]
}

func IntStrToFormatPrecision(intStr string, pricisionNum int) string {
	floatStr := IntStrToFloatStr(intStr)
	domainIndex := strings.Index(floatStr, ".")
	floatStr = floatStr[:domainIndex+pricisionNum+1]
	return strings.TrimRight(strings.TrimRight(floatStr, "0"), ".")
}

//将浮点字符串转换成小数点向右偏移18位的整数字符串
func FloatStrToIntStr(floatStr string) (intStr string) {
	intStr = "0"
	tempFloatVal, _ := big.NewFloat(0).SetString(floatStr)
	if tempFloatVal.Cmp(big.NewFloat(0)) == 0 {
		return
	}
	domainIndex := strings.Index(floatStr, ".")
	tempZeroStr := "000000000000000000" //18位的0串
	var buffer strings.Builder
	if domainIndex <= 0 {
		buffer.WriteString(floatStr)
		buffer.WriteString(tempZeroStr)
	} else {
		buffer.WriteString(floatStr[:domainIndex])
		decimalPartLen := len(floatStr) - (domainIndex + 1)
		buffer.WriteString(floatStr[domainIndex+1:])
		buffer.WriteString(tempZeroStr[0:(18 - decimalPartLen)])
	}
	intStr = buffer.String()
	if intStr[0:1] == "-" {
		intStr = fmt.Sprintf("-%s", strings.TrimLeft(intStr[1:], "0"))
	} else {
		intStr = strings.TrimLeft(buffer.String(), "0")
	}
	return
}

//针对参数浮点字符串保留指定位数的小数位
//floatStr: 浮点字符串， decimalLen ：保留的小数点位数
func FloatStrRound(floatStr string, decimalLen int) (floatRes string) {
	tempZeroStr := "000000000000000000" //18位的0串
	if len(floatStr) == 0 {
		return
	}
	var buffer bytes.Buffer
	domainIndex := strings.Index(floatStr, ".")

	switch {
	case domainIndex < 0:
		buffer.WriteString(floatStr)
		if decimalLen > 0 {
			buffer.WriteByte('.')
			buffer.WriteString(tempZeroStr[0:decimalLen])
		}
		floatRes = buffer.String()
	case domainIndex == 0:
		floatRes = floatStr
	case domainIndex > 0:
		buffer.WriteString(floatStr[0:domainIndex])
		if decimalLen > 0 {
			buffer.WriteByte('.')
			domainRightLen := len(floatStr[domainIndex+1:])
			if domainRightLen >= decimalLen {
				buffer.WriteString(floatStr[domainIndex+1 : domainIndex+1+decimalLen])
			} else {
				buffer.WriteString(floatStr[domainIndex+1:])
				buffer.WriteString(tempZeroStr[0 : decimalLen-domainRightLen])
			}
		}
		floatRes = buffer.String()
	}
	return
}

//返回两个大整形字符串中较小的那个
func ReturnMinBigInt(leftIntStr, rightIntStr string) (retBigInt *big.Int) {
	tempLeftBigInt, _ := big.NewInt(0).SetString(leftIntStr, 10)
	tempRightBigInt, _ := big.NewInt(0).SetString(rightIntStr, 10)

	retBigInt = tempLeftBigInt
	if retBigInt.Cmp(tempRightBigInt) > 0 {
		retBigInt = tempRightBigInt
	}

	return
}

func GetCurrentDirectory() (ret string) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func CheckPanic(funcName string, err *error) {
	if r := recover(); r != nil {
		if err != nil {
			*err = fmt.Errorf("server error")
		}
		log.Errorf("[%s]panic: %s", funcName, r)
	}
}

func IsInList(target string, list []string) bool {
	if target == "" {
		return false
	}
	for idx, _ := range list {
		if target == list[idx] {
			return true
		}
	}
	return false
}

func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	return conn.LocalAddr().String()
}

// recive quit signal
func QuitSignal() <-chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	return signals
}

func IsStrBigger(str1, str2 string) bool {
	if str1 == str2 {
		return false
	}
	f1, err := strconv.ParseFloat(str1, 64)
	if err != nil {
		return false
	}
	f2, err := strconv.ParseFloat(str2, 64)
	if err != nil {
		return false
	}
	if f1 <= f2 {
		return false
	}
	return true
}
