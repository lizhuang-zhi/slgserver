package static_conf

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path"
	"slgserver/config"
	"slgserver/log"
)
var Basic basic

type NeedRes struct {
	Decree 		int	`json:"decree"`
	Grain		int `json:"grain"`
	Wood		int `json:"wood"`
	Iron		int `json:"iron"`
	Stone		int `json:"stone"`
	Gold		int	`json:"gold"`
}


type conscript struct {
	Des       string `json:"des"`
	CostWood  int    `json:"cost_wood"`
	CostIron  int    `json:"cost_iron"`
	CostStone int    `json:"cost_stone"`
	CostGrain int    `json:"cost_grain"`
	CostGold  int    `json:"cost_gold"`
}

type general struct {
	Des                   string `json:"des"`
	PhysicalPowerLimit    int    `json:"physical_power_limit"`    //体力上限
	CostPhysicalPower     int    `json:"cost_physical_power"`     //消耗体力
	RecoveryPhysicalPower int    `json:"recovery_physical_power"` //恢复体力
	ReclamationTime       int    `json:"reclamation_time"`        //屯田消耗时间，单位秒
	ReclamationCost       int    `json:"reclamation_cost"`        //屯田消耗政令
	DrawGeneralCost       int    `json:"draw_general_cost"`        //抽卡消耗金币

}

type basic struct {
	ConScript 	conscript
	General		general
}

func (this *basic) Load()  {
	jsonDir := config.File.MustValue("logic", "json_data", "../data/conf/")
	fileName := path.Join(jsonDir, "basic.json")
	jdata, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.DefaultLog.Error("basic load file error", zap.Error(err), zap.String("file", fileName))
		os.Exit(0)
	}

	json.Unmarshal(jdata, this)

	fmt.Println(this)
}
