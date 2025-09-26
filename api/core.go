package api

import (
	"QzoneDown-Go/utils"
	_ "QzoneDown-Go/utils/login"
	"QzoneDown-Go/utils/progress"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"math/rand/v2"

	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// PhotoListResponseStruct 定义一个结构体来匹配 JSON 数据结构
type PhotoListResponseStruct struct {
	Code    int    `json:"code"`
	Subcode int    `json:"subcode"`
	Message string `json:"message"`
	Data    struct {
		VFeeds []struct {
			Pic PhotoListPicStruct `json:"pic"`
		} `json:"vFeeds"`
		HasMore     int `json:"has_more"`
		RemainCount int `json:"remain_count"` // 剩余数量
	} `json:"data"`
}

// PhotoImgListResponseStruct 相册图片列表Struct
type PhotoImgListResponseStruct struct {
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

// PhotoListPicStruct 相册信息Struct
type PhotoListPicStruct struct {
	Id             int             `json:"id"`             //自定义相册ID索引
	Albumid        string          `json:"albumid"`        //相册id
	Desc           string          `json:"desc"`           //相册描述
	Albumname      string          `json:"albumname"`      //相册名称
	Albumnum       int             `json:"albumnum"`       //相册照片数量
	Albumquestion  string          `json:"albumquestion"`  //相册问题
	Albumrights    int             `json:"albumrights"`    //相册访问权限
	Lastupdatetime int             `json:"lastupdatetime"` //相册最后更新时间
	Anonymity      int             `json:"anonymity"`      //主题
	Picdata        json.RawMessage `json:"picdata"`        //其他属性
	Photos         [][]string
}

var PicArray []PhotoListPicStruct // 相册信息列表
var CurrenPic PhotoListPicStruct  // 当前相册信息
var photoPn = 20                  // 相册图片列表分页
var picPn = 40                    // 相册列表分页最小10，最大40

var bar progress.Bar              // 下载总数进度条初始化
var photoDownSuccessNum int32 = 0 // 相册图片下载成功数量
// 相册图片下载成功总数量
var photoDownSuccessNumCount int32 = 0

// PhotoListApi 相册列表接口
var PhotoListApi = "https://mobile.qzone.qq.com/list?g_tk=%s&format=json&list_type=album&action=0&res_uin=%s&count=%d&res_attach="

// PhotoImgApi 相册图片列表接口
var PhotoImgApi = "https://h5.qzone.qq.com/webapp/json/mqzone_photo/getPhotoList2?g_tk=%s&uin=%s&albumid=xxxxxxxxx&ps=0&pn=20&password=&password_cleartext=0&swidth=1080&sheight=1920"

// GlobalConfig 全局配置对象
var GlobalConfig *utils.Configs

func init() {
	InitApi()
}

// InitApi 初始化API
func InitApi() {
	GlobalConfig, _ = utils.LoadConfig()
	PhotoListApi = utils.UrlSetValue(PhotoListApi, "g_tk", GlobalConfig.GTk)
	PhotoListApi = utils.UrlSetValue(PhotoListApi, "res_uin", GlobalConfig.Uin)
	PhotoListApi = utils.UrlSetValue(PhotoListApi, "count", strconv.Itoa(picPn))

	PhotoImgApi = utils.UrlSetValue(PhotoImgApi, "g_tk", GlobalConfig.GTk)
	PhotoImgApi = utils.UrlSetValue(PhotoImgApi, "uin", GlobalConfig.Uin)
}

// GetPhotoImages 获取指定相册图片列表
//
//	@param picId	相册ID（序号）
func GetPhotoImages(picId int) (errs error) {
	if picId <= 0 || picId > len(PicArray) {
		errs = fmt.Errorf("相册ID：%d不存在，请重新输入", picId)
		return
	}
	picInfo := PicArray[picId-1]
	CurrenPic = picInfo
	albumid := picInfo.Albumid
	fmt.Printf("开始下载 相册名称：%s 照片数量：%s albumid：%s \n", color.CyanString(picInfo.Albumname), color.CyanString(strconv.Itoa(picInfo.Albumnum)), albumid)

	bar = progress.Bar{} // 在这里重新初始化bar，否则会出现进度条叠加的情况
	bar.NewOptionWithGraph(0, int64(picInfo.Albumnum), "✨")
	photoDownSuccessNum = 0 // 重置下载成功数量

	// 计算分页
	pageCount := int(math.Ceil(float64(picInfo.Albumnum) / float64(photoPn)))
	for i := 0; i < pageCount; i++ {
		urls, err := GetPhotoImageUrls(albumid, i)
		if err != nil {
			errs = fmt.Errorf("获取相册图片列表失败:%s", err)
			return
		}
		picInfo.Photos = append(picInfo.Photos, urls)

		for _, photoUrl := range urls {
			err = downloadImage(photoUrl, picInfo.Albumname, func() {

			})
		}

	}
	bar.Finish()
	return errs
}

// 下载图片
func downloadImage(photoUrl string, Albumname string, fn func()) (err error) {
	var timestamp = time.Now().Unix()
	uUploadTimeString := time.Unix(timestamp, 0).Format("20060102150405")
	var wg sync.WaitGroup // 用于等待所有 goroutine 完成
	wg.Add(1)             // 增加等待组计数
	go func(url string) {
		defer wg.Done() // 标记 goroutine 完成
		fileName := fmt.Sprintf("%s_%04d", uUploadTimeString, rand.IntN(10000))
		_, err := utils.Download(url, "images/"+utils.FileNameFiltering(Albumname)+"/", fileName)
		if err != nil {
			fmt.Println("Download err", err)
		}
		// 使用原子操作安全地增加计数器
		atomic.AddInt32(&photoDownSuccessNum, 1)
		atomic.AddInt32(&photoDownSuccessNumCount, 1)
		fn()
	}(photoUrl)
	wg.Wait() // 等待所有 goroutine 完成
	return err
}

// GetPhotoImageUrls 获取相册Url链接
//
//	@param albumid	相册ID（内部唯一ID）
//	@param page 页码
func GetPhotoImageUrls(albumid string, page int) (photoImgList []string, errs error) {
	photoUrl := utils.UrlSetValue(PhotoImgApi, "albumid", albumid)
	photoUrl = utils.UrlSetValue(photoUrl, "ps", strconv.Itoa(page*photoPn))
	photoUrl = utils.UrlSetValue(photoUrl, "uin", GlobalConfig.Uin)
	//fmt.Println("photoUrl", photoUrl)
	//return
	body, err := Request(photoUrl)
	if err != nil {
		errs = fmt.Errorf("%s", err)
		return
	}
	var photoImgListResponse PhotoImgListResponseStruct
	err = json.Unmarshal(body, &photoImgListResponse)
	if err != nil {
		errs = fmt.Errorf("解析 JSON 数据失败.getPhotoImages：%s", err)
		return
	}
	if photoImgListResponse.Code != 0 {
		errs = fmt.Errorf("接口返回错误.photoImgList：%s", photoImgListResponse.Message)
		return
	}

	photosData := photoImgListResponse.Data.Photos.(map[string]interface{})
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
				photoImgList = append(photoImgList, pInfo.URL)
			}
		}
	}
	return photoImgList, errs
}

// GetPicList 获取相册列表
//
//	@return picArrayData
//	@return err
func GetPicList() (picArrayData []PhotoListPicStruct, err error) {
	// 初始化一个变量用于存储所有分页的相册数据
	var allPicArrayData []PhotoListPicStruct

	// 定义当前页码
	currentPage := 1
	for {
		// 构建当前页码的请求 URL
		resAttach := fmt.Sprintf("att=start_count=%d", (currentPage-1)*picPn)
		PhotoListApi = utils.UrlSetValue(PhotoListApi, "res_attach", resAttach)
		currentPhotoListApi := utils.UrlSetValue(PhotoListApi, "res_uin", GlobalConfig.Uin)
		// 发起请求
		body, err := Request(currentPhotoListApi)
		if err != nil {
			err = fmt.Errorf("获取相册图片列表失败:%s", err)
			return nil, err
		}
		var photoList PhotoListResponseStruct
		err = json.Unmarshal(body, &photoList)
		if err != nil {
			err = fmt.Errorf("解析 JSON 数据失败.getPicList：%s", err)
			return nil, err
		}
		if photoList.Code != 0 {
			err = fmt.Errorf("接口返回错误：%s", photoList.Message)
			return nil, err
		}

		// 提取当前页的相册数据
		var currentPageData []PhotoListPicStruct
		for _, VFeeds := range photoList.Data.VFeeds {
			// 创建一个映射来存储当前的值
			item := PhotoListPicStruct{
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

// Request 统一请求方法
//
//	@param apiUrl
//	@return body
func Request(apiUrl string) (body []byte, err error) {
	httpClient := &http.Client{}
	var req *http.Request
	req, _ = http.NewRequest("GET", apiUrl, nil)
	req.Header.Add("Cookie", GlobalConfig.Cookie)

	response, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		err = fmt.Errorf(response.Status)
		return nil, err
	}

	body, err = io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("读取接口返回数据失败:%s", err)
		return nil, err
	}
	return body, err
}

// GetPhotoList 获取相册列表
func GetPhotoList(currentPage int) (photoList PhotoListResponseStruct, err error) {
	// 构建当前页码的请求 URL
	resAttach := fmt.Sprintf("att=start_count=%d", (currentPage-1)*picPn)
	PhotoListApi = utils.UrlSetValue(PhotoListApi, "res_attach", resAttach)
	currentPhotoListApi := utils.UrlSetValue(PhotoListApi, "res_uin", GlobalConfig.Uin)
	fmt.Println("currentPhotoListApi", currentPhotoListApi)
	// 发起请求
	body, err := Request(currentPhotoListApi)
	if err != nil {
		err = fmt.Errorf("获取相册图片列表失败:%s", err)
		return photoList, err
	}
	err = json.Unmarshal(body, &photoList)
	if err != nil {
		err = fmt.Errorf("解析 JSON 数据失败.getPicList：%s", err)
		return photoList, err
	}
	if photoList.Code != 0 {
		err = fmt.Errorf("接口返回错误：%s", photoList.Message)
		return photoList, err
	}
	return photoList, nil
}

// GetPhotoImages2 获取相册图片URL列表（gui）
func GetPhotoImages2(albumid string, Albumnum int) (photos []string, err error) {
	// 计算分页
	pageCount := int(math.Ceil(float64(Albumnum) / float64(photoPn)))
	for i := 0; i < pageCount; i++ {
		urls, err := GetPhotoImageUrls(albumid, i)
		if err != nil {
			err = fmt.Errorf("获取相册图片列表失败:%s", err)
			return nil, err
		}
		photos = append(photos, urls...)
	}
	return photos, err
}

// 获取相册Url链接
//
//	@param albumid	相册ID（内部唯一ID）
//	@param page 页码
func getPhotoImageUrls2(albumid string, page int) (photoImgUrlList []string, errs error) {
	photoUrl := utils.UrlSetValue(PhotoImgApi, "albumid", albumid)
	photoUrl = utils.UrlSetValue(photoUrl, "ps", strconv.Itoa(page*photoPn))
	photoUrl = utils.UrlSetValue(photoUrl, "uin", GlobalConfig.Uin)
	//fmt.Println("photoUrl", photoUrl)
	//return
	body, err := Request(photoUrl)
	if err != nil {
		errs = fmt.Errorf("%s", err)
		return
	}
	var photoImgListResponse PhotoImgListResponseStruct
	err = json.Unmarshal(body, &photoImgListResponse)
	if err != nil {
		errs = fmt.Errorf("解析 JSON 数据失败.getPhotoImages：%s", err)
		return
	}
	if photoImgListResponse.Code != 0 {
		errs = fmt.Errorf("接口返回错误.photoImgList：%s", photoImgListResponse.Message)
		return
	}

	photosData := photoImgListResponse.Data.Photos.(map[string]interface{})
	for _, photo := range photosData {
		for _, info := range photo.([]interface{}) {
			_info := info.(map[string]interface{})
			if data, ok := _info["1"]; ok {
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
				photoUrl := pInfo.URL
				photoImgUrlList = append(photoImgUrlList, photoUrl)
			}
		}
	}
	return photoImgUrlList, errs
}

// TestReq 测试连接
func TestReq() (count int, err error) {
	InitApi()
	resAttach := fmt.Sprintf("att=start_count=%d", 0)
	PhotoListApi = utils.UrlSetValue(PhotoListApi, "res_attach", resAttach)
	currentPhotoListApi := utils.UrlSetValue(PhotoListApi, "res_uin", GlobalConfig.Uin)
	fmt.Println("currentPhotoListApi", currentPhotoListApi)
	// 发起请求
	body, err := Request(currentPhotoListApi)
	if err != nil {
		err = fmt.Errorf("获取相册图片列表失败:%s", err)
		return 0, err
	}
	var photoList PhotoListResponseStruct
	err = json.Unmarshal(body, &photoList)
	if err != nil {
		err = fmt.Errorf("解析 JSON 数据失败.TestReq：%s", err)
		return 0, err
	}
	if photoList.Code != 0 {
		err = fmt.Errorf("接口返回错误：%s", photoList.Message)
		return 0, err
	}
	photoCount := len(photoList.Data.VFeeds)
	return photoCount, nil
}
