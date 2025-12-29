package repo

import (
	"context"
	"time"

	"github.com/zjutjh/mygo/ndb"

	"app/dao/model"
	"app/dao/query"
)

type ResultRepo struct {
	query *query.Query
}

func NewResultRepo(tx ...*query.Query) *ResultRepo {
	var q *query.Query
	if len(tx) > 0 {
		q = tx[0]
	} else {
		q = query.Use(ndb.Pick())
	}
	return &ResultRepo{
		query: q,
	}
}

func (r *ResultRepo) FindPage(ctx context.Context, surveyID int64, page, pageSize int) ([]*model.Result, int64, error) {
	q := r.query.Result
	do := q.WithContext(ctx).Where(q.SurveyID.Eq(surveyID))

	list, err := do.Order(q.ID.Desc()).Limit(pageSize).Offset((page - 1) * pageSize).Find()
	if err != nil {
		return nil, 0, err
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *ResultRepo) CountBySurveyID(ctx context.Context, surveyID int64) (int64, error) {
	q := r.query.Result
	return q.WithContext(ctx).Where(q.SurveyID.Eq(surveyID)).Count()
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func (r *ResultRepo) CountByUser(ctx context.Context, sid int64, user string, tr *TimeRange) (int64, error) {
	q := r.query.Result
	db := q.WithContext(ctx).Where(q.SurveyID.Eq(sid), q.Username.Eq(user))
	if tr != nil {
		db = db.Where(q.CreatedAt.Between(tr.Start, tr.End))
	}
	return db.Count()
}

func (r *ResultRepo) Create(ctx context.Context, result *model.Result) error {
	res := r.query.Result
	return res.WithContext(ctx).Create(result)
}
