package kline

import (
	"errors"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"math"
	"option-kline/common"
	//"sort"
	"strconv"
	"time"
)

type OptionOrder struct {
	Id          int64  `gorm:"column:id" json:"id"`
	UserId      string `gorm:"column:userId" json:"userId"`
	AgentId     string `gorm:"column:agentId" json:"agentId"`
	TokenType   string `gorm:"column:tokenType" json:"tokenType"`
	CoinType    string `gorm:"column:coinType" json:"coinType"`
	Type        string `gorm:"column:type" json:"type"` // 1:买涨 2:买跌
	Amount      string `gorm:"column:amount" json:"amount"`
	AgentAmount string `gorm:"column:agentAmount" json:"agentAmount"`
	IssueNumber string `gorm:"column:issueNumber" json:"issueNumber"`
	Status      int    `gorm:"column:status" json:"status"` // 订单状态，1：已开奖，0：未开奖
	OpenTime    int64  `gorm:"column:openTime" json:"openTime"`
	Result      int    `gorm:"column:result" json:"result"` // 1:赢， 2：输， 0：平
	CreateTime  int64  `gorm:"column:createTime" json:"createTime"`
	UpdateTime  int64  `gorm:"column:updateTime" json:"updateTime"`
	OpenPrice   string `gorm:"column:openPrice" json:"openPrice"`
	ClosePrice  string `gorm:"column:closePrice" json:"closePrice"`
	Profit      string `gorm:"column:profit" json:"profit"`
	Fee         string `gorm:"column:fee" json:"fee"`
	Revenue     string `gorm:"column:revenue" json:"revenue"`
}

type RevenuePrice struct {
	CoinType      string
	Price         string
	Magnification string
}

type RevenuePriceList []RevenuePrice

func (rp RevenuePriceList) Len() int {
	return len(rp)
}

func (rp RevenuePriceList) Swap(i, j int) {
	rp[i], rp[j] = rp[j], rp[i]
}

func (rp RevenuePriceList) Less(i, j int) bool {
	ret, err := common.BcCmp(rp[i].Magnification, rp[j].Magnification)
	if err != nil {
		return false
	}

	if ret == -1 {
		return true
	}
	return false
}

const (
	PriceUp   = "1"
	PriceDown = "2"
	PriceDraw = "3"

	// 用户下注结果
	OptionResultWin  = 1
	OptionResultLose = 2
	OptionResultDraw = 3

	// 期权订单状态
	OptionOrderOpened    = 1
	OptionOrderNotOpened = 0
)

// 获取当天倍率
// @param: openTime: 当期开奖时间
// @param: t: 当前K线时间
// @return: amountTotal:非平局下单总营收, feeTotal：非平局下单总手续费 mag: 已结算订单倍率
func GetMagnificationData(coinType string, tn int64) (string, string, float64, error) {
	fn := "GetMagnificationData"
	db := common.GetDbGormConn()
	revenueSettled, feeSettled := "0.0", "0.0"
	var mag float64
	t := time.Unix(tn, 0)
	timeBegin := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
	if err := db.Table("option_order").Select("COALESCE(SUM(revenue), 0), COALESCE(SUM(fee), 0), COALESCE(SUM(revenue)/SUM(fee), 0)").
		Where("coinType = ? AND openTime > ? AND status = ? AND result != ?", coinType, timeBegin, 1, 3).
		Row().Scan(&revenueSettled, &feeSettled, &mag); err != nil {
		log.Errorf("[%s]failed to query db: %s", fn, err)
		return "0.0", "0.0", 0.0, err
	}
	return revenueSettled, feeSettled, mag, nil
}

// 获取历史报价，包括订单下单时的报价
func GetKLineList(db *gorm.DB, kline *OptionKline) (klineList []*OptionKline) {
	fn := "GetKLineList"
	sqlStr := "select * from option_kline where coinType = ? order by id desc limit ?"
	if err := db.Raw(sqlStr, kline.CoinType, common.KlineSampleNum.Load().(int)).Scan(&klineList).Error; err != nil {
		log.Errorf("[%s]failed to query db: %s", fn, err)
	}
	return
}

// 根据当前报价，推算正负30个单位报价
// 计算各报价收益
// 取最接近目标倍率收益的报价
// @param: msg:倍率
// @return:
func GetDstPrice(kline *OptionKline, precision int) (float64, error) {
	fn := "GetDstPrice"
	defer common.CheckPanic(fn, nil)
	openTime := kline.Time - kline.Time%60 + 60
	orderList := []*OptionOrder{}
	db := common.GetDbGormConn()
	// 1. 获取当期所有订单数据
	if err := db.Table("option_order").Where("coinType = ? AND openTime = ?", kline.CoinType, openTime).
		Scan(&orderList).Error; err != nil {
		log.Errorf("[%s]failed to get order data: %s", fn, err)
		return -1, err
	}
	// 若无订单，则不干预报价
	if len(orderList) <= 0 {
		//log.Debugf("[%s]no orders for %s at openTime %d", fn, kline.CoinType, openTime)
		return 0, nil
	}

	// 2. 获取已结算订单的营收和手续费（注:平局用户不收手续费）
	revenueSettled, feeSettled, magSettled, err := GetMagnificationData(kline.CoinType, kline.Time)
	if err != nil {
		log.Errorf("[%s]failed to get magnification: %s", fn, err)
		return 0, err
	}
	// 若当天还无历史订单，则不干预报价
	if magSettled == 0 {
		return 0, err
	}

	// 3. 获取历史（1～２分钟）的报价:样本数据越多，接接近目标倍率,但报价震幅越大
	klineList := GetKLineList(db, kline)
	if len(klineList) <= 0 {
		log.Errorf("[%s]asshole! kline list shouldn't be empty!!!", fn)
		return 0, errors.New("kline list is empty")
	}

	// 4. 获取最接近差额倍率的报价
	dstPriceStr := GetOrderPrice(kline.Open, precision, klineList, orderList, revenueSettled, feeSettled, magSettled)
	dstPrice, err := strconv.ParseFloat(dstPriceStr, 64)
	if err != nil {
		log.Errorf("[%s]failed to parse float for %s: %s", fn, dstPriceStr, err)
		return 0, err
	}
	return dstPrice, nil
}

// 根据当前报价，推算正负30个单位报价
// @param: lastkline
// @param: precision: 当前货币精度
// @return: 正负30个单位的报价
func CalcPrices(lastKline *OptionKline, precision int) (klineList []*OptionKline) {
	klineList1 := CalcPricesBySign(lastKline, precision, 1.0)
	klineList2 := CalcPricesBySign(lastKline, precision, -1.0)
	klineList = append(klineList, klineList1...)
	klineList = append(klineList, klineList2...)
	return
}
func CalcPricesBySign(lastKline *OptionKline, precision int, sign float64) (klineList []*OptionKline) {
	fn := "CalcPricesBySign"
	unit := math.Pow(0.1, float64(precision))
	klinePrice, err := strconv.ParseFloat(lastKline.Open, 64)
	if err != nil {
		log.Printf("[%s]failed to parse float for %s", fn, lastKline.Open)
		return
	}
	for i := 1; i <= 30; i++ {
		dstPrice := klinePrice + unit*float64(i)*sign*100
		dstPriceStr := strconv.FormatFloat(dstPrice, 'f', precision, 64)
		kline := &OptionKline{
			CoinType: lastKline.CoinType,
			Open:     dstPriceStr,
			Close:    dstPriceStr,
		}
		klineList = append(klineList, kline)
	}
	return

}

// 获取最接近差额倍率的报价
// 1.获取每个报价的倍率
// 2.计算报价倍率与目标倍率差MD
// 3.MD的最小值对应的报价即为目标报价
// @param: klineList: 报价样本
// @param: orderKlineList: 订单报价
// @param: orderList: 未结算订单
// @param: revenueSettled: 已结算订单营收(不含平局)
// @param: feeSettled: 已结算订单手续费(平局用户不收手续费)
// @param: magSettled: 已结算订单倍率
// @return: 满足目标倍率的报价
func GetOrderPrice(curPriceStr string, precision int, klineList []*OptionKline, orderList []*OptionOrder, revenueSettled, feeSettled string, magSettled float64) string {
	fn := "GetOrderPrice"
	if len(klineList) <= 0 {
		log.Errorf("[%s]asshole! kline list shouldn't be empty!!!", fn)
		return ""
	}
	// 控制震幅
	//curPrice, err := strconv.ParseFloat(curPriceStr, 64)
	//if err != nil {
	//	log.Errorf("[%s]failed to ParseFloat(%s, 64)", curPriceStr)
	//	return ""
	//}
	//unit := float64(common.PriceAmplitude.Load().(int)*5) / math.Pow10(precision)
	//maxPrice, minPrice := curPrice+unit, curPrice-unit
	klineMap := make(map[int64]string, 60)
	for _, kline := range klineList {
		klineMap[kline.Time] = kline.Open
	}
	var minMag, okMag float64
	dstPrice := klineList[0].Open
	for idx, kline := range klineList {
		//klinePrice, err := strconv.ParseFloat(kline.Open, 64)
		//if err != nil {
		//	log.Errorf("[%s]failed to ParseFloat(%s, 64)", kline.Open)
		//	continue
		//}
		//if klinePrice > maxPrice || klinePrice < minPrice {
		//	continue
		//}
		revenueUnsettled, feeUnsettled, err := SumUnsettledOrders(kline, klineMap, orderList)
		// 0.排除平局的可能性
		if err != nil || feeUnsettled == "0.0" {
			continue
		}
		// 1. 计算所有订单的总营收和手续费
		revenueTotal, err := common.BcAdd(revenueSettled, revenueUnsettled, 18)
		if err != nil {
			log.Errorf("[%s]failed to BcDiv(%s,%s,18): %s", fn, revenueSettled, revenueUnsettled, err)
			continue
		}
		feeTotal, err := common.BcAdd(feeSettled, feeUnsettled, 18)
		if err != nil {
			log.Errorf("[%s]failed to BcDiv(%s,%s,18): %s", fn, feeSettled, feeUnsettled, err)
			continue
		}
		// 2. 求当前报价订单倍率
		magStr, err := common.BcDiv(revenueTotal, feeTotal, 18)
		if err != nil {
			log.Errorf("[%s]failed to BcDiv(%s,%s,18): %s", fn, revenueTotal, feeTotal, err)
			continue
		}
		priceMag, err := strconv.ParseFloat(magStr, 64)
		if err != nil {
			log.Errorf("[%s]failed to ParseFloat(%s, 64)", magStr)
			continue
		}
		// 2. 获取最接近目标倍率的报价
		absMag := math.Abs(common.OrderRate.Load().(float64) - priceMag)
		if idx == 0 || absMag < minMag {
			minMag = absMag
			okMag = priceMag
			dstPrice = kline.Open
		}
		log.Debugf("[%s][%d][%s:%d]%v, priceMag:%v, absMag:%v, revenue:%v, fee:%v",
			fn, idx, kline.CoinType, kline.Time, kline.Open, priceMag, absMag, revenueUnsettled, feeUnsettled)
	}
	log.Debugf("[%s]===revenueSettled:%v, feeSettled:%v, magSettled:%v", fn, revenueSettled, feeSettled, magSettled)
	log.Debugf("[%s]===ok mag:%v, ok dstPrice:[%s]%v", fn, okMag, klineList[0].CoinType, dstPrice)
	return dstPrice
}

// 根据给定报价，计算所有订单的总营收和手续费
// @param: kline: 指定的报价
// @param: orderList: 要计算的订单
// @return: amountTotal, feeTotal, err
func SumUnsettledOrders(kline *OptionKline, klineMap map[int64]string, orderList []*OptionOrder) (string, string, error) {
	fn := "SumUnsettledOrders"
	amountTotal, feeTotal := "0.0", "0.0"
	for _, order := range orderList {
		orderPrice, ok := klineMap[order.CreateTime]
		if !ok {
			orderPrice, ok = klineMap[order.CreateTime-1]
			if !ok {
				continue
			}
		}
		trend := PriceDraw
		if kline.Open > orderPrice {
			trend = PriceUp
		} else if kline.Open < orderPrice {
			trend = PriceDown
		}
		amount, err := common.BcSub(order.Amount, order.Fee, 18)
		if err != nil {
			log.Errorf("[%s]failed to BcSub(%s,%s,18): %s", fn, order.Amount, order.Fee, err)
			continue
		}
		// 1. 平:无赢收
		if trend == PriceDraw {
			order.Status = OptionResultDraw
			continue
		} else if trend != order.Type {
			// 2. 输：减去用户本金
			order.Status = OptionResultLose
			amountTotal, err = common.BcSub(amountTotal, amount, 18)
			if err != nil {
				log.Errorf("[%s]failed to BcSub(%s,%s,18): %s", fn, amountTotal, amount, err)
				continue
			}
		} else {
			// 3. 赢：加上代理商本金
			order.Status = OptionResultWin
			amountTotal, err = common.BcAdd(amountTotal, order.AgentAmount, 18)
			if err != nil {
				log.Errorf("[%s]failed to BcAdd(%s,%s,18): %s", fn, amountTotal, order.AgentAmount, err)
				continue
			}
		}
		feeTotal, err = common.BcAdd(feeTotal, order.Fee, 18)
		if err != nil {
			log.Errorf("[%s]failed to BcSub(%s,%s,18): %s", fn, feeTotal, order.Fee, err)
			continue
		}
		//	log.Debugf("[%s]order id:%v openPrice:%v, closePrice:%v, type:%v, status:%v", fn, order.Id, orderPrice, kline.Open, order.Type, order.Status)
	}
	return amountTotal, feeTotal, nil
}
