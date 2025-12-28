package schema

import "app/comm"

type SurveySchema struct {
	Version      string       `json:"version" binding:"required,semver" desc:"版本号"`
	BaseConf     BaseConf     `json:"base_conf" binding:"required" desc:"基础配置"`
	QuestionConf QuestionConf `json:"question_conf" binding:"required" desc:"题目配置"`
	BannerConf   BannerConf   `json:"banner_conf" binding:"required" desc:"页头配置"`
}

type BaseConf struct {
	BeginTime       string          `json:"begin_time" binding:"required,datetime=2006-01-02 15:04:05" desc:"问卷有效期 开始时间"`
	EndTime         string          `json:"end_time" binding:"required,datetime=2006-01-02 15:04:05" desc:"问卷有效期 结束时间"`
	IsLoginRequired bool            `json:"is_login_required" desc:"是否需要登录"`
	DailyLimit      int             `json:"daily_limit" binding:"gte=0" desc:"每日提交限制 is_login_required=true时生效"`
	TotalLimit      int             `json:"total_limit" binding:"omitempty,gte=0,gtefield=DailyLimit" desc:"总提交限制 is_login_required=true时生效"`
	AllowedUserType []comm.UserType `json:"allowed_user_type" binding:"unique,dive,oneof=undergrad postgrad" desc:"允许提交的用户类型 is_login_required=true时生效"`
}

type QuestionConf struct {
	Items []QuestionItem `json:"items" binding:"required,min=1,dive" desc:"题目列表"`
}

type QuestionItem struct {
	// 基本结构
	ID    string            `json:"id" binding:"required" desc:"题目ID"`
	Type  comm.QuestionType `json:"type" binding:"required,oneof=text textarea radio checkbox vote-radio vote-checkbox upload" desc:"题型"`
	Title string            `json:"title" binding:"required" desc:"题目标题"`
	Desc  string            `json:"desc" desc:"题目描述"`

	// 题型功能
	IsRequired bool `json:"is_required" desc:"是否必填"`

	// 输入类题型
	Placeholder string       `json:"placeholder,omitempty" desc:"引导提示文案"`
	Valid       string       `json:"valid,omitempty" binding:"required_if=Type text,required_if=Type textarea,omitempty,oneof=* n m e idcard" desc:"内容格式限制 *:任意格式 n:数值格式 m:手机号格式 e:Email格式 idcard:身份证格式"`
	TextRange   *TextRange   `json:"text_range,omitempty" binding:"required_if=Valid *" desc:"文本长度区间限制 valid=*时生效"`
	Regex       string       `json:"regex,omitempty" desc:"正则表达式 valid=*时生效"`
	NumberRange *NumberRange `json:"number_range,omitempty" binding:"required_if=Valid n" desc:"数值区间限制 valid=n时生效"`

	// 选项类题型
	Options              []Option `json:"options,omitempty" binding:"required_if=Type radio,required_if=Type checkbox,required_if=Type vote-radio,required_if=Type vote-checkbox,omitempty,min=1,dive" desc:"选项列表"`
	Layout               string   `json:"layout,omitempty" binding:"omitempty,oneof=vertical horizontal" desc:"排列方式 vertical:竖排 horizontal:横排"`
	MinNum               int      `json:"min_num,omitempty" binding:"gte=0" desc:"最少选择数 type=checkbox/vote-checkbox时生效"`
	MaxNum               int      `json:"max_num,omitempty" binding:"required_if=Type checkbox,required_if=Type vote-checkbox,omitempty,gte=1,gtefield=MinNum" desc:"最多选择数 type=checkbox/vote-checkbox时生效"`
	ShowStats            bool     `json:"show_stats,omitempty" desc:"是否显示选项统计数据 type=vote-radio/vote-checkbox时生效"`
	ShowStatsAfterSubmit bool     `json:"show_stats_after_submit,omitempty" desc:"是否在提交后显示选项统计数据 type=vote-radio/vote-checkbox时生效"`
	ShowRank             bool     `json:"show_rank,omitempty" desc:"是否显示选项排名 type=vote-radio/vote-checkbox时生效"`

	// 上传类题型
	UploadType      string   `json:"upload_type,omitempty" binding:"required_if=Type upload,omitempty,oneof=file image" desc:"上传文件类型 file:文件 image:图片(jpg/jpeg/png/webp)"`
	AllowedFileType []string `json:"allowed_file_type,omitempty" binding:"unique" desc:"允许上传的文件类型 空表示不限制 upload_type=file时生效"`
	MaxFileSize     int      `json:"max_file_size,omitempty" binding:"required_if=Type upload,omitempty,gte=1,lte=100" desc:"最大上传文件大小 单位MB"`
	MaxFileNum      int      `json:"max_file_num,omitempty" binding:"required_if=Type upload,omitempty,gte=1,lte=10" desc:"最多上传文件数量"`
}

type TextRange struct {
	Min int `json:"min" binding:"gte=0" desc:"最短文本长度"`
	Max int `json:"max" binding:"gte=1,gtefield=Min" desc:"最长文本长度"`
}

type NumberRange struct {
	Min float64 `json:"min" desc:"最小值"`
	Max float64 `json:"max" binding:"gtefield=Min" desc:"最大值"`
}

type Option struct {
	ID          string `json:"id" binding:"required" desc:"选项ID"`
	Text        string `json:"text" binding:"required" desc:"选项文本"`
	Others      bool   `json:"others" desc:"是否支持自定义输入内容"`
	OthersKey   string `json:"others_key,omitempty" binding:"required_if=Others true" desc:"自定义输入内容ID others=true时生效"`
	MustOthers  bool   `json:"must_others,omitempty" desc:"自定义输入内容是否必填 others=true时生效"`
	Placeholder string `json:"placeholder,omitempty" desc:"输入提示文案 others=true时生效"`
}

type BannerConf struct {
	TitleConf TitleConf `json:"title_conf" binding:"required" desc:"标题配置"`
}

type TitleConf struct {
	MainTitle string `json:"main_title" binding:"required" desc:"主标题"`
	SubTitle  string `json:"sub_title" desc:"页头文案"`
}
