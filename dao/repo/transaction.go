package repo

import (
	"github.com/zjutjh/mygo/ndb"

	"app/dao/query"
)

// Transaction 事务处理
func Transaction(fc func(tx *query.Query) error) error {
	return query.Use(ndb.Pick()).Transaction(fc)
}
