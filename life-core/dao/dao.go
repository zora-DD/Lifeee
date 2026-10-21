// Package dao 封装项目的数据访问操作。
package dao

import (
	"errors"

	"gorm.io/gorm"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

var errNilDB = errors.New("dao: nil db")

func checkDB(db *gorm.DB) error {
	if db == nil {
		return errNilDB
	}
	return nil
}

func pageLimit(page, pageSize int) (offset, limit int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return (page - 1) * pageSize, pageSize
}
