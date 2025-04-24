package main

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	// 修改为相对导入路径
	"qzone-down/utils"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
)

// QzoneImgDown 结构体
type QzoneImgDown struct {
	gTk     string
	resUin  string
	cookie  string
	listApi string
	imgApi  string
	threads int // 线程数字段
}

// NewQzoneImgDown 构造函数
func NewQzoneImgDown(gTk, resUin, cookie string, threads int) *QzoneImgDown {
	listApi := fmt.Sprintf("https://mobile.qzone.qq.com/list?g_tk=%s&format=json&list_type=album&action=0&res_uin=%s&count=99", gTk, resUin)
	imgApi := fmt.Sprintf("https://h5.qzone.qq.com/webapp/json/mqzone_photo/getPhotoList2?g_tk=%s&uin=%s&albumid=xxxxxxxxxxxxx&ps=0&pn=999&password=&password_cleartext=0&swidth=1080&sheight=1920", gTk, resUin)
	return &QzoneImgDown{
		gTk:     gTk,
		resUin:  resUin,
		cookie:  cookie,
		listApi: listApi,
		imgApi:  imgApi,
		threads: threads,
	}
}

// getResponse 发送 HTTP 请求并获取响应数据
func (q *QzoneImgDown) getResponse(ctx context.Context, url string) (string, error) {
	c := g.Client()
	c.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	c.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:43.0) Gecko/20100101 Firefox/43.0")
	c.SetHeader("Accept", "*/*")
	c.SetHeader("Connection", "keep-alive")
	c.SetHeader("Cookie", q.cookie)
	if debugMode {
		fmt.Println("url", url)
	}
	r, e := c.Get(gctx.New(), url)
	if e != nil {
		panic(e)
	}
	body := r.ReadAllString()
	return body, e
}

// getImg 获取图片列表
func (q *QzoneImgDown) getImg(ctx context.Context, url string) error {
	body, err := q.getResponse(ctx, url)
	if err != nil {
		return fmt.Errorf("请求出错: %w", err)
	}

	data, err := utils.ParseJSON(body)
	if err != nil {
		return fmt.Errorf("解析 JSON 出错: %w", err)
	}

	if data["data"] != nil {
		dataMap, ok := data["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("数据格式错误: data 不是 map")
		}
		if dataMap["album"] != nil {
			albumMap, ok := dataMap["album"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("数据格式错误: album 不是 map")
			}
			fileName, ok := albumMap["name"].(string)
			if !ok {
				return fmt.Errorf("数据格式错误: album.name 不是字符串")
			}

			// 检查并创建 download_links 目录
			if err := utils.EnsureDir(utils.DownloadLinksDir); err != nil {
				return err
			}

			txtPath := filepath.Join(utils.DownloadLinksDir, fileName+".txt")
			// 第一次清空文件
			file, err := os.OpenFile(txtPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("文件写入失败，请检查目录读写权限是否正常: %w", err)
			}
			file.Close()

			// 后续采用追加模式写入
			file, err = os.OpenFile(txtPath, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("文件写入失败，请检查目录读写权限是否正常: %w", err)
			}
			defer file.Close()

			if dataMap["photos"] != nil {
				photos, ok := dataMap["photos"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("数据格式错误: photos 不是切片")
				}
				var imgUrls []string
				for _, photo := range photos {
					photoMap, ok := photo.([]interface{})
					if !ok {
						return fmt.Errorf("数据格式错误: photo 不是 map")
					}
					for _, subPhoto := range photoMap {
						subPhotoMap, ok := subPhoto.(map[string]interface{})
						if !ok {
							return fmt.Errorf("数据格式错误: subPhoto 不是 map")
						}
						if imgInfo, ok := subPhotoMap["1"].(map[string]interface{}); ok {
							if imgUrl, ok := imgInfo["url"].(string); ok {
								if debugMode {
									fmt.Println(imgUrl)
								}
								imgUrls = append(imgUrls, imgUrl)
							}
						}
					}
				}
				for _, url1 := range imgUrls {
					_, err := file.WriteString(url1 + "\n")
					if err != nil {
						return fmt.Errorf("写入文件出错: %w", err)
					}
				}

				// 设置最大并发数
				maxConcurrentDownloads := q.threads // 使用结构体中的线程数

				// 读取txt文件中的链接
				links, err := utils.ReadLinksFromFile(txtPath)
				if err != nil {
					return fmt.Errorf("读取链接文件出错: %w", err)
				}

				totalLinks := len(links)
				if totalLinks == 0 {
					fmt.Println("没有可下载的图片。")
					return nil
				}

				// 只有有图片时才启动进度显示和下载协程
				var wg sync.WaitGroup
				semaphore := make(chan struct{}, maxConcurrentDownloads)
				var progressMutex sync.Mutex
				progress := 0
				done := make(chan struct{})

				// 启动进度显示协程
				startTime := time.Now() // 记录开始时间
				go func(albumName string) {
					for {
						progressMutex.Lock()
						cur := progress
						progressMutex.Unlock()
						fmt.Printf("\r正在下载相册 '%s': %d/%d ...", albumName, cur, totalLinks)
						if cur >= totalLinks {
							break
						}
						time.Sleep(200 * time.Millisecond)
					}
					elapsedTime := time.Since(startTime) // 计算耗时
					fmt.Printf("\r相册 '%s' 下载完成: %d/%d, 耗时: %s\n", albumName, totalLinks, totalLinks, elapsedTime)
					close(done)
				}(fileName)

				for idx := range links {
					wg.Add(1)
					semaphore <- struct{}{}
					go func(idx int, albumName string) {
						defer func() {
							if r := recover(); r != nil {
								fmt.Printf("下载协程异常: %v\n", r)
							}
							wg.Done()
							<-semaphore
						}()
						// 传递相册名
						if err := utils.DownloadFileWithAlbum(links[idx], albumName); err != nil {
							fmt.Printf("\nFailed to download %s from album '%s': %v\n", links[idx], albumName, err)
						}
						progressMutex.Lock()
						progress++
						progressMutex.Unlock()
					}(idx, fileName)
				}
				wg.Wait()
				<-done
			}
		}
	} else {
		return fmt.Errorf("error: 响应数据中 data 字段为空")
	}
	return nil
}

// ret 计算翻页
func (q *QzoneImgDown) ret(ctx context.Context, url string) error {
	body, err := q.getResponse(ctx, url)
	if err != nil {
		return fmt.Errorf("请求出错: %w", err)
	}

	data, err := utils.ParseJSON(body)
	if err != nil {
		return fmt.Errorf("解析 JSON 出错: %w", err)
	}

	if data["data"] != nil {
		dataMap, ok := data["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("数据格式错误: data 不是 map")
		}
		if dataMap["album"] != nil {
			totalCount, ok := dataMap["total_count"].(float64)
			if !ok {
				return fmt.Errorf("数据格式错误: total_count 不是浮点数")
			}
			pageSize := 999
			pageCount := int(math.Ceil(totalCount/float64(pageSize))) - 1

			// 先处理第一页（已拿到数据）
			err := q.handleImgData(ctx, url, data)
			if err != nil {
				return err
			}
			// 处理后续页
			for i := 1; i <= pageCount; i++ {
				newUrl := q.urlSetValue(url, "ps", i*pageSize)
				err := q.getImg(ctx, newUrl)
				if err != nil {
					return err
				}
			}
			fmt.Println("All downloads completed.") // 只在所有分页完成后输出
		} else {
			return fmt.Errorf("error: 响应数据中 album 字段为空")
		}
	} else {
		return fmt.Errorf("error: 响应数据中 data 字段为空")
	}
	return nil
}

// 专门处理图片数据（与 getImg 逻辑一致，但直接用已获取的数据）
func (q *QzoneImgDown) handleImgData(ctx context.Context, url string, data map[string]interface{}) error {
	dataMap, ok := data["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("数据格式错误: data 不是 map")
	}
	if dataMap["album"] != nil {
		albumMap, ok := dataMap["album"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("数据格式错误: album 不是 map")
		}
		fileName, ok := albumMap["name"].(string)
		if !ok {
			return fmt.Errorf("数据格式错误: album.name 不是字符串")
		}

		// 检查并创建 download_links 目录
		if err := utils.EnsureDir(utils.DownloadLinksDir); err != nil {
			return err
		}

		txtPath := filepath.Join(utils.DownloadLinksDir, fileName+".txt")
		// 第一次清空文件
		file, err := os.OpenFile(txtPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("文件写入失败，请检查目录读写权限是否正常: %w", err)
		}
		file.Close()

		// 后续采用追加模式写入
		file, err = os.OpenFile(txtPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("文件写入失败，请检查目录读写权限是否正常: %w", err)
		}
		defer file.Close()

		if dataMap["photos"] != nil {
			photos, ok := dataMap["photos"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("数据格式错误: photos 不是切片")
			}
			var imgUrls []string
			for _, photo := range photos {
				photoMap, ok := photo.([]interface{})
				if !ok {
					return fmt.Errorf("数据格式错误: photo 不是 map")
				}
				for _, subPhoto := range photoMap {
					subPhotoMap, ok := subPhoto.(map[string]interface{})
					if !ok {
						return fmt.Errorf("数据格式错误: subPhoto 不是 map")
					}
					if imgInfo, ok := subPhotoMap["1"].(map[string]interface{}); ok {
						if imgUrl, ok := imgInfo["url"].(string); ok {
							if debugMode {
								fmt.Println(imgUrl)
							}
							imgUrls = append(imgUrls, imgUrl)
						}
					}
				}
			}
			for _, url1 := range imgUrls {
				_, err := file.WriteString(url1 + "\n")
				if err != nil {
					return fmt.Errorf("写入文件出错: %w", err)
				}
			}

			// 设置最大并发数
			maxConcurrentDownloads := q.threads // 使用结构体中的线程数

			// 读取txt文件中的链接
			links, err := utils.ReadLinksFromFile(txtPath)
			if err != nil {
				return fmt.Errorf("读取链接文件出错: %w", err)
			}

			totalLinks := len(links)
			if totalLinks == 0 {
				fmt.Println("没有可下载的图片。")
				return nil
			}

			// 只有有图片时才启动进度显示和下载协程
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, maxConcurrentDownloads)
			var progressMutex sync.Mutex
			progress := 0
			done := make(chan struct{})

			// 启动进度显示协程
			startTime := time.Now() // 记录开始时间
			go func(albumName string) {
				for {
					progressMutex.Lock()
					cur := progress
					progressMutex.Unlock()
					fmt.Printf("\r正在下载相册 '%s': %d/%d ...", albumName, cur, totalLinks)
					if cur >= totalLinks {
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				elapsedTime := time.Since(startTime) // 计算耗时
				fmt.Printf("\r相册 '%s' 下载完成: %d/%d, 耗时: %s\n", albumName, totalLinks, totalLinks, elapsedTime)
				close(done)
			}(fileName)

			for idx := range links {
				wg.Add(1)
				semaphore <- struct{}{}
				go func(idx int, albumName string) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("下载协程异常: %v\n", r)
						}
						wg.Done()
						<-semaphore
					}()
					// 传递相册名
					if err := utils.DownloadFileWithAlbum(links[idx], albumName); err != nil {
						fmt.Printf("\nFailed to download %s from album '%s': %v\n", links[idx], albumName, err)
					}
					progressMutex.Lock()
					progress++
					progressMutex.Unlock()
				}(idx, fileName)
			}
			wg.Wait()
			<-done
		}
	}
	return nil
}

type AlbumInfo struct {
	Name       string
	AlbumID    string
	PhotoCount int
}

// getList 获取相册列表，只返回相册信息
func (q *QzoneImgDown) getList(ctx context.Context) ([]AlbumInfo, error) {
	body, err := q.getResponse(ctx, q.listApi)
	if err != nil {
		return nil, fmt.Errorf("请求出错: %w", err)
	}

	data, err := utils.ParseJSON(body)
	if err != nil {
		return nil, fmt.Errorf("解析 JSON 出错1: %w", err)
	}

	var albums []AlbumInfo

	if data["data"] != nil {
		dataMap, ok := data["data"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("数据格式错误: data 不是 map")
		}
		if dataMap["vFeeds"] != nil {
			vFeeds, ok := dataMap["vFeeds"].([]interface{})
			if !ok {
				return nil, fmt.Errorf("数据格式错误: vFeeds 不是切片")
			}
			for _, feed := range vFeeds {
				feedMap, ok := feed.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("数据格式错误: feed 不是 map")
				}
				if feedMap["pic"] != nil {
					picMap, ok := feedMap["pic"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("数据格式错误: pic 不是 map")
					}
					if debugMode {
						fmt.Printf("DEBUG: 相册原始数据: %+v\n", picMap)
					}
					albumID, ok := picMap["albumid"].(string)
					if !ok {
						return nil, fmt.Errorf("数据格式错误: albumid 不是字符串")
					}
					albumName, _ := picMap["albumname"].(string)
					photocnt := 0
					if v, ok := picMap["albumnum"].(float64); ok {
						photocnt = int(v)
					}
					albums = append(albums, AlbumInfo{
						Name:       albumName,
						AlbumID:    albumID,
						PhotoCount: photocnt,
					})
				}
			}
		} else {
			return nil, fmt.Errorf("error: 响应数据中 vFeeds 字段为空")
		}
	} else {
		if data["message"] != nil {
			message, _ := data["message"].(string)
			return nil, fmt.Errorf("error: " + message)
		}
		return nil, fmt.Errorf("error: 响应数据中 data 字段为空")
	}
	return albums, nil
}

// urlSetValue 替换 URL 参数
func (q *QzoneImgDown) urlSetValue(urlStr, key string, value interface{}) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		fmt.Println("解析 URL 出错:", err)
		return urlStr
	}
	query := u.Query()
	query.Set(key, fmt.Sprintf("%v", value))
	u.RawQuery = query.Encode()
	return u.String()
}

var debugMode = false // 默认为非调试模式，如需调试可设为 true

// main 函数
func main() {
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
		"\033[36mVersion\033[0m：\033[32m1.0.0\033[0m\n" +
		"\033[36mDescription\033[0m：\n" +
		"	本程序用于下载QQ空间相册中的图片。\n" +
		"	\033[33m使用方法\033[0m：\n" +
		"		\033[34m1. 登录\033[4mhttps://qzone.qq.com\033[0m\033[34m并获取你的cookie以及g_tk和uin\n" +
		"		2. 运行程序并输入你的cookie以及g_tk和uin\n" +
		"		3. 程序会自动下载相册中的图片\033[0m\n" +
		"\033[31mWarning\033[0m：本程序仅用于学习和研究，不得用于商业用途。\n")
	var gTk, resUin, cookie string
	// 加载配置
	config, _ := utils.LoadConfig()
	if config.GTk != "" && config.ResUin != "" && config.Cookie != "" {
		fmt.Println("\n检测到已保存的配置：")
		fmt.Printf("GTk: %s\nQQ号: %s\n", config.GTk, config.ResUin)
		fmt.Print("是否使用已保存的配置？(y/n): ")
		var useConfig string
		fmt.Scanln(&useConfig)
		if useConfig == "y" || useConfig == "Y" {
			gTk = config.GTk
			resUin = config.ResUin
			cookie = config.Cookie
		}
	}

	if gTk == "" {
		fmt.Print("请输入g_tk值: ")
		fmt.Scanln(&gTk)
	}

	if resUin == "" {
		fmt.Print("请输入QQ号(uin): ")
		fmt.Scanln(&resUin)
	}

	if cookie == "" {
		fmt.Println("请输入cookie值(完整cookie字符串): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			cookie = scanner.Text()
		}
	}

	if gTk == "" || resUin == "" || cookie == "" {
		fmt.Println("错误：g_tk、QQ号和cookie都不能为空")
		return
	}

	var threads int
	if config.Threads > 0 {
		threads = config.Threads
	} else {
		threads = 5 // 默认线程数
	}

	fmt.Printf("当前线程数为: %d\n", threads)
	fmt.Print("请输入下载线程数 (1-20): ")
	_, err := fmt.Scanln(&threads)
	if err != nil || threads < 1 || threads > 20 {
		fmt.Println("输入无效，使用默认线程数 5。")
		threads = 5
	}

	// 保存配置
	utils.SaveConfig(&utils.Config{
		GTk:     gTk,
		ResUin:  resUin,
		Cookie:  cookie,
		Threads: threads,
	})

	qzone := NewQzoneImgDown(gTk, resUin, cookie, threads) // 传入线程数
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for {
		albums, err := qzone.getList(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("所有相册列表：")
		fmt.Println("0. 全部下载")

		// 计算相册名称最大宽度
		maxNameLen := 0
		for _, album := range albums {
			if l := len([]rune(album.Name)); l > maxNameLen {
				maxNameLen = l
			}
		}
		// 每行显示的相册数
		const albumsPerRow = 3
		// 计算每列的宽度，确保序号对齐
		colWidth := maxNameLen + 20 // 为序号、括号和总数预留足够空间

		for i, album := range albums {
			// 序号固定宽度，名称和总数之间没有空格
			fmt.Printf("%2d. %s(总数:%d)", i+1, album.Name, album.PhotoCount)

			// 如果不是行尾，添加适当的空格使下一列的序号对齐
			if (i+1)%albumsPerRow != 0 {
				// 计算当前输出的长度
				curLen := 4 + len([]rune(album.Name)) + 8 + len(fmt.Sprint(album.PhotoCount))
				// 补充空格使下一列序号对齐
				fmt.Print(strings.Repeat(" ", colWidth-curLen))
			} else {
				fmt.Println() // 换行
			}
		}
		if len(albums)%albumsPerRow != 0 {
			fmt.Println()
		}

		fmt.Print("请输入要拉取的相册编号：")
		var inputIndex int
		_, err = fmt.Scanln(&inputIndex)
		if err != nil || inputIndex < 0 || inputIndex > len(albums) {
			fmt.Println("输入编号无效，请输入正确的编号。")
			continue
		}

		if inputIndex == 0 {
			fmt.Println("正在收集所有相册的图片链接...")
			// 下载全部相册，合并进度
			type ImgTask struct {
				Url       string
				AlbumName string
			}
			var allTasks []ImgTask
			pageSize := 999

			// 添加相册收集进度
			totalAlbums := len(albums)
			for i, album := range albums {
				fmt.Printf("\n正在处理相册(%d/%d): %s\n", i+1, totalAlbums, album.Name)
				// 先获取总数，计算分页
				imgApi := qzone.urlSetValue(qzone.imgApi, "albumid", album.AlbumID)
				body, err := qzone.getResponse(ctx, imgApi)
				if err != nil {
					fmt.Printf("获取相册 %s 列表失败: %v\n", album.Name, err)
					continue
				}
				data, err := utils.ParseJSON(body)
				if err != nil {
					fmt.Printf("解析相册 %s JSON 失败: %v\n", album.Name, err)
					continue
				}
				if data["data"] == nil {
					continue
				}
				dataMap, ok := data["data"].(map[string]interface{})
				if !ok {
					continue
				}
				totalCount, ok := dataMap["total_count"].(float64)
				if !ok {
					continue
				}
				pageCount := int(math.Ceil(totalCount/float64(pageSize))) - 1
				// 只收集所有分页的图片链接，不在这里下载
				for i := 0; i <= pageCount; i++ {
					fmt.Printf("\r  - 正在处理第%d/%d页", i+1, pageCount+1)
					pageApi := qzone.urlSetValue(imgApi, "ps", i*pageSize)
					pageBody, err := qzone.getResponse(ctx, pageApi)
					if err != nil {
						fmt.Printf("获取相册 %s 第%d页失败: %v\n", album.Name, i+1, err)
						continue
					}
					pageData, err := utils.ParseJSON(pageBody)
					if err != nil {
						fmt.Printf("解析相册 %s 第%d页JSON失败: %v\n", album.Name, i+1, err)
						continue
					}
					if pageData["data"] == nil {
						continue
					}
					pageDataMap, ok := pageData["data"].(map[string]interface{})
					if !ok || pageDataMap["photos"] == nil {
						continue
					}
					photos, ok := pageDataMap["photos"].(map[string]interface{})
					if !ok {
						continue
					}
					for _, photo := range photos {
						photoMap, ok := photo.([]interface{})
						if !ok {
							continue
						}
						for _, subPhoto := range photoMap {
							subPhotoMap, ok := subPhoto.(map[string]interface{})
							if !ok {
								continue
							}
							if imgInfo, ok := subPhotoMap["1"].(map[string]interface{}); ok {
								if imgUrl, ok := imgInfo["url"].(string); ok {
									allTasks = append(allTasks, ImgTask{
										Url:       imgUrl,
										AlbumName: album.Name,
									})
								}
							}
						}
					}
				}
				fmt.Println() // 添加换行，分隔不同相册的处理信息
			}
			fmt.Println("\n链接收集完成，开始下载...")

			totalLinks := len(allTasks)
			if totalLinks == 0 {
				fmt.Println("没有可下载的图片。")
				continue
			}

			// 只在这里统一启动一次进度显示和下载
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, threads)
			var progressMutex sync.Mutex
			progress := 0
			done := make(chan struct{})

			go func() {
				for {
					progressMutex.Lock()
					cur := progress
					progressMutex.Unlock()
					fmt.Printf("\r正在下载 %d/%d ...", cur, totalLinks)
					if cur >= totalLinks {
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				fmt.Printf("\r下载完成 %d/%d          \n", totalLinks, totalLinks)
				close(done)
			}()

			for idx, task := range allTasks {
				wg.Add(1)
				semaphore <- struct{}{}
				go func(idx int, t ImgTask) {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("下载协程异常: %v\n", r)
						}
						wg.Done()
						<-semaphore
					}()
					if err := utils.DownloadFileWithAlbum(t.Url, t.AlbumName); err != nil {
						fmt.Printf("\nFailed to download %s: %v\n", t.Url, err)
					}
					progressMutex.Lock()
					progress++
					progressMutex.Unlock()
				}(idx, task)
			}
			wg.Wait()
			<-done
		} else {
			selectedAlbum := &albums[inputIndex-1]
			// 拼接图片API
			imgApi := qzone.urlSetValue(qzone.imgApi, "albumid", selectedAlbum.AlbumID)
			err = qzone.ret(ctx, imgApi)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}