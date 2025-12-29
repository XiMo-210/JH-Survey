package survey

import (
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"github.com/zjutjh/mygo/foundation/reply"
	"github.com/zjutjh/mygo/jwt"
	"github.com/zjutjh/mygo/kit"
	"github.com/zjutjh/mygo/nlog"
	"github.com/zjutjh/mygo/swagger"

	"app/comm"
	"app/dao/model"
	"app/dao/query"
	"app/dao/repo"
	"app/schema"
)

var (
	regexMobile = regexp.MustCompile(`^1[3-9]\d{9}$`)
	regexEmail  = regexp.MustCompile(`^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`)
	regexIDCard = regexp.MustCompile(`(^\d{15}$)|(^\d{18}$)|(^\d{17}(\d|X|x)$)`)
)

// SubmitHandler API router注册点
func SubmitHandler() gin.HandlerFunc {
	api := SubmitApi{}
	swagger.CM[runtime.FuncForPC(reflect.ValueOf(hfSubmit).Pointer()).Name()] = api
	return hfSubmit
}

type SubmitApi struct {
	Info     struct{}          `name:"提交问卷" desc:"提交问卷"`
	Request  SubmitApiRequest  // API请求参数 (Uri/Header/Query/Body)
	Response SubmitApiResponse // API响应数据 (Body中的Data部分)
}

type SubmitApiRequest struct {
	Body struct {
		ID     int64             `json:"id" binding:"required,gte=1" desc:"问卷ID"`
		Result []comm.ResultItem `json:"result" binding:"required,min=1" desc:"答卷结果"`
	}
}

type SubmitApiResponse struct{}

// Run Api业务逻辑执行点
func (s *SubmitApi) Run(ctx *gin.Context) kit.Code {
	req := s.Request.Body

	// 查询问卷
	survey, err := repo.NewSurveyRepo().FindByID(ctx, req.ID)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷失败")
		return comm.CodeDatabaseError
	}
	if survey == nil {
		return comm.CodeDataNotFound
	}

	// 检查问卷状态
	if comm.SurveyStatus(survey.Status) != comm.SurveyStatusPublished {
		return comm.CodeDataNotFound
	}

	// 问卷结构反序列化
	var surveySchema schema.SurveySchema
	if err := sonic.UnmarshalString(survey.Schema, &surveySchema); err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("问卷结构反序列化失败")
		return comm.CodeDataParseError
	}

	// 检查问卷时间有效期
	now := time.Now()
	beginTime, _ := time.ParseInLocation(time.DateTime, surveySchema.BaseConf.BeginTime, time.Local)
	endTime, _ := time.ParseInLocation(time.DateTime, surveySchema.BaseConf.EndTime, time.Local)
	if now.Before(beginTime) || now.After(endTime) {
		return comm.CodeSurveyTimeInvalid
	}

	// 检查登录及提交限制
	var username string
	if surveySchema.BaseConf.IsLoginRequired {
		// 获取登录用户信息
		user, err := jwt.GetIdentity[comm.UserIdentity](ctx)
		if err != nil {
			return comm.CodeNotLoggedIn
		}
		username = user.Username

		// 检查用户类型
		if len(surveySchema.BaseConf.AllowedUserType) > 0 {
			if !lo.Contains(surveySchema.BaseConf.AllowedUserType, user.Type) {
				return comm.CodePermissionDenied
			}
		}

		// 检查总提交限制
		if surveySchema.BaseConf.TotalLimit > 0 {
			count, err := repo.NewResultRepo().CountByUser(ctx, survey.ID, username, nil)
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷总提交次数失败")
				return comm.CodeDatabaseError
			}
			if count >= surveySchema.BaseConf.TotalLimit {
				return comm.CodeSurveySubmitLimit
			}
		}

		// 检查每日提交限制
		if surveySchema.BaseConf.DailyLimit > 0 {
			start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			end := start.Add(24 * time.Hour)
			count, err := repo.NewResultRepo().CountByUser(ctx, survey.ID, username, &repo.TimeRange{
				Start: start,
				End:   end,
			})
			if err != nil {
				nlog.Pick().WithContext(ctx).WithError(err).Error("查询问卷今日提交次数失败")
				return comm.CodeDatabaseError
			}
			if count >= surveySchema.BaseConf.DailyLimit {
				return comm.CodeSurveySubmitLimit
			}
		}
	}

	// 答卷结果校验
	answerMap := lo.SliceToMap(req.Result, func(item comm.ResultItem) (string, string) {
		return item.QuestionID, item.Answer
	})
	statsUpdates := make([]repo.StatsUpdate, 0)
	for _, item := range surveySchema.QuestionConf.Items {
		val, exists := answerMap[item.ID]

		// 检查必填
		if item.IsRequired && (!exists || val == "") {
			nlog.Pick().WithContext(ctx).Warnf("必填项未填 ID:%s", item.ID)
			return comm.CodeParameterInvalid
		}

		if !exists || val == "" {
			continue
		}

		// 选项类题型
		if item.IsOptionType() {
			selectedOpts := strings.Split(val, ",")

			// 多选题校验选项数量
			if item.IsCheckboxType() {
				if (item.MinNum > 0 && len(selectedOpts) < item.MinNum) ||
					(item.MaxNum > 0 && len(selectedOpts) > item.MaxNum) {
					nlog.Pick().WithContext(ctx).Warnf("选项数量不符合要求 ID:%s", item.ID)
					return comm.CodeParameterInvalid
				}
			}

			// 校验选项是否存在
			optMap := lo.KeyBy(item.Options, func(o schema.Option) string {
				return o.ID
			})
			for _, optID := range selectedOpts {
				opt, ok := optMap[optID]
				if !ok {
					nlog.Pick().WithContext(ctx).Warnf("选项不存在 ID:%s OptID:%s", item.ID, optID)
					return comm.CodeParameterInvalid
				}

				// 收集统计数据
				statsUpdates = append(statsUpdates, repo.StatsUpdate{
					QuestionID: item.ID,
					OptionID:   optID,
				})

				// 检查自定义输入内容选项
				if opt.Others {
					othersVal, ok := answerMap[opt.OthersKey]
					if opt.MustOthers && (!ok || othersVal == "") {
						nlog.Pick().WithContext(ctx).Warnf("自定义输入内容选项必填未填 ID:%s OptID:%s", item.ID, optID)
						return comm.CodeParameterInvalid
					}
				}
			}
		} else if item.IsInputType() {
			switch item.Valid {
			case "n": // 数字校验及范围检查
				valDec, err := decimal.NewFromString(val)
				if err != nil {
					nlog.Pick().WithContext(ctx).Warnf("数字格式错误 ID:%s Val:%s", item.ID, val)
					return comm.CodeParameterInvalid
				}
				if item.NumberRange != nil {
					minDec, _ := decimal.NewFromString(item.NumberRange.Min)
					maxDec, _ := decimal.NewFromString(item.NumberRange.Max)
					if valDec.LessThan(minDec) || valDec.GreaterThan(maxDec) {
						nlog.Pick().WithContext(ctx).Warnf("数字超出范围 ID:%s Val:%s", item.ID, val)
						return comm.CodeParameterInvalid
					}
				}
			case "m": // 手机号校验
				if !regexMobile.MatchString(val) {
					nlog.Pick().WithContext(ctx).Warnf("手机号格式错误 ID:%s Val:%s", item.ID, val)
					return comm.CodeParameterInvalid
				}
			case "e": // 邮箱校验
				if !regexEmail.MatchString(val) {
					nlog.Pick().WithContext(ctx).Warnf("邮箱格式错误 ID:%s Val:%s", item.ID, val)
					return comm.CodeParameterInvalid
				}
			case "idcard": // 身份证号校验
				if !regexIDCard.MatchString(val) {
					nlog.Pick().WithContext(ctx).Warnf("身份证号格式错误 ID:%s Val:%s", item.ID, val)
					return comm.CodeParameterInvalid
				}
			default: // 普通文本校验
				if item.TextRange != nil {
					l := len([]rune(val))
					if l < item.TextRange.Min || l > item.TextRange.Max {
						nlog.Pick().WithContext(ctx).Warnf("文本长度超出范围 ID:%s Val:%s", item.ID, val)
						return comm.CodeParameterInvalid
					}
				}
				if item.Regex != "" {
					if match, _ := regexp.MatchString(item.Regex, val); !match {
						nlog.Pick().WithContext(ctx).Warnf("文本格式错误 ID:%s Val:%s", item.ID, val)
						return comm.CodeParameterInvalid
					}
				}
			}
		} else if item.IsUploadType() {
			files := strings.Split(val, ",")
			if item.MaxFileNum > 0 && len(files) > item.MaxFileNum {
				nlog.Pick().WithContext(ctx).Warnf("上传文件数量超出限制 ID:%s", item.ID)
				return comm.CodeParameterInvalid
			}

			for _, f := range files {
				ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f), "."))
				switch item.UploadType {
				case "image":
					if !slices.Contains([]string{"jpg", "jpeg", "png", "webp"}, ext) {
						nlog.Pick().WithContext(ctx).Warnf("上传图片格式错误 ID:%s Ext:%s", item.ID, ext)
						return comm.CodeParameterInvalid
					}
				case "file":
					if len(item.AllowedFileType) > 0 && !slices.Contains(item.AllowedFileType, ext) {
						nlog.Pick().WithContext(ctx).Warnf("上传文件格式错误 ID:%s Ext:%s", item.ID, ext)
						return comm.CodeParameterInvalid
					}
				}
			}
		}
	}

	// 答卷结果序列化
	data, err := sonic.MarshalString(req.Result)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("答卷结果序列化失败")
		return comm.CodeDataParseError
	}

	// 排序 避免死锁
	slices.SortFunc(statsUpdates, func(a, b repo.StatsUpdate) int {
		if c := strings.Compare(a.QuestionID, b.QuestionID); c != 0 {
			return c
		}
		return strings.Compare(a.OptionID, b.OptionID)
	})

	// 事务 创建答卷 -> 更新统计数据
	err = repo.Transaction(func(tx *query.Query) error {
		// 创建答卷
		if err := repo.NewResultRepo(tx).Create(ctx, &model.Result{
			Username: username,
			SurveyID: survey.ID,
			Data:     data,
		}); err != nil {
			return err
		}

		// 更新统计数据
		if len(statsUpdates) > 0 {
			if _, err := repo.NewStatsRepo(tx).BatchIncr(ctx, survey.ID, statsUpdates); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Error("提交问卷失败")
		return comm.CodeDatabaseError
	}

	return comm.CodeOK
}

// Init Api初始化 进行参数校验和绑定
func (s *SubmitApi) Init(ctx *gin.Context) (err error) {
	err = ctx.ShouldBindJSON(&s.Request.Body)
	if err != nil {
		return err
	}
	return err
}

// hfSubmit API执行入口
func hfSubmit(ctx *gin.Context) {
	api := &SubmitApi{}
	err := api.Init(ctx)
	if err != nil {
		nlog.Pick().WithContext(ctx).WithError(err).Warn("参数绑定校验错误")
		reply.Fail(ctx, comm.CodeParameterInvalid)
		return
	}
	code := api.Run(ctx)
	if !ctx.IsAborted() {
		if code == comm.CodeOK {
			reply.Success(ctx, api.Response)
		} else {
			reply.Fail(ctx, code)
		}
	}
}
