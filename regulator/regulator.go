package regulator

import (
	"math"
	"math/rand"
	"option-kline/common"
	"time"

	log "github.com/sirupsen/logrus"
)

/*
kline局部修正算法介绍：

大盘顶线为selectTop，初始20，底线为selectBottom，初始-20

有一条标准线，StandLine，初始为0，在标准线上下波动为5的区间内，我们取随机数，取到的数值加到需要适配的Price上，就是最终的价格。
当随机数为正数时，StandLine自增1个值，反之自减1个值。
当标准线触达selectTop或者selectBottom的时候，我们以步长2强制增加或者减少标准线的值，直到标准线穿越或刚好到达0值。

在标准线移动的过程中，我们会有一个百分之30的概率，强制改变标准线的走势，例如本来是+1，干扰后就变成-1，以此来增加kline的不确定性。
另外在标准线处于2的整数倍取值时，还会有一个百分之10的概率，触发一个等同触顶或者触底的效果。

最后当标准线触顶或者触底时，我们将大盘的边界扩张10个值，让玩家抓不住底线在哪里。如果10分钟内，不再发生触顶或者触底的情况，则将大盘的边界收缩10个点，防止大盘边界无线扩大。
*/

// ---------------------------------------------------------------------------------------------------------------------

const (
	stateNormal = 0 // 正常
	stateDrop   = 1 // 断崖式下跌
	stateRise   = 2 // 断崖式上涨
)

var (
	selectTop    = common.SelectRange  // 当前大盘最高点
	selectBottom = -common.SelectRange // 当前大盘最低点
	selectStep   = common.SelectStep   // 大盘浮动值
	StateString  = map[int]string{
		stateNormal: "stateNormal",
		stateDrop:   "stateDrop",
		stateRise:   "stateRise",
	}
)

// ---------------------------------------------------------------------------------------------------------------------

type Section struct {
	Top               int
	Bottom            int
	StandLine         int
	state             int
	trim              int // 微调因子，再取随机数的时候，Top会减去该值，Bottom会加上该值
	trims             []*Trim
	lastPrice         float64
	name              string
	index             int
	touchBoundary     bool
	sectionUpdateTime time.Time // 大盘边界扩张/收缩的时间点
	randomNum         *rand.Rand
}

// 当标准线触碰到顶/底线后，强制扩张大盘边界，防止用户盯着一条底线
func (s *Section) expend() {
	s.Top += selectStep
	s.Bottom -= selectStep
	s.sectionUpdateTime = time.Now()
}

// 尝试收缩边界
func (s *Section) tryShrink() {
	// 在初始大盘，不可再收缩
	if s.Top == selectTop || s.Bottom == selectBottom {
		return
	}

	// 如果收缩以后，标准线发生了溢出，则不允许收缩
	if s.StandLine >= s.Top-selectStep || s.StandLine <= s.Bottom+selectStep {
		return
	}

	// 离上次扩张已经过去了10分钟，收缩一下大盘
	if time.Now().After(s.sectionUpdateTime.Add(time.Minute * 10)) {
		s.Top -= selectStep
		s.Bottom += selectStep
		s.sectionUpdateTime = time.Now()
	}
}

func (s *Section) updateState() {
	if s.StandLine <= s.Bottom {
		s.expend()
		if s.randomNum.Intn(10) < 5 {
			s.state = stateRise
			s.touchBoundary = true
			log.Infof("[Regulator] updateState get Top [%v]", s.index)
		}

	} else if s.StandLine >= s.Top {
		s.expend()
		if s.randomNum.Intn(10) < 5 {
			s.state = stateDrop
			s.touchBoundary = true
			log.Infof("[Regulator] updateState get Bottom [%v]", s.index)
		}

	} else if s.StandLine == 0 {
		s.state = stateNormal
		s.touchBoundary = false
	}

	// 标准状态下，给一个小概率强制上升或者下降，防止用户太容易窥探到底
	if s.state == stateNormal {
		if s.StandLine%2 == 0 && s.StandLine != 0 {
			if s.randomNum.Intn(100) < 20 {
				if s.StandLine > 0 {
					s.state = stateDrop
					s.touchBoundary = false
				} else {
					s.state = stateRise
					s.touchBoundary = false
				}
				// log.Infof("[Regulator] ESES%v ForceReverse CurStandLine[%v] CurState[%v]", s.name, s.StandLine, StateString[s.state])
			}
		}

	} else {
		// 触边界后，断崖下降或者上升2步以后，有百分之50的概率恢复成正常状态
		if ((s.Top-s.StandLine >= 2) || (s.StandLine-s.Bottom) >= 2) && s.touchBoundary {
			if s.randomNum.Intn(100) < 50 {
				log.Infof("[Regulator] updateState make normal [%v] [%v]", s.index, StateString[s.state])
				s.state = stateNormal
				s.touchBoundary = false
			}
		}
	}
}

func (s *Section) move(step int) {
	// 有百分之30的概率干扰，形成相反行为
	if s.randomNum.Intn(100) < 20 {
		// log.Infof("[Regulator] ESES%v move has been disturb, Origin[%v], After[%v]", s.name, step, step*-1)
		step *= -1
	}

	s.StandLine += step
	s.updateState()
	s.index++
}

func (s *Section) chooseTrim() {
	sum := 0
	for _, t := range s.trims {
		sum += t.weight
	}
	p := s.randomNum.Intn(sum)
	ts := 0
	for _, t := range s.trims {
		ts += t.weight
		if ts >= p {
			s.trim = t.value
			// log.Infof("[Regulator] Choose trim[%v]", s.trim)
			return
		}
	}
}

func (s *Section) trimTop() int {
	return s.StandLine + s.trim
}

func (s *Section) trimBottom() int {
	return s.StandLine - s.trim
}

func (s *Section) randomMove() int {
	s.tryShrink()

	s.chooseTrim()
	ret := s.randomNum.Intn(s.trimTop() - s.trimBottom())
	ret += s.trimBottom()

	// log.Infof("[Regulator] Name[%v], StandLine[%v], Move[%v]", s.name, s.StandLine, ret)

	switch s.state {
	case stateRise:
		s.move(1)

	case stateDrop:
		s.move(-1)

	case stateNormal:
		if ret > 0 {
			s.move(1)
		}
		if ret < 0 {
			s.move(-1)
		}
	}

	return ret
}

// ---------------------------------------------------------------------------------------------------------------------

type AdJustRequest struct {
	CoinType  string
	Price     float64
	Precision int
	Flag      int // 1:use last price, 2:don't change input value
}

// ---------------------------------------------------------------------------------------------------------------------

type Regulator struct {
	section  *Section
	request  chan *AdJustRequest
	response chan float64
}

// ---------------------------------------------------------------------------------------------------------------------

type Trim struct {
	value  int
	weight int
}

func NewRegulator(name string, trims []*Trim) *Regulator {
	return &Regulator{
		section: &Section{
			Top:       selectTop,
			Bottom:    selectBottom,
			state:     stateNormal,
			trims:     trims,
			name:      name,
			randomNum: rand.New(rand.NewSource(time.Now().UnixNano())),
		},
		request:  make(chan *AdJustRequest),
		response: make(chan float64),
	}
}

func (r *Regulator) AdjustPrice(req *AdJustRequest) float64 {
	r.request <- req
	return <-r.response
}

// 控制报价时，防止报价溢出真实报价震辐太多,目前控制为真实报价上下:20个单位
func (r *Regulator) workLoop() {
	fn := "workLoop"
	for req := range r.request {
		move := r.section.randomMove()
		rd := rand.New(rand.NewSource(time.Now().UnixNano()))
		if req.Flag == 2 {
			// don't change price if Flag is 2
			r.section.lastPrice = req.Price
			r.response <- r.section.lastPrice
		} else if (req.Flag == 1 || rd.Int63n(100) < 25) && r.section.lastPrice > 0 {
			// use last price
			r.response <- r.section.lastPrice
		} else {
			precisionN := math.Pow10(req.Precision)
			amplitudeConf := common.PriceAmplitude.Load().(int)
			factor := float64(move%amplitudeConf) / precisionN
			basePrice := req.Price
			if r.section.lastPrice > 0 {
				req.Price += (r.section.lastPrice - req.Price) / 2
			} else {
				r.section.lastPrice = req.Price
			}
			// 1. 控制价格震幅尽量平滑
			amplitude := float64(amplitudeConf) / precisionN
			maxPrice, minPrice := r.section.lastPrice+amplitude, r.section.lastPrice-amplitude
			newPrice := req.Price + factor
			adjDiff := float64(rd.Intn(amplitudeConf)) / precisionN
			if newPrice > maxPrice {
				newPrice = maxPrice - adjDiff
			} else if newPrice < minPrice {
				newPrice = minPrice + adjDiff
			}
			// 2. 控制价格震幅在基本报价范围内
			basePriceTop, basePriceBottom := basePrice+0.1, basePrice-0.1
			if newPrice > basePriceTop {
				newPrice = newPrice - adjDiff
			} else if newPrice < basePriceBottom {
				newPrice = newPrice + adjDiff
			} else if rd.Intn(100) < 20.0 {
				if rd.Intn(100) < 50 {
					newPrice = newPrice - adjDiff
				} else {
					newPrice = newPrice + adjDiff
				}
			}
			log.Debugf("[%s][%s]ori: %f, amplitude: %f, factor: %f, dst: %f",
				fn, req.CoinType, basePrice, amplitude, factor, newPrice)
			r.section.lastPrice = newPrice
			r.response <- newPrice
		}
	}
}

var (
	RegulatorMap = make(map[string]*Regulator)
)

func init() {
	for _, coinType := range common.CoinSupported.Load().([]string) {
		trims := []*Trim{{10, 10}, {4, 6}, {8, 8}}
		if coinType == "USDT" {
			trims = []*Trim{{2, 10}, {5, 20}, {10, 8}}
		}
		RegulatorMap[coinType] = NewRegulator(coinType, trims)
		go RegulatorMap[coinType].workLoop()
	}
}
