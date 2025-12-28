package comm

type UserType string

const (
	UserTypeUndergrad UserType = "undergrad" // 本科生
	UserTypePostgrad  UserType = "postgrad"  // 研究生
)

type AdminType int8

const (
	AdminTypeNormal AdminType = 1 // 普通管理员
	AdminTypeSuper  AdminType = 2 // 超级管理员
)

type SurveyType int8

const (
	SurveyTypeSurvey SurveyType = 1 // 问卷
	SurveyTypeVote   SurveyType = 2 // 投票
)

type SurveyStatus int8

const (
	SurveyStatusUnpublished SurveyStatus = 1 // 未发布
	SurveyStatusPublished   SurveyStatus = 2 // 已发布
)

type QuestionType string

const (
	QuestionTypeText         QuestionType = "text"          // 单行输入
	QuestionTypeTextArea     QuestionType = "textarea"      // 多行输入
	QuestionTypeRadio        QuestionType = "radio"         // 单项选择
	QuestionTypeCheckbox     QuestionType = "checkbox"      // 多项选择
	QuestionTypeVoteRadio    QuestionType = "vote-radio"    // 投票-单选
	QuestionTypeVoteCheckbox QuestionType = "vote-checkbox" // 投票-多选
	QuestionTypeUpload       QuestionType = "upload"        // 上传
)
