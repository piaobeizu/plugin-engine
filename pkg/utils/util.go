/*
 @Version : 1.0
 @Author  : steven.wong
 @Email   : 'wwangxiaoakng@modelbest.cn'
 @Time    : 2024/04/11 14:33:49
 Desc     :
*/

package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

type FlatteItem struct {
	Name string
	Val  any
	Kind string
}

func FormatStruct(data any, indent bool) string {
	if !indent {
		str, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}
		return string(str)
	}
	str, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}
	var content = string(str)
	content = strings.Replace(content, "\\u003c", "<", -1)
	content = strings.Replace(content, "\\u003e", ">", -1)
	content = strings.Replace(content, "\\u0026", "&", -1)
	content = strings.Replace(content, "\\\\", "", -1)
	return content
}

func WriteFile(data any, filename string, append bool) error {
	var (
		err  error
		file *os.File
	)
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}
	if append {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	} else {
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(FormatStruct(data, true))
	if err != nil {
		return err
	}
	return nil
}

func CreateToken() string {
	return "token"
}

func SortMapByKey(data map[string]any) map[string]any {
	var (
		keys []string
		maps = make(map[string]any)
	)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i := 0; i < len(keys); i++ {
		maps[keys[i]] = data[keys[i]]
	}
	return maps
}

func StructToMap(obj any) map[string]FlatteItem {
	return flatten(obj)
}

func ConfigEqual(a, b any) bool {
	aFI := flatten(a)
	bFI := flatten(b)
	logrus.Infof("aFI: %v", aFI)
	logrus.Infof("bFI: %v", bFI)
	if len(aFI) != len(bFI) {
		return false
	}
	for k, aval := range aFI {
		if _, ok := bFI[k]; !ok {
			return false
		}
		if aval.Kind != bFI[k].Kind {
			return false
		}
		if !reflect.DeepEqual(aval.Val, bFI[k].Val) {
			return false
		}
	}
	return true
}

func flatten(obj any) map[string]FlatteItem {
	result := make(map[string]FlatteItem)
	// mp := structToMap(obj)
	var f func(any, string)
	f = func(o any, prefix string) {

		var mp = make(map[string]any)
		objValue := reflect.ValueOf(o)
		if reflect.TypeOf(o).Kind() == reflect.Ptr {
			objValue = objValue.Elem()
		}
		kind := objValue.Kind()
		switch kind {
		case reflect.Slice, reflect.Array:
			for i := 0; i < objValue.Len(); i++ {
				mp[fmt.Sprintf("%d", i)] = objValue.Index(i).Interface()
			}
		case reflect.Struct:
			for i := 0; i < objValue.NumField(); i++ {
				field := objValue.Type().Field(i)
				// skip unexported field
				if field.PkgPath != "" {
					continue
				}
				fieldValue := objValue.Field(i).Interface()
				jsonName := field.Tag.Get("json")
				jsonName = strings.Replace(jsonName, ",inline", "", -1)
				jsonName = strings.Replace(jsonName, ",omitempty", "", -1)
				mp[jsonName] = fieldValue
			}
		case reflect.Interface:
			mp[prefix] = o
		case reflect.Map, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.Invalid:
			return
		default:
			name := strings.Trim(prefix, ".")
			// remove the inline and omitempty
			name = strings.Replace(name, "..", ".", -1)
			result[strings.ToLower(name)] = FlatteItem{
				Name: name,
				Val:  o,
				Kind: kind.String(),
			}
		}
		for k, v := range mp {
			f(v, prefix+k+".")
		}
	}
	f(obj, "")
	return result
}

func FindCaller(skip int) (string, int) {
	file := ""
	line := 0
	for i := 0; i < 10; i++ {
		file, line = getCaller(skip + i)
		if !strings.HasPrefix(file, "logrus") {
			break
		}
	}
	return file, line
	// return fmt.Sprintf("%s:%d", file, line)
}

func getCaller(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0
	}
	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 3 {
				file = file[i+1:]
				break
			}
		}
	}
	return file, line
}
