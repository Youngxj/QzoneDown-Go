package main

import (
	"DzoneDown-Go/enum"
	"DzoneDown-Go/utils"
	"DzoneDown-Go/utils/progress"
	"DzoneDown-Go/utils/table_format"
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// 定义一个结构体来匹配 JSON 数据结构
type photoListResponseStruct struct {
	Code    int    `json:"code"`
	Subcode int    `json:"subcode"`
	Message string `json:"message"`
	Data    struct {
		VFeeds []struct {
			Pic photoListPicStruct `json:"pic"`
		} `json:"vFeeds"`
		HasMore     int `json:"has_more"`
		RemainCount int `json:"remain_count"` // 剩余数量
	} `json:"data"`
}

// 相册图片列表Struct
type photoImgListResponseStruct struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Album struct {
			Name           string `json:"name"`           // 相册名称
			Desc           string `json:"desc"`           // 相册描述
			Createtime     int    `json:"createtime"`     // 相册创建时间
			Moditytime     int    `json:"moditytime"`     // 相册修改时间
			Lastuploadtime int    `json:"lastuploadtime"` // 相册最后上传时间
		} `json:"album"` // 相册详情
		TotalCount int         `json:"total_count"` // 相册图片总数
		ListCount  int         `json:"list_count"`  // 相册图片列表数量
		Photos     interface{} `json:"photos"`      // 相册图片列表
	} `json:"data"`
}

// PhotoInfo 定义图片信息的结构
type PhotoInfo struct {
	URL         string `json:"url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	FocusX      int    `json:"focus_x"`
	FocusY      int    `json:"focus_y"`
	EnlargeRate int    `json:"enlarge_rate"`
}

// 相册信息Struct
type photoListPicStruct struct {
	Albumid        string          `json:"albumid"`        //相册id
	Desc           string          `json:"desc"`           //相册描述
	Albumname      string          `json:"albumname"`      //相册名称
	Albumnum       int             `json:"albumnum"`       //相册照片数量
	Albumquestion  string          `json:"albumquestion"`  //相册问题
	Albumrights    int             `json:"albumrights"`    //相册访问权限
	Lastupdatetime int             `json:"lastupdatetime"` //相册最后更新时间
	Anonymity      int             `json:"anonymity"`      //主题
	Picdata        json.RawMessage `json:"picdata"`        //其他属性
	Photos         [][]PhotoInfo
}

var picArray []photoListPicStruct // 相册信息列表
var currenPic photoListPicStruct  // 当前相册信息
var photoPn = 20                  // 相册图片列表分页
var picPn = 40                    // 相册列表分页最小10，最大40

var bar progress.Bar              // 下载总数进度条初始化
var photoDownSuccessNum int32 = 0 // 相册图片下载成功数量

// 相册列表接口
var photoListApi = "https://mobile.qzone.qq.com/list?g_tk=%s&format=json&list_type=album&action=0&res_uin=%s&count=%d&res_attach="

// 相册图片列表接口
var photoImgApi = "https://h5.qzone.qq.com/webapp/json/mqzone_photo/getPhotoList2?g_tk=%s&uin=%s&albumid=xxxxxxxxx&ps=0&pn=20&password=&password_cleartext=0&swidth=1080&sheight=1920"

// GlobalConfig 全局配置对象
var GlobalConfig *utils.Configs

func main() {
	headerText()
	configInit()
	initApi()
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
		"\033[36mVersion\033[0m：\033[32m2.0.0\033[0m\n" +
		"\033[36mDescription\033[0m：\n" +
		"	本程序用于下载QQ空间相册中的图片。\n" +
		"	\033[33m使用方法\033[0m：\n" +
		"		\033[34m1. 登录\033[4mhttps://qzone.qq.com\033[0m\033[34m并获取你的cookie以及g_tk和uin\n" +
		"		2. 运行程序并输入你的cookie以及g_tk和uin\n" +
		"		3. 程序会自动下载相册中的图片\033[0m\n" +
		"\033[31mWarning\033[0m：本程序仅用于学习和研究，不得用于商业用途。\n")
}

// initApi 初始化API
func initApi() {
	photoListApi = fmt.Sprintf(photoListApi, GlobalConfig.GTk, GlobalConfig.Uin, picPn)
	photoImgApi = fmt.Sprintf(photoImgApi, GlobalConfig.GTk, GlobalConfig.Uin)
}

// configInit 配置初始化
func configInit() {
	GlobalConfig, _ = utils.LoadConfig()
	if GlobalConfig.Cookie == "" || GlobalConfig.GTk == "" || GlobalConfig.Uin == "" {
		newConfig()
	} else {
		color.Red("已配置Cookie和GTK >>>")
		fmt.Printf("%v%s\n%v%s\n%v%s\n", color.GreenString("Cookie："), GlobalConfig.Cookie, color.GreenString("GTk："), GlobalConfig.GTk, color.GreenString("Uin："), GlobalConfig.Uin)
		isAgent := "y"
		fmt.Print("是否使用已有配置？(y/n) 默认y：")
		_, err := fmt.Scanln(&isAgent)
		if err != nil {
			return
		}
		if isAgent == "n" {
			newConfig()
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
func newConfig() {
	fmt.Print("请输入Cookie:")
	cookie := ""
	scanner := bufio.NewScanner(os.Stdin) // 特殊输入
	if scanner.Scan() {
		cookie = scanner.Text()
	}
	GlobalConfig.Cookie = cookie
	if &GlobalConfig.Cookie == nil {
		color.Red("Cookie不能为空")
		return
	}

	gTk := fmt.Sprint(utils.GetGTK2(photoImgApi, utils.GetCookieKey(GlobalConfig.Cookie, "skey"), GlobalConfig.Cookie)) // 自动计算的gtk
	GlobalConfig.GTk = gTk
	if &GlobalConfig.GTk == nil {
		fmt.Print("请输入GTK:")
		fmt.Scanln(&GlobalConfig.GTk)
		if &GlobalConfig.GTk == nil {
			color.Red("GTK不能为空")
			return
		}
	}

	GlobalConfig.Uin = utils.GetUin(GlobalConfig.Cookie)
	if &GlobalConfig.Uin == nil {
		fmt.Print("请输入Uin:")
		fmt.Scanln(&GlobalConfig.Uin)
		if &GlobalConfig.Uin == nil {
			color.Red("Uin不能为空")
			return
		}
	}
	err := utils.SaveConfig(GlobalConfig)
	if err != nil {
		return
	}
}

// getData
func getData() {
	picList, err := getPicList()
	picArray = picList
	if err != nil {
		color.Red("获取相册列表失败:", err)
		return
	} else if len(picArray) <= 0 {
		color.Red("相册列表为空")
		return
	}
	picFormat() // 打印输出格式化表格
	// 创建一个 Scanner 对象，用于读取标准输入
	scanner := bufio.NewScanner(os.Stdin)
	color.Green("请输入编号继续操作 全部下载输入0 其他任意字符退出：（默认0）")
	for {
		// 提示用户输入
		fmt.Print(">>> ")
		// 读取一行输入
		if scanner.Scan() {
			picScanln := scanner.Text() // 获取输入的文本
			// 输入编号执行任务
			picId, err := strconv.Atoi(picScanln)
			if picScanln != "" && err != nil { // 非数字或不等于空都退出
				color.Red("程序即将退出……👋")
				return
			}
			currenPicName := ""
			if picId > 0 {
				err = getPhotoImages(picId)
				if err != nil {
					fmt.Println(err)
				}
				currenPicName = currenPic.Albumname
			} else if picScanln == "" {
				// 全部下载
				for i := range picArray {
					err = getPhotoImages(i + 1)
					if err != nil {
						fmt.Println(err)
					}
				}
				currenPicName = "全部相册"
			} else {
				color.Red("输入有误，请重新输入")
				continue
			}
			picFormat() // 打印输出格式化表格
			fmt.Printf("%v 请输入编号继续操作 全部下载输入0 其他任意字符退出：\n", color.GreenString(fmt.Sprintf("<%s> 下载完成👌", currenPicName)))
		} else {
			// 如果读取失败，打印错误信息
			color.Red("程序即将退出……👋")
			break
		}
	}
}

// 获取指定相册图片列表
//
//	@param picId	相册ID（序号）
func getPhotoImages(picId int) (errs error) {
	picInfo := picArray[picId-1]
	currenPic = picInfo
	albumid := picInfo.Albumid
	fmt.Printf("开始下载 相册名称：%s 照片数量：%d albumid：%s \n", picInfo.Albumname, picInfo.Albumnum, albumid)

	bar = progress.Bar{} // 在这里重新初始化bar，否则会出现进度条叠加的情况
	bar.NewOptionWithGraph(0, int64(picInfo.Albumnum), "✨")
	photoDownSuccessNum = 0 // 重置下载成功数量

	// 计算分页
	pageCount := int(math.Ceil(float64(picInfo.Albumnum) / float64(photoPn)))
	for i := 0; i < pageCount; i++ {
		urls, err := getPhotoImageUrls(albumid, i)
		if err != nil {
			errs = fmt.Errorf("获取相册图片列表失败:%s", err)
			return
		}
		picInfo.Photos = append(picInfo.Photos, urls)
	}
	bar.Finish()
	return errs
}

// 获取相册Url链接
//
//	@param albumid	相册ID（内部唯一ID）
//	@param page 页码
func getPhotoImageUrls(albumid string, page int) (photoImgList []PhotoInfo, errs error) {
	photoUrl := utils.UrlSetValue(photoImgApi, "albumid", albumid)
	photoUrl = utils.UrlSetValue(photoUrl, "ps", strconv.Itoa(page*photoPn))
	//fmt.Println("photoUrl", photoUrl)
	//return
	body := request(photoUrl)
	var photoImgListResponse photoImgListResponseStruct
	err := json.Unmarshal(body, &photoImgListResponse)
	if err != nil {
		errs = fmt.Errorf("解析 JSON 数据失败.getPhotoImages：%s", err)
		return
	}
	if photoImgListResponse.Code != 0 {
		errs = fmt.Errorf("接口返回错误.photoImgList：%s", photoImgListResponse.Message)
		return
	}

	photosData := photoImgListResponse.Data.Photos.(map[string]interface{})
	var wg sync.WaitGroup // 用于等待所有 goroutine 完成
	for _, photo := range photosData {
		for _, info := range photo.([]interface{}) {
			_info := info.(map[string]interface{})
			// 检查 _info["1"] 是否存在
			if data, ok := _info["1"]; ok {
				// 将 data 序列化为 JSON 字节切片
				jsonData, err := json.Marshal(data)
				if err != nil {
					err = fmt.Errorf("序列化数据失败:%s", err)
					continue
				}
				var pInfo PhotoInfo
				// 将 JSON 字节切片反序列化为 PhotoInfo 结构体
				err = json.Unmarshal(jsonData, &pInfo)
				if err != nil {
					errs = fmt.Errorf("反序列化数据失败:%s", err)
					continue
				}
				photoImgList = append(photoImgList, pInfo)

				photoUrl := pInfo.URL
				wg.Add(1) // 增加等待组计数
				go func(url string) {
					defer wg.Done() // 标记 goroutine 完成
					_, err = utils.Download(url, "images/"+currenPic.Albumname+"/", utils.MD5(url))
					if err != nil {
						errs = fmt.Errorf("%s", err)
					}
					// 使用原子操作安全地增加计数器
					atomic.AddInt32(&photoDownSuccessNum, 1)
					bar.Play(int64(photoDownSuccessNum))
				}(photoUrl)
			}
		}
	}
	wg.Wait() // 等待所有 goroutine 完成
	return photoImgList, errs
}

// 获取相册列表
//
//	@return picArrayData
//	@return err
func getPicList() (picArrayData []photoListPicStruct, err error) {
	// 初始化一个变量用于存储所有分页的相册数据
	var allPicArrayData []photoListPicStruct

	// 定义当前页码
	currentPage := 1
	for {
		// 构建当前页码的请求 URL
		resAttach := fmt.Sprintf("att=start_count=%d", (currentPage-1)*picPn)
		currentPhotoListApi := utils.UrlSetValue(photoListApi, "res_attach", resAttach)

		// 发起请求
		body := request(currentPhotoListApi)
		var photoList photoListResponseStruct
		err = json.Unmarshal(body, &photoList)
		if err != nil {
			err = fmt.Errorf("解析 JSON 数据失败.getPicList：%s", err)
			return
		}
		if photoList.Code != 0 {
			err = fmt.Errorf("接口返回错误：%s", photoList.Message)
			return
		}

		// 提取当前页的相册数据
		var currentPageData []photoListPicStruct
		for _, VFeeds := range photoList.Data.VFeeds {
			// 创建一个映射来存储当前的值
			item := photoListPicStruct{
				Albumname:      VFeeds.Pic.Albumname,
				Albumid:        VFeeds.Pic.Albumid,
				Albumnum:       VFeeds.Pic.Albumnum,
				Desc:           VFeeds.Pic.Desc,
				Lastupdatetime: VFeeds.Pic.Lastupdatetime,
				Albumrights:    VFeeds.Pic.Albumrights,
				Anonymity:      VFeeds.Pic.Anonymity,
			}
			currentPageData = append(currentPageData, item)
		}

		// 合并当前页的数据到总数据中
		allPicArrayData = append(allPicArrayData, currentPageData...)

		// 判断是否还有更多数据
		if photoList.Data.HasMore == 0 {
			break
		}

		// 增加页码
		currentPage++
	}

	return allPicArrayData, nil
}

// 统一请求方法
//
//	@param apiUrl
//	@return body
func request(apiUrl string) (body []byte) {
	httpClient := &http.Client{}
	var req *http.Request
	req, _ = http.NewRequest("GET", apiUrl, nil)
	req.Header.Add("Cookie", GlobalConfig.Cookie)

	var response, err = httpClient.Do(req)
	if err != nil {
		fmt.Println("请求"+apiUrl+"接口失败:", err)
		return
	}
	body, err = io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("读取"+apiUrl+"接口返回数据失败:", err)
		return
	}
	return body
}

// 相册格式化输出
func picFormat() {
	t := table_format.NewTable()
	t.AddTitle(fmt.Sprintf("QQ：%s 相册列表", GlobalConfig.Uin))
	header := table.Row{"相册名称", "相册数量", "最后更新", "访问权限", "相册描述"}
	t.MakeHeader(header)
	var rows []table.Row
	for _, pic := range picArray {
		_time := time.Unix(int64(pic.Lastupdatetime), 0).Format("2006-01-02")
		_albumrights, _ := enum.ConvertRightsEnum(pic.Albumrights)
		rows = append(rows, table.Row{pic.Albumname, pic.Albumnum, _time, _albumrights, pic.Desc})
	}
	t.AppendRows(rows)
	t.Print()
}
