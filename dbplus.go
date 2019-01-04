package sqlplus

import "database/sql"

type DbPlus struct {
	*sql.DB
}

func Open(driverName, dataSourceName string) (*DbPlus, error) {
	db := &DbPlus{}
	var err error
	db.DB, err = sql.Open(driverName, dataSourceName)
	return db, err
}

func (db DbPlus) QueryStruct(obj interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisStruct(obj)
	if err != nil {
		return
	}

	b.rows, err = db.Query(b.mustLimit1(query), args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseStruct()
	if err != nil {
		return
	}

	return
}

func (db DbPlus) QuerySlice(list interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisSlice(list)
	if err != nil {
		return
	}

	b.rows, err = db.Query(query, args...)
	if err != nil {
		return
	}
	defer b.rows.Close()

	err = b.parseSlideAll()
	if err != nil {
		return
	}

	return
}

func (db DbPlus) Begin() (*TxPlus, error) {
	tx := &TxPlus{}
	var err error
	tx.Tx, err = db.DB.Begin()
	return tx, err
}
