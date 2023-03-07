package kline

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"option-kline/common"
	"option-kline/regulator"
	//"sync/atomic"

	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	url         = "http://gold.fulbright.com.hk/fxtrader/xmlprice.asp"
	CoinTypeMap = map[string]string{
		"GOLD":   "GT",
		"XAUUSD": "GT",
		"USDCNH": "USDT",
		"BTCUSD": "BTC",
	}
	KLineDataMap     = make(map[string]*KLineData, 10)
	firstFlag        = false
	klineDstPriceMap = make(map[string]float64, 2)
)

type Trader struct {
	Contracts []*Contract `xml:"contract"`
}

type Contract struct {
	Name       string `xml:"name,attr" json:"coinType"`
	Bid        string `xml:"bid"`
	Ask        string `xml:"ask"`
	High       string `xml:"high" json:"high"`
	Low        string `xml:"low" json:"low"`
	Open       string `xml:"open" json:"open"`
	Close      string `xml:"close" json:"close"`
	RateDecpt  string `xml:"ratedecpt"`
	LastUpdate string `xml:"lastupdate" json:"lastupdate"`
}

func (c Contract) String() string {
	return fmt.Sprintf("Name: %s, Bid: %s, Ask: %s, High: %s, Low: %s, Open: %s, Close: %s, RateDecpt: %s, LastUpdate: %s",
		c.Name, c.Bid, c.Ask, c.High, c.Low, c.Open, c.Close, c.RateDecpt, c.LastUpdate)
}

func (c Contract) CsvString() []string {
	return []string{c.Name, c.Bid, c.Ask, c.High, c.Low, c.Open, c.Close, c.RateDecpt, c.LastUpdate}
}

// db table: option_kline
type OptionKline struct {
	//Id         string `gorm:"column:id" json:"-"`
	CoinType   string `gorm:"column:coinType" json:"coinType"`
	Open       string `gorm:"column:open" json:"open"`
	Close      string `gorm:"column:close" json:"close"`
	High       string `gorm:"column:high" json:"high"`
	Low        string `gorm:"column:low" json:"low"`
	Time       int64  `gorm:"column:time" json:"time"`
	LastUpdate int64  `gorm:"column:lastupdate" json:"lastupdate"`
	Origin     uint8  `gorm:"column:origin" json:"-"` // 是否为原始数据
}

func (k OptionKline) String() string {
	return strings.Join(
		[]string{
			"CoinType:", k.CoinType, "High:", k.High, "Low:", k.Low,
			"Open:", k.Open, "Close:", k.Close, "Time:", strconv.FormatInt(k.Time, 10),
			"LastUpdate:", strconv.FormatInt(k.LastUpdate, 10), "Origin:", strconv.Itoa(int(k.Origin)),
		},
		" ",
	)
}

type KLineData struct {
	Data     []*OptionKline
	CoinType string
	RWMutex  *sync.RWMutex
}

func NewKLineData(coinType string) *KLineData {
	return &KLineData{
		Data:     []*OptionKline{},
		CoinType: coinType,
		RWMutex:  new(sync.RWMutex),
	}
}

func (this *KLineData) CheckCapacity() {
	// 若缓存数据超出其容量,则删除旧数据
	for {
		n := len(this.Data)
		if n < common.CacheCapacity {
			break
		}
		this.Data = append(this.Data[:0], this.Data[(n-common.CacheCapacity):]...)
	}
}

func httpGet(url string, klineChan chan *OptionKline) {
	fn := "httpGet"
	log.Debugf("[%s]begin...", fn)
	transport := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 3 * time.Second,
		}).Dial,
	}
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: transport,
	}
	res, err := client.Get(url)
	if err != nil {
		log.Printf("[%s]http request error: %s\r\n", fn, err)
		return
	}
	defer res.Body.Close()
	data, _ := ioutil.ReadAll(res.Body)
	var trader Trader
	if err := xml.Unmarshal([]byte(data), &trader); err != nil {
		log.Errorf("[%s]Failed to unmarshal xml data: %s", fn, err)
		return
	}
	for idx, _ := range trader.Contracts {
		ctr := trader.Contracts[idx]
		coinType, ok := CoinTypeMap[ctr.Name]
		if !ok {
			continue
		}
		log.Debugf("[%s]coin_supported: %v", fn, common.CoinSupported.Load().([]string))
		if !common.IsInList(coinType, common.CoinSupported.Load().([]string)) {
			continue
		}

		loc, _ := time.LoadLocation("Asia/Shanghai")
		tn := time.Now()
		// ts := tn.Format("2006-01-02 15:04:05")
		ts := tn.In(loc).Format("2006-01-02 15:04:05")
		ts = strings.Split(ts, " ")[0]
		ts = strings.Join([]string{ts, ctr.LastUpdate}, " ")
		lastupdate, err := time.ParseInLocation("2006-01-02 15:04:05", ts, loc)
		if err != nil {
			log.Errorf("[%s]Failed to parse time in localtino: %s", fn, err)
			continue
		}

		kline := &OptionKline{
			CoinType:   coinType,
			High:       strings.Replace(ctr.High, ",", "", -1),
			Low:        strings.Replace(ctr.Low, ",", "", -1),
			Open:       strings.Replace(ctr.Open, ",", "", -1),
			Close:      strings.Replace(ctr.Close, ",", "", -1),
			LastUpdate: lastupdate.Unix(),
			Time:       tn.Unix(),
			Origin:     1,
		}
		log.Debugf("[%s]send kline to chan: %v", fn, kline)
		klineChan <- kline
		log.Debugf("[%s]====  success to send kline to chan", fn)
	}
}

// 获取K线数据，每秒获取两次: 该数据源已弃用
func GetKLineData(klineChan chan *OptionKline) {
	defer func() { go GetKLineData(klineChan) }()
	defer common.CheckPanic("GetKLineData", nil)
	for {
		httpGet(url, klineChan)
		time.Sleep(1000 * time.Millisecond)
	}
}

func SaveKLine2DB(kline *OptionKline) (err error) {
	db := common.GetDbGormConn()
	//if err := db.Where(OptionKline{CoinType: kline.CoinType, Time: kline.Time}).
	//	Assign(*kline).FirstOrCreate(kline).Error; err != nil {
	if err := db.Create(kline).Error; err != nil {
		return err
	}
	return nil
}

func SaveKLine2Csv(trader *Trader) {
	f, err := os.OpenFile("./data.csv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("%s", err)
	}
	w := csv.NewWriter(f)

	if firstFlag {
		firstFlag = false
		data := []string{"name", "bid", "ask", "high", "low", "open", "close", "ratedecpt", "lastupdate"}
		w.Write(data)
	}

	data := [][]string{}
	for idx, _ := range trader.Contracts {
		contract := trader.Contracts[idx]
		data = append(data, contract.CsvString())
	}

	w.WriteAll(data)
	w.Flush()
}

func DealAllHistoryPrice(klineChanMap map[string]chan *OptionKline) {
	fn := "DealAllHistoryPrice"
	defer common.CheckPanic(fn, nil)
	for _, coinType := range common.CoinSupported.Load().([]string) {
		if ch, ok := klineChanMap[coinType]; ok {
			DealHistoryPrice(ch, coinType)
		}
	}
}

func DealHistoryPrice(ch chan *OptionKline, coinType string) {
	fn := "DealHistoryPrice"
	klineData, ok := KLineDataMap[coinType]
	if !ok {
		log.Errorf("[%s]kline data for %s doesn't exist", fn, coinType)
		return
	}
	klineData.RWMutex.Lock()
	defer klineData.RWMutex.Unlock()
	klineList := &klineData.Data
	n := len(*klineList)
	// 0. 程序刚启动，无数据，则跳过
	if n == 0 {
		return
	}
	lastKline := (*klineList)[n-1]
	tn := time.Now().Unix()
	// 1. 若最新一条数据时间不晚于当前时间，则不用补数据
	if lastKline.Time >= tn {
		return
	}
	// 2. 若最新一条数据比当前时间晚1秒以上，则以最新一条数据为基础，制造并保存一条数据，并推送
	tmpKline := *lastKline
	tmpKline.Time = tn
	tmpKline.Origin = 0
	*klineList = append(*klineList, &tmpKline)
	dstKline := tmpKline

	AdjustPrice(&dstKline)
	log.Debugf("[%s][%s] before: %s after: %s, Time: %d",
		fn, tmpKline.CoinType, tmpKline.Open, dstKline.Open, tmpKline.Time)
	ch <- &dstKline
	n = len(*klineList)
	if n > common.CacheCapacity {
		*klineList = append((*klineList)[:0], (*klineList)[n-common.CacheCapacity:]...)
	}
}

func DealCurrentKLine(kline *OptionKline, dbChan chan *OptionKline) {
	fn := "DealCurrentKLine"
	defer common.CheckPanic(fn, nil)
	klineData, ok := KLineDataMap[kline.CoinType]
	if !ok {
		log.Errorf("[%s]kline data for %s doesn't exist", fn, kline.CoinType)
		return
	}
	klineData.RWMutex.Lock()
	defer klineData.RWMutex.Unlock()
	klineList := &klineData.Data
	n := len(*klineList)
	if n == 0 {
		*klineList = append(*klineList, kline)
		dbChan <- kline
		return
	}

	// 1. 以原始数据为基础，都要保存
	tn := time.Now().Unix()
	lastKline := (*klineList)[n-1]
	// 若当前时间已经有数据且已发布，则不处理
	if lastKline.Time >= tn {
		return
	}
	kline.Time = tn
	*klineList = append(*klineList, kline)
	dstKline := *kline
	// 2. 干预价格
	AdjustPrice(&dstKline)
	log.Debugf("[%s][%s] before: %s after: %s, Time: %d",
		fn, kline.CoinType, kline.Open, dstKline.Open, kline.Time)

	// 3. 保存新数据
	dbChan <- &dstKline
	n = len(*klineList)
	if n > common.CacheCapacity {
		*klineList = append((*klineList)[:0], (*klineList)[n-common.CacheCapacity:]...)
	}
}

func CheckPriceAmplitude(lastPrice, curPrice string) bool {
	lastOpenF, err := strconv.ParseFloat(lastPrice, 64)
	if err != nil {
		return false
	}
	curOpenF, err := strconv.ParseFloat(curPrice, 64)
	if err != nil {
		return false
	}
	if curOpenF/lastOpenF > 1.3 {
		return false
	}
	if curOpenF/lastOpenF < 0.7 {
		return false
	}
	return true
}

func AdjustPrice(kline *OptionKline) {
	fn := "AdjustPrice"
	kline.Origin = 0
	open, err := strconv.ParseFloat(kline.Open, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.Open)
		return
	}
	high, err := strconv.ParseFloat(kline.High, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.High)
		return
	}
	low, err := strconv.ParseFloat(kline.Low, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.Low)
		return
	}

	// 所有币种小数后添加1位
	precision := 5
	if kline.CoinType == "GT" {
		precision = 3 // 最少3位
	} else if kline.CoinType == "BTC" {
		precision = 3
	}

	reg, ok := regulator.RegulatorMap[kline.CoinType]
	if !ok {
		log.Errorf("[%s]coin type not supported:%s", fn, kline.CoinType)
		return
	}

	newOpen := reg.AdjustPrice(&regulator.AdJustRequest{
		CoinType:  kline.CoinType,
		Price:     open,
		Precision: precision,
	})
	kline.Open = strconv.FormatFloat(newOpen, 'f', precision, 64)
	if newOpen > high {
		kline.High = kline.Open
	}
	if newOpen < low {
		kline.Low = kline.Open
	}
	// 检查当前用户营收，并调整K线
	CheckRevenue(kline, precision)
}

func AdjustPrice1(kline *OptionKline) {
	fn := "AdjustPrice1"
	kline.Origin = 0
	// 若报价一直没有变化，则修改最后一位,随机涨跌
	open, err := strconv.ParseFloat(kline.Open, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.Open)
		return
	}
	closed, err := strconv.ParseFloat(kline.Close, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.Close)
		return
	}
	high, err := strconv.ParseFloat(kline.High, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.High)
		return
	}
	low, err := strconv.ParseFloat(kline.Low, 64)
	if err != nil {
		log.Errorf("[%s]Failed to parse float, err: %s, data: %s", fn, err, kline.Low)
		return
	}

	precision := 3
	denominator := float64(1000)
	if kline.CoinType == "USDT" {
		denominator = float64(100000)
		precision = 5
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rd := r.Intn(29)
	newOpen := open - float64(rd+1)/denominator
	if rd%2 != 0 {
		newOpen = open + float64(rd+1)/denominator
	}

	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	rd = r.Intn(33)
	newClose := closed - float64(rd+1)/denominator
	if rd%2 != 0 {
		newClose = closed + float64(rd+1)/denominator
	}

	log.Debugf("[%s]newOpen: %v", fn, newOpen)
	kline.Open = strconv.FormatFloat(newOpen, 'f', precision, 64)
	kline.Close = strconv.FormatFloat(newClose, 'f', precision, 64)
	if newOpen > high {
		kline.High = kline.Open
	}
	if newOpen < low {
		kline.Low = kline.Open
	}
	//kline.LastUpdate = kline.Time
}

// 检查当前用户营收，调整报价使营收与目标营收一致
func CheckRevenue(kline *OptionKline, precision int) {
	fn := "CheckRevenue"
	defer common.CheckPanic(fn, nil)
	tn := kline.Time
	t := tn - tn%60
	adjustTimeBegin := t + 40
	if kline.Time%60 == 0 {
		adjustTimeBegin = tn - 20
	}
	adjustTimeEnd := adjustTimeBegin + 20
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	var err error
	ts := time.Unix(kline.Time, 0).Second()
	// 1. 仅在用户停止下单期间调整价格
	if ts > 5 && ts < 40 /* && kline.Time < adjustTimeBegin) || kline.Time > adjustTimeEnd */ {
		delete(klineDstPriceMap, kline.CoinType)
		return
	}

	// 2. 若未计算过dstPrice，则计算并获取dstPrice
	dstPrice, ok := klineDstPriceMap[kline.CoinType]
	if !ok {
		dstPrice, err = GetDstPrice(kline, precision)
		if err != nil {
			log.Errorf("[%s]failed to GetDstPrice:%s", fn, err)
			return
		}
		// 若无订单，则不干预报价
		if dstPrice <= 0 {
			return
		}
		klineDstPriceMap[kline.CoinType] = dstPrice
	}

	// 3. 调整当前报价向dstPrice靠近
	oldPrice, err := strconv.ParseFloat(kline.Open, 64)
	if err != nil {
		log.Errorf("[%s]failed to parse float for %s: %s", fn, kline.Open, err)
		return
	}
	tpos := tn - adjustTimeBegin
	propotion := float64(tpos) / float64(adjustTimeEnd-adjustTimeBegin-2) //18.0
	if propotion > 1 || propotion < 0 {
		propotion = 1
	}
	diff := (dstPrice - oldPrice) * propotion
	okPrice := oldPrice + diff
	flag := 2 // 0: normal schedule, 1: use last price, 2: use adjusted dstPrice, don't change price value
	if tn < adjustTimeEnd-2 {
		// 使用上一条报价,使报价连续平滑
		if rd.Intn(100) < 30.0 {
			flag = 1
		}
	} else if tn < adjustTimeEnd /*&& rd.Intn(100) < 70.0 */ {
		// 3.1 提前在结算之前，到达目标报价(N% probability)
		okPrice = dstPrice
	} else if tn == adjustTimeEnd {
		// 3.2 结算时，报价为dstPrice
		okPrice = dstPrice
	} else if ts < 5 {
		// 3.3 结算后，随机用上一条报价
		if rd.Intn(100) < 60.0 {
			flag = 1
		}
	}
	if reg, ok := regulator.RegulatorMap[kline.CoinType]; ok {
		okPrice = reg.AdjustPrice(&regulator.AdJustRequest{
			CoinType:  kline.CoinType,
			Price:     okPrice,
			Precision: precision,
			Flag:      flag,
		})
	}
	kline.Open = strconv.FormatFloat(okPrice, 'f', precision, 64)
	kline.Close = kline.Open
	log.Debugf("[%s][%s]origin price: %v, adjusted price: %s, dstPrice: %v, klineTime: %v", fn, kline.CoinType, oldPrice, kline.Open, dstPrice, kline.Time)
}

func InitKLine() {
	for _, coinType := range common.CoinSupported.Load().([]string) {
		KLineDataMap[coinType] = NewKLineData(coinType)
	}
}
