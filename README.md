# sqlplus
golang databse/sql 的通用扩展

[![GoDoc](https://godoc.org/github.com/blusewang/sqlplus?status.svg)](https://godoc.org/github.com/blusewang/sqlplus)
[![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/blusewang/sqlplus/blob/master/LICENSE)

## 安装

	go get github.com/blusewang/sqlplus

## 文档
详细文档，请前往 <https://godoc.org/github.com/blusewang/sqlplus>.

## 使用

```go
type TestObj struct {
	Id       string `json:"id"`
	UserName string `json:"user_name"`
}

db,err := sqlplus.Open("postgres","dsn...")
if err != nil {
	log.Fatal(err)
}

// 查列表
var list []TestObj
err = db.QuerySlice(&list,"select * from test_table where id < $1",100)
if err != nil {
	log.Fatal(err)
}
log.Pringln(list)
// [{3,""},{4,""}]

// 查单行
var obj TestObj
err = db.QueryStruct(&obj,"select * from test_table where id=$1",3)
if err != nil {
	log.Fatal(err)
}
log.Pringln(obj)
// {3,""}


// 事务
tx,err := db.Begin()
err = tx.QueryStruct(&obj,"select * from test_table where id=$1",3)
if err != nil {
	_ = tx.Rollback()
	log.Fatal(err)
}else{
	_ = tx.Commit()
}
log.Pringln(obj)
// {3,""}
```