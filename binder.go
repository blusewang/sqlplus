package sqlplus

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
)

type binder struct {
	rows   *sql.Rows
	ats    reflect.Type
	avs    reflect.Value
	item   reflect.Value
	keys   map[string]reflect.Value
	fields []interface{}
}

func (b *binder) analysisSlice(list interface{}) (err error) {
	b.ats = reflect.TypeOf(list)
	b.avs = reflect.ValueOf(list)

	if b.ats.Kind() != reflect.Ptr {
		return errors.New("传入的list必须是指针")
	}
	if b.ats.Elem().Kind() != reflect.Slice {
		return errors.New("传入的list必须是个slice")
	}
	if b.ats.Elem().Elem().Kind() != reflect.Struct {
		return errors.New("传入的list必须是struct类型的slice")
	}

	b.item = reflect.New(b.ats.Elem().Elem())
	b.keys = make(map[string]reflect.Value)
	return
}

func (b *binder) parseSlideAll() (err error) {
	cts, err := b.rows.ColumnTypes()
	if err != nil {
		return
	}

	b.decode(b.item.Elem())

	err = b.merge(cts)
	if err != nil {
		return
	}

	for b.rows.Next() {
		err = b.rows.Scan(b.fields...)
		// 记下错误，同时也赋值，不因为个别字段问题丧失所有数据
		b.avs.Elem().Set(reflect.Append(b.avs.Elem(), b.item.Elem()))
	}
	return
}

func (b *binder) analysisStruct(obj interface{}) (err error) {
	b.ats = reflect.TypeOf(obj)
	b.avs = reflect.ValueOf(obj)

	if b.ats.Kind() != reflect.Ptr {
		return errors.New("传入的 obj 必须是指针")
	}
	if b.ats.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("传入的 obj %v 必须是个 struct", b.ats.Elem().Kind())
	}

	b.item = b.avs
	b.keys = make(map[string]reflect.Value)
	return
}

func (b *binder) parseStruct() (err error) {
	cts, err := b.rows.ColumnTypes()
	if err != nil {
		return
	}

	b.decode(b.item.Elem())

	err = b.merge(cts)
	if err != nil {
		return
	}

	b.rows.Next()
	err = b.rows.Scan(b.fields...)

	return
}

func (b *binder) mustLimit1(query string) string {
	query = strings.TrimSpace(query)
	//if !strings.Contains(strings.ToLower(query), "limit") && query[len(query)-1] != 42 {
	//	query += " limit 1"
	//}
	return query
}

type jsonField struct {
	Field interface{}
}

func (jf *jsonField) Scan(src interface{}) (err error) {
	switch src.(type) {
	case string:
		err = json.Unmarshal([]byte(src.(string)), jf.Field)
	case []byte:
		err = json.Unmarshal(src.([]byte), jf.Field)
	}
	return
}

func (b *binder) merge(cts []*sql.ColumnType) (err error) {
	for _, v := range cts {
		if f := b.keys[v.Name()]; f.CanAddr() && f.Addr().CanInterface() {
			// 要先检查类型是否匹配

			if b.canScan(v, f.Type()) {
				b.fields = append(b.fields, f.Addr().Interface())
			} else {
				if v.DatabaseTypeName() == "PgTypeJsonb" || v.DatabaseTypeName() == "PgTypeJson" {
					b.fields = append(b.fields, &jsonField{f.Addr().Interface()})
				} else {
					log.Println("ParseRows type not pare -> ", v.Name(), v.DatabaseTypeName(), v.ScanType(), f.Type())
					b.fields = append(b.fields, reflect.New(v.ScanType()).Interface())
				}
			}
		} else {
			/*
				如果查询出的字段，不在struct有标记的field中，会导致Scan时数量对不上的问题
				为了补齐，需创建一个对应字段类型的变量指针
			*/
			f := reflect.New(v.ScanType()).Interface()
			b.fields = append(b.fields, &f)
		}
	}
	return
}

func (b *binder) canScan(t1 *sql.ColumnType, t2 reflect.Type) bool {
	if t1.ScanType() == t2 || "*"+t1.ScanType().String() == t2.String() {
		return true
	} else {
		if len(t1.DatabaseTypeName()) > 2 && t1.DatabaseTypeName()[0:3] == "INT" {
			return t1.ScanType().String()[0:3] == "int" && t2.String()[0:3] == "int"
		} else if t1.ScanType().String() == "time.Time" && t2.String() == "pq.NullTime" {
			return true
		} else if t1.DatabaseTypeName() == "_INT4" && t2.String() == "pq.Int64Array" {
			return true
		} else if t1.DatabaseTypeName() == "_VARCHAR" && t2.String() == "pq.StringArray" {
			return true
		} else if t1.DatabaseTypeName() == "TEXT" && t2.String() == "sql.NullString" {
			return true
		} else {
			return false
		}
	}
}

func (b *binder) decode(v reflect.Value) {
	if !v.IsValid() {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		tag := b.getTag(v.Type().Field(i).Tag)

		if tag == "" {
			if f.Kind() == reflect.Struct {
				// 没有tag的类型引用
				b.decode(f.Addr().Elem())
			}
		} else {
			/*
				只要有tag，视为解析的终点
				因为一条记录是一个线形的一维数组，不是树形结构
			*/
			if f.CanInterface() && f.CanAddr() {
				// 忽略得到了类型，也无法赋值的私有类型
				b.keys[tag] = f
			}
		}
	}
	return
}

func (b *binder) getTag(t reflect.StructTag) (tag string) {
	if tag = t.Get("sql"); tag == "" {
		if tag = t.Get("json"); tag == "" {
			tag = t.Get("xml")
		}
	}
	return
}
