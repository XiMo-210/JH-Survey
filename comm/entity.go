package comm

type UserIdentity struct {
	Username string   `json:"username" desc:"用户名"`
	Type     UserType `json:"type" desc:"用户类型"`
}

type AdminIdentity struct {
	Username string    `json:"username" desc:"用户名"`
	Type     AdminType `json:"type" desc:"用户类型"`
}
