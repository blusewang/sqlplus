package sqlplus

import (
	"database/sql"
	"log"
	"reflect"
	"strings"
)

type bindTool struct{}

func (bt bindTool) mustLimit1(query string) string {
	if !strings.Contains(strings.ToLower(query), "limit") {
		query += " limit 1"
	}
	return query
}

func (bt bindTool) merge(cts []*sql.ColumnType, keys map[string]reflect.Value) (fields []interface{}, err error) {
	for _, v := range cts {
		if f := keys[v.Name()]; f.CanAddr() && f.Addr().CanInterface() {
			// 要先检查类型是否匹配
			if bt.canScan(v.ScanType(), f.Type()) {
				fields = append(fields, f.Addr().Interface())
			} else {
				log.Println("ParseRows type not pare -> ", v.Name(), v.DatabaseTypeName(), v.ScanType(), f.Type())
				fields = append(fields, reflect.New(v.ScanType()).Interface())
			}
		} else {
			/*
				如果查询出的字段，不在struct有标记的field中，会导致Scan时数量对不上的问题
				为了补齐，需创建一个对应字段类型的变量指针
			*/
			fields = append(fields, reflect.New(v.ScanType()).Interface())
		}
	}
	return
}

func (bt bindTool) canScan(t1 reflect.Type, t2 reflect.Type) bool {
	if t1 == t2 {
		return true
	} else {
		if t1.String()[0:3] == "int" {
			return t1.String()[0:3] == "int" && t2.String()[0:3] == "int"
		} else if t1.String() == "time.Time" && t2.String() == "pq.NullTime" {
			return true
		} else {
			return false
		}
	}
}

func (bt bindTool) decode(v reflect.Value) (keys map[string]reflect.Value) {
	keys = make(map[string]reflect.Value)
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		tag := bt.getTag(v.Type().Field(i).Tag)

		if tag == "" {
			if f.Kind() == reflect.Struct {
				// 没有tag的类型引用
				bt.decode(f.Addr().Elem())
			}
		} else {
			/*
				只要有tag，视为解析的终点
				因为一条记录是一个线形的一维数组，不是树形结构
			*/
			if f.CanInterface() && f.CanAddr() {
				// 忽略得到了类型，也无法赋值的私有类型
				keys[tag] = f
			}
		}
	}
	return
}

func (bt bindTool) getTag(t reflect.StructTag) (tag string) {
	if tag = t.Get("sql"); tag == "" {
		if tag = t.Get("json"); tag == "" {
			tag = t.Get("xml")
		}
	}
	return
}
