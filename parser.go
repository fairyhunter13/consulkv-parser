package consulparser

import (
	"reflect"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)

type ParserIface interface {
	Parse(interface{}) error
}

//Parser defines struct for the parser API.
type Parser struct {
	consulKV *api.KV
}

const (
	keyTag   = "consulkv"
	timeType = "time.Time"
)

var (
	timeLayout = time.RFC3339
)

//NewParser initialize a new parser with the supplied consul client.
func NewParser(client *api.Client) (parser ParserIface, err error) {
	if client == nil {
		return nil, ErrNilClient
	}
	parser = &Parser{
		consulKV: client.KV(),
	}
	return
}

//Parse gives the value to the target from the consul server.
//Parse uses the struct tag to identify the value of the key.
func (parser *Parser) Parse(target interface{}) (err error) {
	valueStruct := reflect.ValueOf(target)
	if valueStruct.Kind() != reflect.Ptr || !valueStruct.IsValid() {
		return ErrNonPointerType
	}
	//Start as empty value first.
	//This is acceptable to check the target struct first.
	err = parser.assign(parser.getRecursivePointerVal(valueStruct), "")
	return
}

func (parser *Parser) getRecursivePointerVal(val reflect.Value) (elemVal reflect.Value) {
	elemVal = val
	if val.Kind() == reflect.Ptr {
		elemVal = parser.getRecursivePointerVal(elemVal.Elem())
	}
	return
}

func (parser *Parser) parse(v reflect.Value) (err error) {
	var value string
	typeV := v.Type()
	for index := 0; index < v.NumField(); index++ {
		field := v.Field(index)
		if !field.CanSet() || !field.IsValid() {
			continue
		}
		consulKey := typeV.Field(index).Tag.Get(keyTag)
		value, err = parser.getValue(consulKey)
		if err != nil {
			return
		}
		err = parser.assign(field, value)
		if err != nil {
			return
		}
	}
	return
}

func (parser *Parser) assign(val reflect.Value, value string) (err error) {
	switch val.Kind() {
	case reflect.Ptr:
		err = parser.assignPointer(val, value)
	default:
		err = parser.assignNonPointer(val, value)
	}
	return
}

func (parser *Parser) assignPointer(val reflect.Value, value string) (err error) {
	var tempVal reflect.Value
	switch val.Type().Elem().Kind() {
	case reflect.Ptr:
		tempVal = reflect.New(val.Type().Elem())
		err = parser.assignPointer(tempVal.Elem(), value)
		if err != nil {
			return
		}
	case reflect.Struct:
		if val.Type().Elem().String() == timeType {
			if value == "" {
				return
			}
			var timeVal time.Time
			//Using time.RFC3339 as the layout
			timeVal, err = time.Parse(timeLayout, value)
			if err != nil {
				return
			}
			tempVal = reflect.ValueOf(&timeVal)
		} else {
			tempVal = reflect.New(val.Type().Elem())
			err = parser.parse(tempVal.Elem())
			if err != nil {
				return
			}
		}
	case reflect.Interface, reflect.String:
		if value == "" {
			return
		}
		tempVal = reflect.New(val.Type().Elem())
		tempVal.Elem().Set(reflect.ValueOf(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return
		}
		var temp int64
		temp, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return
		}
		tempVal = reflect.New(val.Type().Elem())
		if tempVal.Elem().OverflowInt(temp) {
			err = ErrOverflowSet
			return
		}
		tempVal.Elem().Set(reflect.ValueOf(temp))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if value == "" {
			return
		}
		var temp uint64
		temp, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return
		}
		tempVal = reflect.New(val.Type().Elem())
		if tempVal.Elem().OverflowUint(temp) {
			err = ErrOverflowSet
			return
		}
		tempVal.Elem().Set(reflect.ValueOf(temp))
	case reflect.Float32, reflect.Float64:
		if value == "" {
			return
		}
		var temp float64
		temp, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return
		}
		tempVal = reflect.New(val.Type().Elem())
		if tempVal.Elem().OverflowFloat(temp) {
			err = ErrOverflowSet
			return
		}
		tempVal.Elem().Set(reflect.ValueOf(temp))
	case reflect.Bool:
		if value == "" {
			return
		}
		var temp bool
		temp, err = strconv.ParseBool(value)
		if err != nil {
			return
		}
		tempVal = reflect.New(val.Type().Elem())
		tempVal.Elem().Set(reflect.ValueOf(temp))
	default:
		err = ErrUnhandledKind
		return
	}
	val.Set(tempVal)
	return
}

func (parser *Parser) assignNonPointer(val reflect.Value, value string) (err error) {
	switch val.Kind() {
	case reflect.Struct:
		if val.Type().String() == timeType {
			if value == "" {
				return
			}
			var timeVal time.Time
			//Using time.RFC3339 layout only
			timeVal, err = time.Parse(timeLayout, value)
			if err != nil {
				return
			}
			val.Set(reflect.ValueOf(timeVal))
		} else {
			err = parser.parse(val)
		}
	case reflect.Interface, reflect.String:
		if value == "" {
			return
		}
		val.Set(reflect.ValueOf(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if value == "" {
			return
		}
		var temp int64
		temp, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return
		}
		if val.OverflowInt(temp) {
			err = ErrOverflowSet
			return
		}
		val.SetInt(temp)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if value == "" {
			return
		}
		var temp uint64
		temp, err = strconv.ParseUint(value, 10, 64)
		if err != nil {
			return
		}
		if val.OverflowUint(temp) {
			err = ErrOverflowSet
			return
		}
		val.SetUint(temp)
	case reflect.Float32, reflect.Float64:
		if value == "" {
			return
		}
		var temp float64
		temp, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return
		}
		if val.OverflowFloat(temp) {
			err = ErrOverflowSet
			return
		}
		val.SetFloat(temp)
	case reflect.Bool:
		if value == "" {
			return
		}
		var temp bool
		temp, err = strconv.ParseBool(value)
		if err != nil {
			return
		}
		val.SetBool(temp)
	default:
		err = ErrUnhandledKind
	}
	return
}

func (parser *Parser) getValue(consulKey string) (value string, err error) {
	if consulKey == "" {
		return
	}
	pair, _, err := parser.consulKV.Get(consulKey, nil)
	if err != nil {
		return
	}
	value = string(pair.Value)
	return
}

func (parser *Parser) SetTimeLayout(layout string) (err error) {
	if layout == "" {
		err = ErrEmptyLayout
		return
	}
	timeLayout = layout
	return
}
