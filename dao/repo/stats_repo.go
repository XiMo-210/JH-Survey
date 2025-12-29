package repo

import (
	"context"

	"github.com/zjutjh/mygo/ndb"

	"app/dao/model"
	"app/dao/query"
)

type StatsRepo struct {
	query *query.Query
}

func NewStatsRepo(tx ...*query.Query) *StatsRepo {
	var q *query.Query
	if len(tx) > 0 {
		q = tx[0]
	} else {
		q = query.Use(ndb.Pick())
	}
	return &StatsRepo{
		query: q,
	}
}

func (r *StatsRepo) BatchCreate(ctx context.Context, records []*model.Stats) error {
	s := r.query.Stats
	return s.WithContext(ctx).CreateInBatches(records, 100)
}
