package enum

// 定义一个映射表，将枚举值映射到描述
var tagMap = map[int]string{
	1: "个性",
	5: "亲子",
	6: "旅游",
	7: "校友",
}

// convertTagEnum 主题类型转换
//
//	@param value
//	@return string
//	@return bool
func convertTagEnum(value int) (string, bool) {
	description, ok := tagMap[value]
	return description, ok
}
