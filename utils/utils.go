package utils

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ExistDir 检查目录是否存在，不存在则创建
//
//	@param path
func ExistDir(path string) {
	// 判断路径是否存在
	_, err := os.ReadDir(path)
	if err != nil {
		// 不存在就创建
		err = os.MkdirAll(path, fs.ModePerm)
		if err != nil {
			fmt.Println(err)
		}
	}
}

// MD5 MD5字符串获取
//
//	@param str
//	@return string
func MD5(str string) string {
	data := []byte(str) //切片
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has) //将[]byte转成16进制
	return md5str
}

// UrlSetValue 替换get参数
//
//	@param rawURL
//	@param key
//	@param value
//	@return string
func UrlSetValue(rawURL string, key string, value string) string {
	parsedURL, err := url.Parse(rawURL) // 解析 URL
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return rawURL
	}
	query := parsedURL.Query()          // 获取查询参数
	query.Set(key, value)               // 替换参数 key 的值为 value
	parsedURL.RawQuery = query.Encode() // 重新构建 URL
	return parsedURL.String()           // 输出新的 URL
}

// GetGTK 获取GTK（通过cookie中skey参数）
//
//	@param skey
//	@return int32
func GetGTK(skey string) int32 {
	hash := int32(5381)
	for i := 0; i < len(skey); {
		r, size := utf8.DecodeRuneInString(skey[i:])
		hash += (hash << 5) + int32(r)
		i += size
	}
	return hash & 0x7fffffff
}

// GetCookieKey 模拟从 cookie 获取值的函数
func GetCookieKey(cookieString string, key string) string {
	re := regexp.MustCompile(key + `=([^;]+)`)
	match := re.FindStringSubmatch(cookieString)

	var val string
	if len(match) > 1 {
		val = match[1]
	} else {
		val = ""
	}
	return val
}

// GetSkey 获取cookie中skey参数
//
//	@param cookieString
//	@return string
func GetSkey(cookieString string) string {
	re := regexp.MustCompile(`skey=([^;]+)`)
	match := re.FindStringSubmatch(cookieString)

	var skey string
	if len(match) > 1 {
		skey = match[1]
	} else {
		skey = ""
	}
	return skey
}

// GetGTK2 获取GTK2（根据 skey 或 cookie 值生成哈希值 官方算法）
//
//	@param urlString
//	@param skey
//	@param cookie
//	@return int32
func GetGTK2(urlString, skey string, cookie string) int32 {
	// 默认值
	str := skey

	// 如果 URL 提供，根据域名调整使用的密钥
	if urlString != "" {
		parsedURL, err := url.Parse(urlString)
		if err == nil {
			hostname := parsedURL.Host
			if strings.Contains(hostname, "qun.qq.com") || (strings.Contains(hostname, "qzone.qq.com") && !strings.Contains(hostname, "qun.qzone.qq.com")) {
				// 这里假设有一个函数 getCookie 获取 cookie 值
				pSkey := GetCookieKey(cookie, "p_skey")
				if pSkey != "" {
					str = pSkey
				}
			}
		}
	}

	// 如果 str 为空，尝试从其他 cookie 获取
	if str == "" {
		str = GetCookieKey(cookie, "skey")
		if str == "" {
			str = GetCookieKey(cookie, "rv2")
		}
	}

	// 计算哈希值
	hash := int32(5381)
	for i := 0; i < len(str); {
		r, size := utf8.DecodeRuneInString(str[i:])
		hash += (hash << 5) + int32(r)
		i += size
	}

	return hash & 0x7fffffff
}

// GetUin 获取cookie中uin参数
//
//	@param cookieString
//	@return string
func GetUin(cookieString string) string {
	return strings.Replace(GetCookieKey(cookieString, "uin"), "o", "", 1)
}

// Loading 加载动画
//
//	@param str
func Loading(str string) {
	// 清空控制台
	fmt.Print("\033[H\033[2J")

	// 定义加载动画
	animation := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	// 模拟耗时操作
	for i := 0; i < len(animation); i++ {
		// 打印加载动画
		fmt.Printf("\r%s %s", str, animation[i%len(animation)])
		os.Stdout.Sync() // 确保立即打印

		// 等待一段时间
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print("\033[H\033[2J")
}
