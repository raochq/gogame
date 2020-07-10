package csvparse

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type exampleStruct struct {
	Id       int64           // 序号
	Train    Star            `csv:"train"`     // 培养组
	Level    int32           `csv:"level"`     // 培养等级
	PieceNum int32           `csv:"piece_num"` // 消耗碎片数量
	Item     []int32         `csv:"item"`      // 消耗材料
	ItemNum  []int32         `csv:"item_num"`  // 消耗材料数量
	ItemCost cost            //消耗材料
	ParamMap map[int32]int32 `csv:"param"` // 词条类型1-5属性值
	ParamArr []int32         `csv:"param"` // 词条类型1-5属性值
}
type cost map[int]int

func (c cost) UnmarshalCSVEx(data map[string]string) error {
	item := strings.Split(data["item"], ",")
	num := strings.Split(data["item_num"], ",")
	if len(num) != len(item) {
		return fmt.Errorf("len(item) != len(item_num) for id =%v", data["id"])
	}
	for i := 0; i < len(num); i++ {
		inum, err := strconv.Atoi(num[i])
		if err != nil {
			return err
		}
		iID, e := strconv.Atoi(item[i])
		if e != nil {
			return e
		}
		c[iID] = inum
	}
	return nil
}

type Star string

func (s *Star) UnmarshalCSV(value string) error {
	switch value {
	case "2":
		*s = "普通"
	case "3":
		*s = "稀有"
	case "4":
		*s = "非凡"
	case "5":
		*s = "闪耀"
	default:
		*s = Star("其他:" + value)
	}
	return nil
}

func TestLoadCSVMap(t *testing.T) {
	type args struct {
		filename string
		//info     map[int32]map[int64]exampleStruct
		info interface{}
		keys []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "example1",
			args: args{filename: "E:/NN4Streams/DevTW/Design/server/ClothesColorV2Train.csv",
				info: map[Star]map[int64]exampleStruct{},
				keys: []string{"train", "ID"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := LoadCSVMap(tt.args.filename, &tt.args.info, tt.args.keys[0], tt.args.keys[1:]...)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadCSVMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Log(tt.args.info)
		})
	}
}
