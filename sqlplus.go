package sqlplus

import "database/sql"

type SqlPlus struct {
	*sql.DB
}


func (pg SqlPlus) QueryStruct(obj interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisStruct(obj)
	if err != nil {
		return
	}

	b.rows, err = pg.Query(b.mustLimit1(query), args...)
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

func (pg SqlPlus) QuerySlice(list interface{}, query string, args ...interface{}) (err error) {
	var b binder

	err = b.analysisSlice(list)
	if err != nil {
		return
	}

	b.rows, err = pg.Query(query, args...)
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
