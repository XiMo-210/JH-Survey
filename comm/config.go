package comm

// BizConf 业务配置
var BizConf *BizConfig

type BizConfig struct {
	AdminCreateSecret string `mapstructure:"admin_create_secret"` // 创建管理员密钥
}
