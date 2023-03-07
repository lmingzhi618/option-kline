package forex

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"net"
	"option-kline/common"
	"option-kline/kline"
	"sort"
	"strconv"
	"strings"
	"time"
)

//var addr = "103.229.144.153:3225"

type ForexData struct {
	Id        string `gorm:"column:id" json:"-"`
	CoinType  string `gorm:"column:coinType" json:"coinType"`
	Exchange  string `gorm:"column:exchange" json:"exchange"`
	SubMarket string `gorm:"column:subMarket" json:"subMarket"`
	Precision string `gorm:"column:precision" json:"precision"`
	Time      int64  `gorm:"column:time" json:"time"`
	NewPrice  string `gorm:"column:newPrice" json:"newPrice"`
	Buy       string `gorm:"column:buy" json:"buy"`
	Sell      string `gorm:"column:sell" json:"sell"`
	Open      string `gorm:"column:open" json:"open"`
	High      string `gorm:"column:high" json:"high"`
	Low       string `gorm:"column:low" json:"low"`
	Close     string `gorm:"column:close" json:"close"`
}

func (k ForexData) String() string {
	return strings.Join(
		[]string{
			"CoinType:", k.CoinType, "Exchange:", k.Exchange, "SubMarket:", k.SubMarket, "Precision:", k.Precision,
			"High:", k.High, "Low:", k.Low, "Open:", k.Open, "Close:", k.Close, "NewPrice:", k.NewPrice,
			"Time:", strconv.FormatInt(k.Time, 10),
		},
		" ",
	)
}

func (f ForexData) CsvString() []string {
	return []string{f.CoinType, f.Exchange, f.SubMarket, f.Precision, strconv.FormatInt(f.Time, 64), f.NewPrice,
		f.Buy, f.Sell, f.Open, f.Close, f.High, f.Low}
}

func FindMax(m map[string]int) string {
	maxKey := ""
	for k, v := range m {
		if maxKey == "" {
			maxKey = k
		}
		if v > m[maxKey] {
			maxKey = k
		}
	}
	return maxKey
}

type Pair struct {
	Key   string
	Value int
}

// A slice of Pairs that implements sort.Interface to sort by Value.
type PairList []Pair

func (p PairList) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p PairList) Len() int {
	return len(p)
}

func (p PairList) Less(i, j int) bool {
	return p[i].Value > p[j].Value
}

func sortMapByValue(m map[string]int) PairList {
	p := make(PairList, 1)
	for k, v := range m {
		p = append(p, Pair{k, v})
	}
	sort.Sort(p)
	return p
}

func GetForexData(klineChan chan *kline.OptionKline) {
	fn := "GetForexData"
	defer func() { go GetForexData(klineChan) }()
	defer common.CheckPanic(fn, nil)
	conn, err := net.DialTimeout("tcp", common.ForexAddr, 3*time.Second)
	if err != nil {
		log.Errorf("[%s] failed to connect tcp addr: %s, error: %s", fn, common.ForexAddr, err)
		return
	}
	buf := make([]byte, 1024)
	for {
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.Errorf("[%s]failed to set read deadline: %s", fn, err)
			return
		}
		if _, err = conn.Read(buf); err != nil {
			log.Errorf("[%s] failed to read tcp data: %s", fn, err)
			return
		}
		coinFlag := make(map[string]bool, 10)
		dataList := bytes.Split(buf, []byte{0x00})
		// 0xFF|69|6000||3||LTCUS|100|255|6|20181229|143135|32.000000|32.000000|32.000005|||||0x00
		for _, data := range dataList {
			if len(data) <= 0 || data[0] != 0xff {
				continue
			}
			fieldList := strings.Split(string(data), "|")
			// �, 60, 6000, , 3, , ETHUSD, 100, 29, 2, 20181229, 145034, 136.46, 136.46, 136.51, , , , ,
			if len(fieldList) < 20 || fieldList[2] != "6000" && fieldList[4] != "3" {
				continue
			}
			coinType, ok := kline.CoinTypeMap[fieldList[6]]
			if !ok || !common.IsInList(coinType, common.CoinSupported.Load().([]string)) {
				continue
			}
			// 对于需要的货币信息,只取一条
			if _, ok := coinFlag[coinType]; ok {
				continue
			}
			coinFlag[coinType] = true
			loc, _ := time.LoadLocation("Asia/Shanghai")
			tm, err := time.ParseInLocation("20060102150405", strings.Join(fieldList[10:12], ""), loc)
			if err != nil {
				log.Errorf("[%s]Failed to parse time in localtino: %s", fn, err)
				continue
			}
			kline := kline.OptionKline{
				CoinType:   coinType,
				Open:       fieldList[12],
				Close:      fieldList[12],
				High:       fieldList[12], // == newprice
				Low:        fieldList[12],
				LastUpdate: tm.Unix(),
				Time:       tm.Unix(),
				Origin:     1,
			}
			log.Debugf("[%s]received kline:%s", fn, kline)
			klineChan <- &kline
		}
	}
}

//		forexData := ForexData{
//			CoinType:  fieldList[6],
//			Exchange:  fieldList[7],
//			SubMarket: fieldList[8],
//			Precision: fieldList[9],
//			Time:      tm.Unix(),
//			NewPrice:  fieldList[12],
//			Buy:       fieldList[13],
//			Sell:      fieldList[14],
//			Open:      fieldList[12], //fieldList[15]
//			High:      fieldList[12],
//			Low:       fieldList[12],
//			Close:     fieldList[12],
//		}
