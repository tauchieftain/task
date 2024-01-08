package proto

type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

const (
	UserAdmin         = 1
	RoleAdmin         = 1
	TimeLayout        = "2006-01-02 15:04:05"
	TimeMicroLayout   = "2006-01-02 15:04:05.000"
	LogPathTimeLayout = "2006-01-02"
	NodeAliveTime     = 60
)

type UserRole struct {
	RoleID uint `json:"role_id"`
}

type EmptyArgs struct{}

type EmptyReply struct{}

type IdsArgs struct {
	IDS uint `json:"ids"`
}

type IdArgs struct {
	ID uint `json:"id"`
}

type DingTalkNoticeArgs struct {
	Address []string
	Body    string
}

type ConfigListArgs struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Pagination
}

type ConfigAddArgs struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ConfigEditArgs struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}
