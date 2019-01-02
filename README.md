# sqlplus
golang databse/sql 的通用扩展

[![GoDoc](https://godoc.org/github.com/blusewang/sqlplus?status.svg)](https://godoc.org/github.com/blusewang/sqlplus)

## 安装

	go get github.com/blusewang/sqlplus

## 文档
详细文档，请前往 <https://godoc.org/github.com/blusewang/sqlplus>.

## 使用

```go
type TestObj struct {
	Id string `json:"id"`
	UserName string `json:"id"`
}

db,err := sqlplus.Open("postgres","dsn...")
if err != nil {
	log.Fatal(err)
}

var list []TestObj
err = db.QuerySlice(&list,"select * from test_table where id < $1",100)
if err != nil {
	log.Fatal(err)
}
log.Pringln(list)
// [{3,""},{4,""}]
```