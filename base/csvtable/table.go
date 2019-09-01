package csvtable

import (
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/raochq/gogame/base/logger"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type StringArray string

type CsvTable struct {
	KeyMap  map[string]int
	Records [][]string
}

// 加载csv文件
func (c *CsvTable) Load(fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		return errors.New(fmt.Sprintf("error opening file %v ", err))

	}
	defer file.Close()

	// csv 读取器
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return errors.New(fmt.Sprint("cannot parse csv file.", file.Name(), err))
	}

	// 是否为空档
	if len(records) == 0 {
		return errors.New(fmt.Sprint("csv file is empty", file.Name()))
	}

	// 记录数据, 第一行为表头，因此从第二行开始
	rownames := records[0]
	keyMap := make(map[string]int)
	for index, name := range rownames {
		keyMap[strings.ToLower(name)] = index
	}
	c.KeyMap = keyMap
	//检查合法性
	keyLen := len(keyMap)
	for line := 1; line < len(records); line++ {
		if len(records[line]) != keyLen {
			return fmt.Errorf("row length  not eqrt key at %v", line)
		}
	}
	c.Records = records[1:]
	return nil
}

// 解析成map[string]value
func (c *CsvTable) Unmarshal(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Map || rv.IsNil() {
		return errors.New(fmt.Sprint("Unmarshal must input map[string]value"))
	}
	if len(c.Records) == 0 {
		return nil
	}

	tyKey := rv.Type().Key()
	tyValue := rv.Type().Elem()

	for idx, line := range c.Records {
		kValue := reflect.New(tyKey).Elem()
		err := setValue(kValue, line[0])
		if err != nil {
			return fmt.Errorf("Unmarshal failed at line %d as %v ", idx, line[0])
		}

		vValue := reflect.New(tyValue).Elem()
		if vValue.Kind() == reflect.Ptr {
			vValue = reflect.New(tyValue.Elem())
		}
		err = c.setRowValue(vValue, line)
		if err != nil {
			return fmt.Errorf("Unmarshal failed at line %d ", idx)
		}
		fmt.Println(kValue.Interface(), vValue.Interface())
		rv.SetMapIndex(kValue, vValue)
	}
	return nil
}
func (c *CsvTable) setRowValue(v reflect.Value, record []string) error {
	v = reflect.Indirect(v)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		field_name := v.Type().Field(i).Name

		tag := v.Type().Field(i).Tag.Get("csv")
		// 判断是否忽略
		if tag == "-" || tag == "_" || tag == "omitempty" {
			continue
		}
		if tag == "" {
			tag = field_name
		}

		keyIdx, ok := c.KeyMap[ strings.ToLower(tag)]
		if !ok {
			logger.Warn("CsvTable.setRowValue: field[%v] value not found", field_name)
			continue
		}
		if err := setValue(field, record[keyIdx]); err != nil {
			logger.Warn("CsvTable.setRowValue: error = %v, field = %v,", err, field_name)
		}
	}
	return nil
}
func setValue(field reflect.Value, value string) error {
	var err error
	field = reflect.Indirect(field)

	field_type := field.Type().String()

	switch field_type {
	case "int", "int8", "int16", "int32", "int64", "*int":
		err = fieldSetInt(field, value)
	case "uint", "uint8", "uint16", "uint32", "uint64":
		err = fieldSetUint(field, value)
	case "float32", "float64":
		err = fieldSetFloat(field, value)
	case "string":
		err = fieldSetString(field, value)
	case "bool":
		err = fieldSetBool(field, value)
	case "[]int", "[]uint32", "[]string", "[]float64", "[]uint8", "[]int32", "[]float32", "[]int64":
		err = fieldSetSlice(field, value, field_type)
	//case "map[int32]float32", "map[int32]float64", "map[int32]string", "map[int32]int32", "map[int32]int", "map[int32]uint8", "map[int32]uint32":
	//err = FieldSetTupleList(field, value, field_type)
	//case "*variable.StringTime":
	//	err = FieldSetTimeSection(field, value)
	default:
		return fmt.Errorf("VariableParser not support field type [%v]", field_type)
	}

	return err
}
func fieldSetInt(field reflect.Value, value string) error {
	i, e := strconv.Atoi(value)
	if e != nil && value != "" {
		return e
	}

	field.SetInt(int64(i))

	return nil
}
func fieldSetUint(field reflect.Value, value string) error {
	i, e := strconv.Atoi(value)
	if e != nil && value != "" {
		return e
	}

	field.SetUint(uint64(i))

	return nil
}

func fieldSetFloat(field reflect.Value, value string) error {
	i, e := strconv.ParseFloat(value, 64)
	if e != nil && value != "" {
		return e
	}

	field.SetFloat(float64(i))

	return nil
}

func fieldSetString(field reflect.Value, value string) error {
	field.SetString(value)
	return nil
}

func fieldSetBool(field reflect.Value, value string) error {
	value = strings.ToLower(value)
	if value == "true" || value == "yes" || value == "1" || value == "y" || value == "enable" || value == "on" {
		field.SetBool(true)
	} else if value == "false" || value == "no" || value == "0" || value == "n" || value == "disable" || value == "off" {
		field.SetBool(false)
	} else {
		return fmt.Errorf("bool value[%v] error", value)
	}
	return nil
}

func fieldSetSlice(field reflect.Value, value string, field_type string) error {
	var slice interface{}
	var err error

	switch field_type {
	case "[]int":
		slice, err = StringArray(value).ToInt()
	case "[]uint32":
		slice, err = StringArray(value).ToUint32()
	case "[]string":
		slice = StringArray(value).ToString()
	case "[]float64":
		slice, err = StringArray(value).ToFloat64()
	case "[]uint8":
		slice, err = StringArray(value).ToUint8()
	case "[]int32":
		slice, err = StringArray(value).ToInt32()
	case "[]float32":
		slice, err = StringArray(value).ToFloat32()
	case "[]int64":
		slice, err = StringArray(value).ToInt64()
	}

	if err != nil {
		return nil
	}

	field.Set(reflect.ValueOf(slice))
	return nil
}

func (s StringArray) ToString() []string {
	param := strings.Trim(string(s), " []{}")
	if param == "" {
		return []string{}
	}
	return strings.Split(param, ",")
}

func (s StringArray) ToInt() ([]int, error) {
	var values []int
	strs := s.ToString()
	for _, str := range strs {
		v, e := strconv.Atoi(str)
		if e == nil {
			values = append(values, v)
		} else {
			return values, fmt.Errorf("Cannot parse str[\"%s\"] to int", str)
		}
	}

	return values, nil
}

func (s StringArray) ToFloat32() ([]float32, error) {
	var values []float32
	strs := s.ToString()
	for _, str := range strs {
		v, e := strconv.ParseFloat(str, 64)
		if e == nil {
			values = append(values, float32(v))
		} else {
			return values, fmt.Errorf("Cannot parse str[\"%s\"] to float32", str)
		}
	}

	return values, nil
}

func (s StringArray) ToFloat64() ([]float64, error) {
	var values []float64
	strs := s.ToString()
	for _, str := range strs {
		v, e := strconv.ParseFloat(str, 64)
		if e == nil {
			values = append(values, v)
		} else {
			return values, fmt.Errorf("Cannot parse str[\"%s\"] to float64", str)
		}
	}

	return values, nil
}

func (s StringArray) ToUint8() ([]uint8, error) {
	var values []uint8
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, uint8(v))
	}

	return values, nil
}

func (s StringArray) ToUint32() ([]uint32, error) {
	var values []uint32
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, uint32(v))
	}

	return values, nil
}

func (s StringArray) ToUint16() ([]uint16, error) {
	var values []uint16
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, uint16(v))
	}

	return values, nil
}

func (s StringArray) ToUint64() ([]uint64, error) {
	var values []uint64
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, uint64(v))
	}

	return values, nil
}

func (s StringArray) ToInt32() ([]int32, error) {
	var values []int32
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, int32(v))
	}

	return values, nil
}

func (s StringArray) ToInt64() ([]int64, error) {
	var values []int64
	int_values, err := s.ToInt()
	if err != nil {
		return values, err
	}

	for _, v := range int_values {
		values = append(values, int64(v))
	}

	return values, nil
}
