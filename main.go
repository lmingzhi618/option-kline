package main

import (
	"encoding/json"
	//"github.com/panjf2000/ants"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"net/http"
	_ "net/http/pprof"
	"option-kline/common"
	"option-kline/forex"
	"option-kline/kline"
	"time"
)

type RabbitMqMsg struct {
	AppId     string `json:"appId"`
	EventType string `json:"eventType"`
	Body      string `json:"body"`
}

type MqBody struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// channel
var (
	klineChan    = make(chan *kline.OptionKline, 100)
	dbChan       = make(chan *kline.OptionKline, 100)
	mqChanMap    = make(map[string]chan *kline.OptionKline, 10)
	klineChanMap = make(map[string]chan *kline.OptionKline, 10)
)

func PublishMqMsg(mqChan chan *kline.OptionKline, routingKey string) {
	fn := "PublishMqMsg"
	defer func() { go PublishMqMsg(mqChan, routingKey) }()
	defer time.Sleep(time.Second)
	conn, err := amqp.Dial(common.RabbitMqUrl)
	if err != nil {
		log.Errorf("[%s]Failed to connect to RabbitMQ, err: %s, rabbitmqUrl: %s", fn, err, common.RabbitMqUrl)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("[%s]Failed to open a RabbitMQ channel: %s", fn, err)
		return
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		common.PushExchange, // name
		"direct",            // type
		true,                // durable
		false,               // auto-deleted
		false,               // internal
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		log.Errorf("[%s]Failed to declare a RabbitMQ exchange: %s", fn, err)
		return
	}

	for klineData := range mqChan {
		mqBody := MqBody{
			Type: "kline", // + klineData.CoinType,
			Data: klineData,
		}

		body, err := json.Marshal(mqBody)
		if err != nil {
			log.Errorf("[%s]Failed to marshal json: %s", fn, err)
			continue
		}

		mqMsg := RabbitMqMsg{
			AppId:     "option",
			EventType: "kline_" + klineData.CoinType,
			Body:      string(body),
		}
		data, err := json.Marshal(mqMsg)
		if err != nil {
			log.Errorf("[%s]Failed to marshal json: %s", fn, err)
			continue
		}
		err = ch.Publish(
			common.PushExchange, // exchange
			routingKey,          // routingKey
			true,                // mandatory
			false,               // immediate
			amqp.Publishing{
				ContentType:  "text/json",
				Body:         data,
				DeliveryMode: amqp.Persistent,
			})
		if err != nil {
			log.Errorf("[%s]Failed to publish msg, err: %s, msg: %v", fn, err, string(data))
			continue
		}
		log.Infof(" [%s]Succeeded to send msg: %s", fn, data)
		//time.Sleep(time.Second)
	}
}

func init() {
	common.ConfigLogger()
	kline.InitKLine()
	InitChan()
}

func pprof() {
	log.Error(http.ListenAndServe(common.LISTENPORT, nil))
}

func DealKlineTask() {
	defer func() { go DealKlineTask() }()
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case klineData := <-klineChan:
			if ch, ok := klineChanMap[klineData.CoinType]; ok {
				kline.DealCurrentKLine(klineData, ch)
			}
		case <-ticker.C:
			// 若当前秒无数据，则用上一秒数据
			kline.DealAllHistoryPrice(klineChanMap)
		}
	}
}

func SaveKlineTask(klineChan chan *kline.OptionKline) {
	fn := "SaveKlineTask"
	defer func() { go SaveKlineTask(klineChan) }()
	defer common.CheckPanic(fn, nil)
	for klineData := range klineChan {
		//klineData.Time = time.Now().Unix()
		if err := kline.SaveKLine2DB(klineData); err != nil {
			log.Errorf("Failed to save KLine to db, err: %s, data: %v", err, klineData)
		} else {
			if mqChan, ok := mqChanMap[klineData.CoinType]; ok {
				mqChan <- klineData
			}
		}
		// <1s, 防止时间不连续
		//time.Sleep(900 * time.Millisecond)
	}
}

func SaveKlineTask1() {
	fn := "SaveKlineTask1"
	defer func() { go SaveKlineTask1() }()
	defer common.CheckPanic(fn, nil)
	for klineData := range dbChan {
		if err := kline.SaveKLine2DB(klineData); err != nil {
			log.Errorf("Failed to save KLine to db, err: %s, data: %v", err, klineData)
		} else {
			if mqChan, ok := mqChanMap[klineData.CoinType]; ok {
				mqChan <- klineData
			}
		}
	}
}

func InitChan() {
	for _, coinType := range common.CoinSupported.Load().([]string) {
		mqChanMap[coinType] = make(chan *kline.OptionKline, 100)
		klineChanMap[coinType] = make(chan *kline.OptionKline, 100)
	}
}

func main() {
	log.Infof("[main]Server %s Begin ...", common.APPNAME)
	go pprof()
	// 1. wait msg and send it to rabbitmq
	for _, routingKey := range common.PushRoutineKeyList {
		for _, coinType := range common.CoinSupported.Load().([]string) {
			if ch, ok := mqChanMap[coinType]; ok {
				go PublishMqMsg(ch, routingKey)
			}
		}
	}

	for _, coinType := range common.CoinSupported.Load().([]string) {
		if ch, ok := klineChanMap[coinType]; ok {
			go SaveKlineTask(ch)
		}
	}

	go DealKlineTask()

	// 2. get kline
	go forex.GetForexData(klineChan)

	// 3. deal kline data
	quitSignal := common.QuitSignal()
	for {
		select {
		case s := <-quitSignal:
			log.Errorf("[main]Received quit signal: %v", s)
			log.Infof("[main]Server %s End ...", common.APPNAME)
			return
		}
	}
}
