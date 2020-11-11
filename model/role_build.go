package model

type RoleBuild struct {
	Id			int			`json:"id" xorm:"id pk autoincr"`
	RId			int			`json:"rid" xorm:"rid"`
	Type		int8		`json:"type"`
	Name		string		`json:"name"`
	Wood		int			`json:"Wood"`
	Iron		int			`json:"iron"`
	Stone		int			`json:"stone"`
	Grain		int			`json:"grain"`
	Durable		int			`json:"durable"`
	Defender	int			`json:"defender"`
}

func (this *RoleBuild) TableName() string {
	return "role_build"
}