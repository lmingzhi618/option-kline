package common

import (
	"testing"
)



func Test_Int64Explode(t *testing.T) {
	if Int64Explode([]int64{10,20}, ",") == "10,20" && Int64Explode([]int64{10}, ",") == "10" {
		t.Log("Int64Explode第一个测试通过")
	}else{
		t.Error("Int64Explode()测试未通过")
	}
	if IntExplode([]int{10,20}, ",") == "10,20" && IntExplode([]int{10}, ",") == "10" {
		t.Log("IntExplode第一个测试通过")
	}else{
		t.Error("IntExplode()测试未通过")
	}
}

type TestObj struct {
	Id int64
	Name string
}

func Test_StructToJsonStr(t *testing.T) {
	testobj := TestObj{
		Id:100,
		Name:"111",
	}
	_,err := StructToJsonStr(testobj)
	if err!=nil  {
		t.Error("第一个StructToJsonStr测试不通过")
	}else{
		t.Log("第一个StructToJsonStr测试通过")
	}
}

func Test_FindMaxNumInt64(t *testing.T) {
	res,err := FindMaxNumInt64(100,200,300,-1000)
	if err!=nil || res!=300 {
		t.Error("FindMaxNumInt64第一个测试未通过")
	}
	res1 ,err := FindMaxNumFloat64(200.122,3911.2,2332.4)
	if err!=nil || res1!=3911.2{
		t.Error("FindMaxNumFloat64 第一个测试不通过")
	}
	res2,err := FindMinNumInt64(1,10,-1000,32323)
	if err!=nil || res2!=-1000 {
		t.Error("FindMinNumInt64 第一个测试不通过")
	}
}

func Test_Contain(t *testing.T) {
	if Contain([]int64{100,200,300}, 1100) {
		t.Error("Contain 第一个测试不通过")
	}
	if !Contain([]float64{111.2,123.4,122.3}, 111.2) {
		t.Error("Contain 第二个测试不通过")
	}
	if !Contain([]float64{112.332}, 112.332) {
		t.Error("Contain 第三个测试不通过")
	}
	if !Contain(map[string]interface{}{"one":1,"two":2}, "one") {
		t.Error("Contain 第4个测试不通过")
	}
}


func Test_FloatStrToIntStr(t *testing.T) {
	if FloatStrToIntStr("1.12") != "1120000000000000000" {
		t.Error("FloatStrToIntStr(1.12) 测试不通过")
	}

	if FloatStrToIntStr("112") != "112000000000000000000" {
		t.Error("FloatStrToIntStr(112) 测试不通过")
	}

	if FloatStrToIntStr("1.0") != "1000000000000000000" {
		t.Error("FloatStrToIntStr(1.0) 测试不通过")
	}

	if FloatStrToIntStr("123456789123456789") != "123456789123456789000000000000000000" {
		t.Error("FloatStrToIntStr(123456789123456789) 测试不通过")
	}

	if FloatStrToIntStr("-1.0") != "-1000000000000000000" {
		t.Error("FloatStrToIntStr(-1.0) 测试不通过")
	}

	if FloatStrToIntStr("-01.0") != "-1000000000000000000" {
		t.Error("FloatStrToIntStr(-01.0) 测试不通过")
	}

	if FloatStrToIntStr("0.002") != "2000000000000000" {
		t.Error("FloatStrToIntStr(0.002) 测试不通过")
	}

	if FloatStrToIntStr("0.00") != "0" {
		t.Error("FloatStrToIntStr(0) 测试不通过, actual value=", FloatStrToIntStr("0"))
	}

}


func Test_IntStrToFloatStr(t *testing.T) {
	if IntStrToFloatStr("123") != "0.000000000000000123" {
		t.Error("IntStrToFloatStr(123) 测试不通过")
	}

	if IntStrToFloatStr("-123") != "-0.000000000000000123" {
		t.Error("IntStrToFloatStr(-123) 测试不通过")
	}

	if IntStrToFloatStr("123456789123456789") != "0.123456789123456789" {
		t.Error("IntStrToFloatStr(123456789123456789) 测试不通过")
	}

	if IntStrToFloatStr("-123456789123456789") != "-0.123456789123456789" {
		t.Error("IntStrToFloatStr(-123456789123456789) 测试不通过")
	}

	if IntStrToFloatStr("123456789123456789001") != "123.456789123456789001" {
		t.Error("IntStrToFloatStr(123456789123456789001) 测试不通过")
	}

	if IntStrToFloatStr("-123456789123456789001") != "-123.456789123456789001" {
		t.Error("IntStrToFloatStr(-123456789123456789001) 测试不通过")
	}

	if IntStrToFloatStr("0") != "0.000000000000000000" {
		t.Error("IntStrToFloatStr(-0) 测试不通过,actual value=", IntStrToFloatStr("0"))
	}
}


func Test_FloatStrRound(t *testing.T) {
	if FloatStrRound("1.234", 2) != "1.23" {
		t.Error("FloatStrRound(1.234, 2) 测试不通过")
	}

	if FloatStrRound("1.234", 3) != "1.234" {
		t.Error("FloatStrRound(1.234, 3) 测试不通过")
	}

	if FloatStrRound("1.234", 4) != "1.2340" {
		t.Error("FloatStrRound(1.234, 4) 测试不通过")
	}

	if FloatStrRound("1.234", 0) != "1" {
		t.Error("FloatStrRound(1.234, 0) 测试不通过")
	}

	if FloatStrRound("12", 2) != "12.00" {
		t.Error("FloatStrRound(12, 2) 测试不通过")
	}
	if FloatStrRound(".0", 0) != ".0" {
		t.Error("FloatStrRound(.0, 0) 测试不通过")
	}
}
















