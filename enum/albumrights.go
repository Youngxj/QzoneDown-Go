package enum

// 定义枚举值
const (
	EveryoneVisible         = 1 // 所有人可见
	QQFriendsVisible        = 4 // QQ好友可见
	AnswerQuestionVisible   = 5 // 回答问题可见
	PartialFriendsVisible   = 6 // 部分好友可见
	PartialFriendsInvisible = 8 // 部分好友不可见【新】
	OnlySelfVisible         = 3 // 仅自己可见
)

// 定义一个映射表，将枚举值映射到描述
var rightsMap = map[int]string{
	EveryoneVisible:         "所有人可见",
	QQFriendsVisible:        "QQ好友可见",
	AnswerQuestionVisible:   "回答问题可见",
	PartialFriendsVisible:   "部分好友可见",
	PartialFriendsInvisible: "部分好友不可见【新】",
	OnlySelfVisible:         "仅自己可见",
}

// ConvertRightsEnum 转换方法
//
//	@param rights
//	@return string
//	@return bool
func ConvertRightsEnum(rights int) (string, bool) {
	description, ok := rightsMap[rights]
	return description, ok
}
