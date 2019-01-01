package sqlplus

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type binder struct {
	bindTool
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

	b.keys = b.decode(b.item.Elem())

	b.fields, err = b.merge(cts, b.keys)
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

	b.keys = b.decode(b.item.Elem())

	b.fields, err = b.merge(cts, b.keys)
	if err != nil {
		return
	}

	b.rows.Next()
	err = b.rows.Scan(b.fields...)

	return
}
