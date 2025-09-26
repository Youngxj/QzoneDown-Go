package main

import (
	"QzoneDown-Go/api"
	"QzoneDown-Go/enum"
	"QzoneDown-Go/utils"
	"QzoneDown-Go/utils/login"
	_ "QzoneDown-Go/utils/login"
	"QzoneDown-Go/utils/table_format"
	"bufio"
	"fmt"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"os"
	"strconv"
	"time"
)

// GlobalConfig 全局配置对象
var GlobalConfig *utils.Configs

func main() {
	headerText()
	configInit()
	api.InitApi()
	getData()
}

// headerText
func headerText() {
	fmt.Println(`
	
   ____                       _____                      
  / __ \                     |  __ \                     
 | |  | | _______  _ __   ___| |  | | _____      ___ __  
 | |  | ||_  / _ \| '_ \ / _ \ |  | |/ _ \ \ /\ / / '_ \ 
 | |__| | / / (_) | | | |  __/ |__| | (_) \ V  V /| | | |
  \___\_\/___\___/|_| |_|\___|_____/ \___/ \_/\_/ |_| |_|
                                                         
	`)
	fmt.Println("\n" +
		"\033[36mName\033[0m：\033[32mQQ空间相册下载器(Golang)\033[0m\n" +
		"\033[36mVersion\033[0m：\033[32m2.4.0\033[0m\n" +
		"\033[36mDescription\033[0m：\n" +
		"	本程序用于下载自己或指定QQ空间相册中的图片。\n" +
		"	\033[33m使用方法\033[0m：\n" +
		"		\033[35m1. 登录\033[4mhttps://qzone.qq.com\033[0m\033[35m并获取你的cookie\n" +
		"		2. 运行程序并输入你的cookie，g_tk和uin将自动识别\n" +
		"		3. 按照要求输入，程序会自动下载相册中的图片\n" +
		"		4. 图片下载完成后会按照相册名分类保存在images目录中\033[0m\n" +
		"\033[31mWarning\033[0m：本程序仅用于学习和研究，不得用于商业用途。")
}

// configInit 配置初始化
func configInit() {
	var err error
	GlobalConfig, err = utils.LoadConfig()
	if err != nil {
		fmt.Println("err：", err)
		err := newConfig("")
		if err != nil {
			color.Red("%s", err)
		}
	} else if GlobalConfig.Cookie == "" || GlobalConfig.GTk == "" || GlobalConfig.Uin == "" {
		err := login.GetClientCookie()
		if err != nil {
			color.Red("%s", err)
			err = newConfig("")
			if err != nil {
				color.Red("%s", err)
			}
		} else {
			err = newConfig("gtk")
			if err != nil {
				color.Red("%s", err)
			}
		}
	} else {
		color.Red("已配置Cookie和GTK >>>")
		fmt.Printf("%v%s\n%v%s\n%v%s\n", color.GreenString("Cookie："), GlobalConfig.Cookie, color.GreenString("GTk："), GlobalConfig.GTk, color.GreenString("Uin："), GlobalConfig.Uin)
		isAgent := "y"
		fmt.Print("是否使用已有配置？(y/n) 默认y：")
		_, err = fmt.Scanln(&isAgent)
		if err != nil {
			return
		}
		if isAgent == "n" {
			err := login.GetClientCookie()
			if err != nil {
				color.Red("%s", err)
				err = newConfig("")
				if err != nil {
					color.Red("%s", err)
				}
			} else {
				err = newConfig("gtk")
				if err != nil {
					color.Red("%s", err)
				}
			}
		} else if isAgent == "y" {
			//使用已有配置
			return
		} else {
			fmt.Println("输入有误，请重新输入")
			configInit()
		}
	}
}

// newConfig 新配置
func newConfig(configType string) error {
	GlobalConfig, _ = utils.LoadConfig()
	if configType == "" || configType == "cookie" {
		fmt.Print("请输入Cookie:")
		cookie := ""
		scanner := bufio.NewScanner(os.Stdin) // 特殊输入
		if scanner.Scan() {
			cookie = scanner.Text()
		}
		GlobalConfig.Cookie = cookie
		if &GlobalConfig.Cookie == nil || GlobalConfig.Cookie == "" {
			color.Red("Cookie不能为空")
			os.Exit(0)
		}
	}

	if configType == "" || configType == "gtk" {
		gTk := fmt.Sprint(utils.GetGTK2(api.PhotoImgApi, utils.GetCookieKey(GlobalConfig.Cookie, "p_skey"), GlobalConfig.Cookie)) // 自动计算的gtk
		GlobalConfig.GTk = gTk
		if &GlobalConfig.GTk == nil {
			fmt.Print("请输入GTK:")
			_, err := fmt.Scanln(&GlobalConfig.GTk)
			if err != nil {
				color.Red("%s", err)
				os.Exit(0)
			}
			if &GlobalConfig.GTk == nil || GlobalConfig.GTk == "" {
				color.Red("GTK不能为空")
				os.Exit(0)
			}
		}
	}

	uin := ""
	if configType == "" || configType == "uin" {
		fmt.Print("请输入要访问的相册QQ号(默认当前登录QQ号):")
		scanner := bufio.NewScanner(os.Stdin) // 特殊输入
		if scanner.Scan() {
			uin = scanner.Text()
		}
		if uin == "" {
			GlobalConfig.Uin = utils.GetUin(GlobalConfig.Cookie)
		} else {
			GlobalConfig.Uin = uin
		}

		if &GlobalConfig.Uin == nil || GlobalConfig.Uin == "" {
			color.Red("Uin不能为空")
			os.Exit(0)
		}
	}
	err := utils.SaveConfig(GlobalConfig)
	return err
}

// getData
func getData() {
	actionTips := "请输入编号继续操作 0=全部下载 q=切换QQ号 (默认0)："
	exitTips := "程序即将退出……👋"
	for {
		picList, err := api.GetPicList()
		api.PicArray = picList
		if err != nil {
			color.Red("%s", err)
			return
		} else if len(api.PicArray) <= 0 {
			color.Red("相册列表为空")
			return
		}
		picFormat() // 打印输出格式化表格
		// 创建一个 Scanner 对象，用于读取标准输入
		scanner := bufio.NewScanner(os.Stdin)
		color.Green(actionTips)
		for {
			// 提示用户输入
			fmt.Print(">>> ")
			// 读取一行输入
			if scanner.Scan() {
				picScanln := scanner.Text() // 获取输入的文本
				// 输入编号执行任务
				picId, err := strconv.Atoi(picScanln)
				if picScanln != "" && err != nil && picScanln != "q" { // 非特定条件都退出程序
					color.Red(exitTips)
					return
				}
				currenPicName := ""
				if picId > 0 {
					err = api.GetPhotoImages(picId)
					if err != nil {
						color.Red("%s", err)
						continue
					}
					currenPicName = api.CurrenPic.Albumname
				} else if picScanln == "" {
					// 全部下载
					for i := range api.PicArray {
						err = api.GetPhotoImages(i + 1)
						if err != nil {
							color.Red("%s", err)
							continue
						}
					}
					currenPicName = "全部相册"
				} else if picScanln == "q" {
					// 调用 setUin 方法
					err := newConfig("uin")
					if err != nil {
						color.Red("%s", err)
					}
					// 跳出内层循环，重新执行流程
					break
				} else {
					color.Red("输入有误，请重新输入")
					continue
				}
				picFormat() // 打印输出格式化表格
				if err == nil {
					color.Green(fmt.Sprintf("<%s> 下载完成👌", currenPicName))
				}
				fmt.Println(actionTips)
			} else {
				// 如果读取失败，打印错误信息
				color.Red(exitTips)
				return
			}
		}
	}
}

// 相册格式化输出
func picFormat() {
	t := table_format.NewTable()
	t.AddTitle(fmt.Sprintf("QQ：%s 相册列表", GlobalConfig.Uin))
	header := table.Row{"相册名称", "相册数量", "最后更新", "访问权限", "相册描述"}
	t.MakeHeader(header)
	var rows []table.Row
	for _, pic := range api.PicArray {
		_time := time.Unix(int64(pic.Lastupdatetime), 0).Format("2006-01-02")
		_albumrights, _ := enum.ConvertRightsEnum(pic.Albumrights)
		rows = append(rows, table.Row{pic.Albumname, pic.Albumnum, _time, _albumrights, pic.Desc})
	}
	t.AppendRows(rows)
	t.Print()
}
