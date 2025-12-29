package schema

import (
	"fmt"
	"regexp"
	"time"

	"app/comm"
)

// NormalizeAndVerify 处理默认值填充、字段清理和深度业务校验
func (s *SurveySchema) NormalizeAndVerify() error {
	// BaseConf
	if err := s.BaseConf.verifyAndFix(); err != nil {
		return fmt.Errorf("base_conf error: %w", err)
	}

	// QuestionConf
	if err := s.QuestionConf.verifyAndFix(); err != nil {
		return fmt.Errorf("question_conf error: %w", err)
	}

	return nil
}

func (b *BaseConf) verifyAndFix() error {
	beginTime, _ := time.Parse(time.DateTime, b.BeginTime)
	endTime, _ := time.Parse(time.DateTime, b.EndTime)
	if !endTime.After(beginTime) {
		return fmt.Errorf("end_time must be after begin_time")
	}

	if !b.IsLoginRequired {
		b.DailyLimit = 0
		b.TotalLimit = 0
		b.AllowedUserType = nil
	}

	return nil
}

func (q *QuestionConf) verifyAndFix() error {
	ids := make(map[string]bool)
	for i := range q.Items {
		item := &q.Items[i]
		if ids[item.ID] {
			return fmt.Errorf("duplicate question id: %s", item.ID)
		}
		ids[item.ID] = true

		if err := item.verifyAndFix(); err != nil {
			return fmt.Errorf("question(id=%s) error: %w", item.ID, err)
		}
	}

	return nil
}

func (item *QuestionItem) verifyAndFix() error {
	if !item.IsInputType() {
		item.Placeholder = ""
		item.Valid = ""
		item.TextRange = nil
		item.Regex = ""
		item.NumberRange = nil
	} else {
		if item.Valid != "*" {
			item.TextRange = nil
			item.Regex = ""
		} else {
			if item.Regex != "" {
				_, err := regexp.Compile(item.Regex)
				if err != nil {
					return fmt.Errorf("invalid regex pattern: %w", err)
				}
			}
		}
		if item.Valid != "n" {
			item.NumberRange = nil
		}
	}

	if !item.IsOptionType() {
		item.Options = nil
		item.Layout = ""
		item.MinNum = 0
		item.MaxNum = 0
		item.ShowStats = false
		item.ShowStatsAfterSubmit = false
		item.ShowRank = false
	} else {
		if item.IsCheckboxType() {
			if len(item.Options) < item.MinNum {
				return fmt.Errorf("min_num cannot be greater than the number of options")
			}
			if len(item.Options) < item.MaxNum {
				return fmt.Errorf("max_num cannot be greater than the number of options")
			}
		} else {
			item.MinNum = 0
			item.MaxNum = 0
		}

		if !item.IsVoteType() {
			item.ShowStats = false
			item.ShowStatsAfterSubmit = false
			item.ShowRank = false
		}

		optionIds := make(map[string]bool)
		for i := range item.Options {
			opt := &item.Options[i]
			if optionIds[opt.ID] {
				return fmt.Errorf("duplicate option id: %s", opt.ID)
			}
			optionIds[opt.ID] = true

			if !opt.Others {
				opt.OthersKey = ""
				opt.MustOthers = false
				opt.Placeholder = ""
			} else {
				if optionIds[opt.OthersKey] {
					return fmt.Errorf("duplicate option id (others_key): %s", opt.OthersKey)
				}
				optionIds[opt.OthersKey] = true
			}
		}

		if item.Layout == "" {
			item.Layout = "vertical"
		}
	}

	if !item.IsUploadType() {
		item.UploadType = ""
		item.AllowedFileType = nil
		item.MaxFileSize = 0
		item.MaxFileNum = 0
	} else {
		if item.UploadType == "image" {
			item.AllowedFileType = nil
		}
	}

	return nil
}

func (item *QuestionItem) IsInputType() bool {
	return item.Type == comm.QuestionTypeText || item.Type == comm.QuestionTypeTextArea
}

func (item *QuestionItem) IsOptionType() bool {
	return item.Type == comm.QuestionTypeRadio || item.Type == comm.QuestionTypeCheckbox ||
		item.Type == comm.QuestionTypeVoteRadio || item.Type == comm.QuestionTypeVoteCheckbox
}

func (item *QuestionItem) IsVoteType() bool {
	return item.Type == comm.QuestionTypeVoteRadio || item.Type == comm.QuestionTypeVoteCheckbox
}

func (item *QuestionItem) IsCheckboxType() bool {
	return item.Type == comm.QuestionTypeCheckbox || item.Type == comm.QuestionTypeVoteCheckbox
}

func (item *QuestionItem) IsUploadType() bool {
	return item.Type == comm.QuestionTypeUpload
}

func (item *QuestionItem) GetCategory() string {
	if item.IsInputType() {
		return "input"
	}
	if item.IsOptionType() {
		return "option"
	}
	if item.IsUploadType() {
		return "upload"
	}
	return "unknown"
}
