package common

import (
	"github.com/shopspring/decimal"
)

// Add returns a + b.
func BcAdd(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Add(bb)
	return aa.StringFixedBank(precision), nil
}

// Sub returns a - b.
func BcSub(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Sub(bb)
	return aa.StringFixedBank(precision), nil
}

func BcDiv(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Div(bb)
	return aa.StringFixedBank(precision), nil
}

func BcMul(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Mul(bb)
	return aa.StringFixedBank(precision), nil
}

func BcPow(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Pow(bb)
	return aa.StringFixedBank(precision), nil
}

func BcMod(a string, b string, precision int32) (string, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}

	aa = aa.Mod(bb)
	return aa.StringFixedBank(precision), nil
}

// Cmp compares the numbers represented by d and d2 and returns:
//     -1 if a <  b
//      0 if a == b
//     +1 if a >  b
func BcCmp(a string, b string) (int, error) {
	aa, err := decimal.NewFromString(a)
	if err != nil {
		return 0, err
	}

	bb, err := decimal.NewFromString(b)
	if err != nil {
		return 0, err
	}

	return aa.Cmp(bb), nil
}

func BcCheckRange(amount, max, min string) (bool, error) {
	// 1. max < amount
	higher, err := BcCmp(amount, max)
	if err != nil {
		return false, err
	}
	if higher > 0 {
		return false, nil
	}
	// 2. min > amount
	lower, err := BcCmp(min, amount)
	if err != nil {
		return false, err
	}
	if lower > 0 {
		return false, nil
	}
	return true, nil
}

// @return agentMoney, fee, winMoney, capital
func BcGetAgentAndWinMoney(amount, feeRate, payRate string) (string, string, string, string, error) {
	// 1. 获取下单本金: 下单金额/(1+手续费率)
	totalFeeRate, err := BcAdd("1", feeRate, 18)
	if err != nil {
		return "", "", "", "", err
	}
	capital, err := BcDiv(amount, totalFeeRate, 18)
	if err != nil {
		return "", "", "", "", err
	}

	// 2.获取头寸
	position, err := BcMul(capital, payRate, 18)
	if err != nil {
		return "", "", "", "", err
	}
	// 3.获取代理商实际金额：本金－头寸
	agentMoney, err := BcSub(capital, position, 18)
	if err != nil {
		return "", "", "", "", err
	}

	winMoney, err := BcAdd(capital, agentMoney, 18)
	if err != nil {
		return "", "", "", "", err
	}

	fee, err := BcSub(amount, capital, 18)
	if err != nil {
		return "", "", "", "", err
	}

	return agentMoney, fee, winMoney, capital, nil
}

func BcGetAgentMoney(amount, feeRate, payRate string) (string, error) {
	// 1. 获取下单本金: 下单金额/(1+手续费率)
	totalFeeRate, err := BcAdd("1", feeRate, 18)
	if err != nil {
		return "", err
	}
	capital, err := BcDiv(amount, totalFeeRate, 18)
	if err != nil {
		return "", err
	}

	// 2.获取头寸
	position, err := BcMul(capital, payRate, 18)
	if err != nil {
		return "", err
	}
	// 3.获取代理商实际金额：本金－头寸
	agentMoney, err := BcSub(capital, position, 18)
	if err != nil {
		return "", err
	}
	return agentMoney, nil
}

var (
	tenPow18 string
)

func init() {
	var err error
	tenPow18, err = BcPow("10", "18", 0)
	if err != nil {
		panic(err)
	}
}

// 普通金额字符串转换为区块链38位金额字符串
func Money2BlockMoney(money string) string {
	if money == "" {
		return "0"
	}

	a, err := BcMul(money, tenPow18, 0)
	if err != nil {
		return "0"
	}

	return a
}

// 区块链38位金额字符串转普通字符串
func BlockMoney2Money(blockMoney string, precision int32) string {
	if blockMoney == "" {
		return "0"
	}

	a, err := BcDiv(blockMoney, tenPow18, precision)
	if err != nil {
		return "0"
	}

	return a
}
