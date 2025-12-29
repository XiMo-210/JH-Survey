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

func (r *StatsRepo) FindListBySurveyID(ctx context.Context, surveyID int64) ([]*model.Stats, error) {
	s := r.query.Stats
	return s.WithContext(ctx).Where(s.SurveyID.Eq(surveyID)).Find()
}

func (r *StatsRepo) BatchCreate(ctx context.Context, records []*model.Stats) error {
	s := r.query.Stats
	return s.WithContext(ctx).CreateInBatches(records, 100)
}

type StatsUpdate struct {
	QuestionID string
	OptionID   string
}

func (r *StatsRepo) BatchIncr(ctx context.Context, surveyID int64, updates []StatsUpdate) (int64, error) {
	s := r.query.Stats
	do := s.WithContext(ctx)
	var conds query.IStatsDo
	for _, u := range updates {
		condition := do.Where(s.QuestionID.Eq(u.QuestionID), s.OptionID.Eq(u.OptionID))
		if conds == nil {
			conds = condition
		} else {
			conds = conds.Or(condition)
		}
	}
	// WHERE survey_id = ? AND ((question_id= ? AND option_id= ?) OR (question_id= ? AND option_id = ?) ...)
	res, err := do.Where(s.SurveyID.Eq(surveyID)).Where(conds).UpdateSimple(s.Count.Add(1))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}
