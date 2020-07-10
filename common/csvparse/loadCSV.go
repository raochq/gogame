package csvparse

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

type TypeUnmarshallerEx interface {
	UnmarshalCSVEx(map[string]string) error
}
type structInfo struct {
	rType  reflect.Type
	Fields []fieldInfo
}
type fieldInfo struct {
	fieldName string
	key       string
	omitEmpty bool
	single    bool
}

//根据csv表头，生产要处理的字段
func getStructInfo(rType reflect.Type, keys []string) *structInfo {
	ret := &structInfo{
		rType: rType,
	}
	if rType.Kind() == reflect.Ptr {
		rType = rType.Elem()
	}
	if rType.Kind() != reflect.Struct {
		return nil
	}
	temV := reflect.New(rType).Elem()

	for i := 0; i < rType.NumField(); i++ {
		fieldName := rType.Field(i).Name
		field := temV.Field(i)
		if !field.CanSet() {
			continue
		}

		tagName := rType.Field(i).Tag.Get("csv")
		tagName = strings.ToLower(tagName)
		if tagName == "_" || tagName == "-" {
			continue
		}
		var key string
		omitEmpty := false
		if tagName != "" {
			key = strings.ToLower(tagName)
			tmp := strings.Split(key, ",")
			if len(tmp) > 0 {
				key = tmp[0]
				for _, t := range tmp {
					if t == "omitempty" {
						omitEmpty = true
						break
					}
				}
			}
		} else {
			key = strings.ToLower(fieldName)
		}
		fdInfo := fieldInfo{
			fieldName: fieldName,
			omitEmpty: omitEmpty,
			key:       key,
		}
		for _, k := range keys {
			if k == key {
				fdInfo.single = true
				break
			}
		}
		ret.Fields = append(ret.Fields, fdInfo)
	}
	return ret
}

// 需要用到其他字段来解析。比如消耗道具和数量是在2个字段，解析时可以放到一起
func tryUnMarshallEx(field reflect.Value, value map[string]string) (bool, error) {
	unMarshallIt := func(finalField reflect.Value) (bool, error) {
		if finalField.CanInterface() {
			fieldIface := finalField.Interface()
			fieldTypeUnmarshaller, ok := fieldIface.(TypeUnmarshallerEx)
			if ok {
				return ok, fieldTypeUnmarshaller.UnmarshalCSVEx(value)
			}
		}
		return false, nil
	}
	dupField := field

	for dupField.Kind() == reflect.Interface || dupField.Kind() == reflect.Ptr {
		if dupField.IsNil() {
			dupField.Set(reflect.New(dupField.Type().Elem()))
			if dupField.Type().Elem().Kind() == reflect.Map {
				v := reflect.MakeMap(field.Type().Elem())
				dupField.Elem().Set(v)
			}
			return unMarshallIt(dupField)
		}
		dupField = dupField.Elem()
	}

	if dupField.CanAddr() {
		return unMarshallIt(dupField.Addr())
	}
	return false, nil
}

// 多个field解析成一个字段
func tryUnMarshInnerField(field reflect.Value, data map[string]string, fdInfo fieldInfo) error {
	find := false
	for k, v := range data {
		if !strings.HasPrefix(k, fdInfo.key) {
			continue
		}
		index, _ := strconv.Atoi(strings.Trim(k, fdInfo.key))
		if index <= 0 {
			continue
		}
		find = true
		el := reflect.New(field.Type().Elem()).Elem()
		err := setField(el, v, true)
		if err != nil {
			return err
		}
		switch field.Kind() {
		case reflect.Slice:
			field.Set(reflect.Append(field, el))
		case reflect.Map:
			if field.IsNil() {
				field.Set(reflect.MakeMap(field.Type()))
			}
			field.SetMapIndex(reflect.ValueOf(int32(index)), el)
		default:
			return fmt.Errorf("tryUnMarshInnerField unsupported type %v", field.Kind())
		}
	}
	if !find && !fdInfo.omitEmpty {
		return fmt.Errorf("[%v] cannot not find data ", fdInfo.fieldName)
	}
	return nil
}
func parseLine(rTypeInfo *structInfo, data map[string]string) (reflect.Value, error) {
	newVal := reflect.New(rTypeInfo.rType).Elem()
	dubValue := newVal
	if newVal.Kind() == reflect.Ptr {
		if newVal.IsNil() {
			newVal.Set(reflect.New(newVal.Type().Elem()))
		}
		dubValue = newVal.Elem()
	}
	for _, fdInfo := range rTypeInfo.Fields {
		field := dubValue.FieldByName(fdInfo.fieldName)
		if field.Kind() == reflect.Map {
			field.Set(reflect.MakeMap(field.Type()))
		}
		if ok, err := tryUnMarshallEx(field, data); ok {
			if err != nil {
				return reflect.Value{}, err
			}
			continue
		}
		if fdInfo.single {
			err := setField(field, data[fdInfo.key], fdInfo.omitEmpty)
			if err != nil {
				return reflect.Value{}, err
			}
		} else {
			if err := tryUnMarshInnerField(field, data, fdInfo); err != nil {
				return reflect.Value{}, err
			}
		}
	}
	return newVal, nil
}

// 检查输入的map的合法性，返回最终元素的类型和错误
func ensureType(rValue reflect.Value, keys []string) (reflect.Type, error) {
	if len(keys) == 0 {
		return nil, errors.New("keys is empty")
	}
	if rValue.Kind() != reflect.Map {
		return nil, fmt.Errorf("the confMap must be a pointer to map")
	}
	tp := rValue.Type()
	for _, key := range keys {
		if tp.Kind() != reflect.Map {
			return nil, fmt.Errorf("the Type of Key[%s] must be a map", key)
		}
		tp = tp.Elem()
	}
	etp := tp
	if etp.Kind() == reflect.Ptr || etp.Kind() == reflect.Interface {
		etp = etp.Elem()
	}
	if etp.Kind() != reflect.Struct {
		return nil, fmt.Errorf("the element must be a struct or a pointer to a struct")
	}

	// 取Key的类型
	for idx, key := range keys {
		find := false
		for i := 0; i < etp.NumField(); i++ {
			f := etp.Field(i)
			if strings.EqualFold(f.Name, key) {
				keys[idx] = f.Name
				find = true
				break
			}
		}
		if !find {
			return nil, fmt.Errorf("not found key %s", key)
		}
	}
	return tp, nil
}

func convertToMap(el reflect.Value, vMap reflect.Value, keys []string) error {
	if len(keys) == 0 {
		return fmt.Errorf("keys is empty")
	}
	key := keys[0]
	elv := reflect.Indirect(el)

	k := elv.FieldByName(key)
	if k.IsValid() == false {
		return fmt.Errorf("[%v] field %s is not a valid value", el, key)
	}
	if vMap.Kind() != reflect.Map {
		return fmt.Errorf("the element of %v is not a map", vMap.Type())
	}
	if len(keys) == 1 {
		vMap.SetMapIndex(k, el)
	} else {
		v := vMap.MapIndex(k)
		if !v.IsValid() || v.IsNil() {
			v = reflect.MakeMap(vMap.Type().Elem())
			vMap.SetMapIndex(k, v)
		}
		convertToMap(el, v, keys[1:])
	}

	return nil
}
func loadFile(fileName string) ([][]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Errorf("error opening file %v ", err)
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, e := reader.ReadAll()
	if e != nil {
		return nil, fmt.Errorf("[%s] parse file failed: %v", fileName, e)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("loadFile [%s] is empty", fileName)
	}
	return records, nil
}

// 加载csv到slice，name： csv文件名，confSlice, slice的地址
func LoadCSVArr(fileName string, confSlice interface{}) (ret error) {
	defer func() {
		r := recover()
		if r != nil {
			ret = fmt.Errorf("LoadCSVArr [%s] failed: %v", fileName, r)
		}
	}()
	rv := reflect.ValueOf(confSlice)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("the confSlice must be a pointer to slice")
	} else {
		if rv.IsNil() {
			rv.Set(reflect.New(rv.Type().Elem()))
		}
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Slice {
		return fmt.Errorf("the out must be a slice")
	}
	eTp := rv.Type().Elem()

	records, err := loadFile(fileName)
	if err != nil {
		return fmt.Errorf("[%s] parse csv file failed:%v", fileName, err)
	}
	recordInfo := getStructInfo(eTp, records[0])
	if recordInfo == nil {
		return fmt.Errorf("[%s] get Struct info failed", fileName)
	}
	for line := 1; line < len(records); line++ {
		if len(records[line]) != len(records[0]) {
			return fmt.Errorf("[%s] len(records[%d]) != len(records[0]", fileName, line)
		}
		data := make(map[string]string)
		for n := 0; n < len(records[0]); n++ {
			key := strings.TrimSpace(strings.ToLower(records[0][n]))
			val := strings.TrimSpace(records[line][n])
			data[key] = val
		}
		v, e := parseLine(recordInfo, data)
		if e != nil {
			return fmt.Errorf("[%s] parse line %d failed: %v", fileName, line, e)
		}
		rv.Set(reflect.Append(rv, v))
	}
	return nil
}

// 加载csv到Map， csv文件名，confMap, map或map地址， key map键值对应的字段名，OptionKeys 多层map的键值对应的字段名，键值不区分大小写
func LoadCSVMap(fileName string, confMap interface{}, Key string, OptionKeys ...string) (ret error) {
	defer func() {
		r := recover()
		if r != nil {
			ret = fmt.Errorf("LoadCSVMap [%s] failed: %v", fileName, r)
		}
	}()
	confV := reflect.ValueOf(confMap)
	confV = reflect.Indirect(confV)
	if confV.Kind() == reflect.Interface {
		confV = confV.Elem()
	}
	var keys []string
	keys = append(keys, Key)
	keys = append(keys, OptionKeys...)
	if confV.IsNil() {
		if !confV.CanSet() {
			return fmt.Errorf("the confMap must can setable")
		}
		confV.Set(reflect.MakeMap(confV.Type()))
	}
	etp, err := ensureType(confV, keys)
	if err != nil {
		return err
	}

	arrV := reflect.New(reflect.SliceOf(etp))
	err = LoadCSVArr(fileName, arrV.Interface())
	if err != nil {
		return err
	}
	arrV = arrV.Elem()
	for i := 0; i < arrV.Len(); i++ {
		err = convertToMap(arrV.Index(i), confV, keys)
		if err != nil {
			return err
		}
	}
	return nil
}
