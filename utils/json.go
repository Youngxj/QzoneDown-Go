package utils

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// RemoveInvalidChars 去除无效字符
func RemoveInvalidChars(data []byte) []byte {
	for len(data) > 0 && (data[0] != '{' && data[0] != '[') {
		data = data[1:]
	}
	return data
}

// ParseJSON 解析 JSON 数据
func ParseJSON(validData string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(validData), &result)
	return result, err
}

// JsonEncode 将对象编码为JSON字符串
func JsonEncode(data interface{}) any {
	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return string(jsonData)
}

// ConvertGBKToUTF8 将GBK编码转换为UTF-8
func ConvertGBKToUTF8(data []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
	return ioutil.ReadAll(reader)
}