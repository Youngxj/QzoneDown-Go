package web

import (
	"QzoneDown-Go/api"
	"QzoneDown-Go/enum"
	"QzoneDown-Go/utils"
	"QzoneDown-Go/utils/login"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	goQrcode "github.com/skip2/go-qrcode"
)

// ---- Download Manager ----

type ProgressItem struct {
	AlbumName string `json:"albumName"`
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	Status    string `json:"status"` // "downloading", "completed", "error"
	Error     string `json:"error,omitempty"`
}

type DownloadManager struct {
	mu           sync.Mutex
	progressList []ProgressItem
	progressChan chan ProgressItem
	isRunning    bool
	stopChan     chan struct{}
}

var dm = &DownloadManager{
	progressChan: make(chan ProgressItem, 200),
	stopChan:     make(chan struct{}),
}

func (d *DownloadManager) sendProgress(p ProgressItem) {
	// completed/error/all_completed 等关键状态绝不丢弃
	if p.Status == "completed" || p.Status == "error" || p.Status == "all_completed" {
		d.progressChan <- p
		return
	}
	// 普通进度更新在 channel 满时可以丢弃
	select {
	case d.progressChan <- p:
	default:
	}
}

// ---- Config Handlers ----

func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		cfg, _ := utils.LoadConfig()
		data, _ := json.Marshal(map[string]interface{}{
			"cookie": cfg.Cookie,
			"gtk":    cfg.GTk,
			"uin":    cfg.Uin,
		})
		jsonResponse(w, data)

	case "POST":
		var body struct {
			Cookie string `json:"cookie"`
			Uin    string `json:"uin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, 400, "invalid request body")
			return
		}

		cfg, _ := utils.LoadConfig()
		if body.Cookie != "" {
			cfg.Cookie = body.Cookie
			cfg.GTk = fmt.Sprint(utils.GetGTK2(api.PhotoImgApi, utils.GetCookieKey(body.Cookie, "p_skey"), body.Cookie))
		}
		if body.Uin != "" {
			cfg.Uin = body.Uin
		}
		if err := utils.SaveConfig(cfg); err != nil {
			jsonError(w, 500, "save config failed")
			return
		}
		api.InitApi()

		data, _ := json.Marshal(map[string]interface{}{
			"cookie": cfg.Cookie,
			"gtk":    cfg.GTk,
			"uin":    cfg.Uin,
		})
		jsonResponse(w, data)

	default:
		jsonError(w, 405, "method not allowed")
	}
}

// ---- Album Handlers ----

func handleAlbums(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, 405, "method not allowed")
		return
	}

	list, err := api.GetPicList()
	if err != nil {
		jsonError(w, 500, err.Error())
		return
	}

	type albumItem struct {
		ID             int    `json:"id"`
		Albumname      string `json:"albumname"`
		Albumnum       int    `json:"albumnum"`
		Desc           string `json:"desc"`
		Lastupdatetime string `json:"lastupdatetime"`
		Albumrights    string `json:"albumrights"`
	}

	cfg, _ := utils.LoadConfig()
	items := make([]albumItem, 0, len(list))
	for i, p := range list {
		rights, _ := enum.ConvertRightsEnum(p.Albumrights)
		items = append(items, albumItem{
			ID:             i + 1,
			Albumname:      p.Albumname,
			Albumnum:       p.Albumnum,
			Desc:           p.Desc,
			Lastupdatetime: time.Unix(int64(p.Lastupdatetime), 0).Format("2006-01-02"),
			Albumrights:    rights,
		})
	}

	data, _ := json.Marshal(map[string]interface{}{
		"uin":    cfg.Uin,
		"albums": items,
	})
	jsonResponse(w, data)
}

// ---- Download Handlers ----

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method == "DELETE" {
		dm.mu.Lock()
		if dm.isRunning {
			close(dm.stopChan)
			dm.isRunning = false
		}
		dm.mu.Unlock()
		jsonResponse(w, []byte(`{"status":"stopped"}`))
		return
	}

	if r.Method != "POST" {
		jsonError(w, 405, "method not allowed")
		return
	}

	var body struct {
		AlbumIds interface{} `json:"albumIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, 400, "invalid request body")
		return
	}

	dm.mu.Lock()
	if dm.isRunning {
		dm.mu.Unlock()
		jsonError(w, 429, "download already in progress")
		return
	}
	dm.isRunning = true
	dm.progressList = nil
	dm.stopChan = make(chan struct{})

	// 排空上一次下载残留的消息
	for {
		select {
		case <-dm.progressChan:
		default:
			goto drained
		}
	}
drained:
	dm.mu.Unlock()

	// 同步获取相册列表（在 goroutine 之前，保证 SSE 连上时进度列表已就绪）
	picList, err := api.GetPicList()
	if err != nil {
		dm.mu.Lock()
		dm.isRunning = false
		dm.mu.Unlock()
		jsonError(w, 500, err.Error())
		return
	}
	api.SetPicArray(picList)

	var ids []int
	switch v := body.AlbumIds.(type) {
	case string:
		if v == "all" {
			for i := range picList {
				ids = append(ids, i+1)
			}
		}
	case []interface{}:
		for _, id := range v {
			if n, ok := id.(float64); ok {
				ids = append(ids, int(n))
			}
		}
	}

	albumIdxMap := make(map[int]int)
	dm.mu.Lock()
	for _, id := range ids {
		if id <= 0 || id > len(picList) {
			continue
		}
		pic := picList[id-1]
		dm.progressList = append(dm.progressList, ProgressItem{
			AlbumName: pic.Albumname,
			Total:     pic.Albumnum,
			Status:    "downloading",
		})
		albumIdxMap[id] = len(dm.progressList) - 1
	}
	dm.mu.Unlock()

	go func() {
		// 最多同时下载 3 个相册
		const maxConcurrent = 3
		sem := make(chan struct{}, maxConcurrent)
		var wg sync.WaitGroup

		for _, id := range ids {
			if id <= 0 || id > len(picList) {
				continue
			}

			// 总控停止检查
			select {
			case <-dm.stopChan:
				goto waitDone
			default:
			}

			idx, ok := albumIdxMap[id]
			if !ok {
				continue
			}

			wg.Add(1)
			go func(albumId, albumIdx int) {
				defer wg.Done()

				// 获取信号量
				select {
				case sem <- struct{}{}:
				case <-dm.stopChan:
					return
				}
				defer func() { <-sem }()

				select {
				case <-dm.stopChan:
					return
				default:
				}

				pic := picList[albumId-1]
				err := downloadAlbum(albumId, albumIdx, picList)
				if err != nil {
					dm.mu.Lock()
					if albumIdx < len(dm.progressList) {
						dm.progressList[albumIdx].Status = "error"
						dm.progressList[albumIdx].Error = err.Error()
						dm.progressList[albumIdx].Current = pic.Albumnum
					}
					dm.mu.Unlock()
					dm.sendProgress(ProgressItem{AlbumName: pic.Albumname, Current: pic.Albumnum, Total: pic.Albumnum, Status: "error", Error: err.Error()})
				} else {
					dm.mu.Lock()
					if albumIdx < len(dm.progressList) {
						dm.progressList[albumIdx].Status = "completed"
						dm.progressList[albumIdx].Current = pic.Albumnum
					}
					dm.mu.Unlock()
					dm.sendProgress(ProgressItem{AlbumName: pic.Albumname, Current: pic.Albumnum, Total: pic.Albumnum, Status: "completed"})
				}
			}(id, idx)
		}

	waitDone:
		wg.Wait()

		dm.mu.Lock()
		dm.isRunning = false
		dm.mu.Unlock()
		dm.sendProgress(ProgressItem{Status: "all_completed"})
	}()

	jsonResponse(w, []byte(`{"status":"started"}`))
}

func handleDownloadProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, 500, "streaming not supported")
		return
	}

	dm.mu.Lock()
	pendingList := make([]ProgressItem, len(dm.progressList))
	copy(pendingList, dm.progressList)
	dm.mu.Unlock()
	if len(pendingList) > 0 {
		data, _ := json.Marshal(map[string]interface{}{"type": "list", "data": pendingList})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	} else {
		data, _ := json.Marshal(map[string]interface{}{"type": "list", "data": []ProgressItem{}})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case p, ok := <-dm.progressChan:
			if !ok {
				return
			}
			data, _ := json.Marshal(map[string]interface{}{"type": "update", "data": p})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			if p.Status == "all_completed" || (p.Status == "error" && p.AlbumName == "") {
				return
			}
		}
	}
}

func downloadAlbum(picId int, idx int, picList []api.PhotoListPicStruct) error {
	picInfo := picList[picId-1]
	total := picInfo.Albumnum

	pageCount := int(math.Ceil(float64(total) / float64(20)))
	if pageCount == 0 {
		pageCount = 1
	}

	currentCount := 0

	for i := 0; i < pageCount; i++ {
		// 每页开始前检查停止信号
		select {
		case <-dm.stopChan:
			dm.mu.Lock()
			if idx < len(dm.progressList) {
				dm.progressList[idx].Current = currentCount
			}
			dm.mu.Unlock()
			dm.sendProgress(ProgressItem{AlbumName: picInfo.Albumname, Current: currentCount, Total: total, Status: "error", Error: "stopped"})
			return fmt.Errorf("download stopped")
		default:
		}

		urls, err := api.GetPhotoImageUrls(picInfo.Albumid, i)
		if err != nil {
			return fmt.Errorf("get photo urls failed: %s", err)
		}

		for _, photoUrl := range urls {
			// 每张图片下载前检查停止信号
			select {
			case <-dm.stopChan:
				dm.mu.Lock()
				if idx < len(dm.progressList) {
					dm.progressList[idx].Current = currentCount
				}
				dm.mu.Unlock()
				dm.sendProgress(ProgressItem{AlbumName: picInfo.Albumname, Current: currentCount, Total: total, Status: "error", Error: "stopped"})
				return fmt.Errorf("download stopped")
			default:
			}

			err := downloadWebImage(photoUrl, picInfo.Albumname)
			if err != nil {
				fmt.Printf("Download err: %s\n", err)
				continue
			}
			currentCount++

			// 每5张照片或最后一张才发送进度，避免 channel 溢出
			if currentCount%5 != 0 && currentCount != total {
				dm.mu.Lock()
				if idx < len(dm.progressList) {
					dm.progressList[idx].Current = currentCount
				}
				dm.mu.Unlock()
				continue
			}

			dm.mu.Lock()
			if idx < len(dm.progressList) {
				dm.progressList[idx].Current = currentCount
			}
			dm.mu.Unlock()

			dm.sendProgress(ProgressItem{
				AlbumName: picInfo.Albumname,
				Current:   currentCount,
				Total:     total,
				Status:    "downloading",
			})
		}
	}

	return nil
}

func downloadWebImage(photoUrl, albumName string) error {
	cfg, _ := utils.LoadConfig()
	savePath := "images/" + cfg.Uin + "/" + utils.FileNameFiltering(albumName) + "/"
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%s_%04d", time.Unix(timestamp, 0).Format("20060102150405"), rand.IntN(10000))

	res, err := http.Get(photoUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	utils.ExistDir(savePath)
	file, err := os.Create(savePath + fileName + ".jpg")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	return err
}

// ---- Login Handlers (Web QR Login) ----

// decodeAndRegenerateQR 从截图解析二维码内容，重新生成更大尺寸的二维码PNG
func decodeAndRegenerateQR(pngBytes []byte) ([]byte, error) {
	// 1. 解码截图图片
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, fmt.Errorf("decode screenshot: %w", err)
	}

	// 2. 从图片中识别二维码
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("create bitmap: %w", err)
	}

	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return nil, fmt.Errorf("decode QR content: %w", err)
	}

	// 3. 用解析出的内容重新生成 300x300 的大尺寸二维码
	newQR, err := goQrcode.Encode(result.GetText(), goQrcode.Medium, 300)
	if err != nil {
		return nil, fmt.Errorf("generate large QR: %w", err)
	}

	return newQR, nil
}

type webLoginState struct {
	mu             sync.Mutex
	running        bool
	err            string
	qrBytes        []byte
	qrReady        bool
	qrReadyCh      chan struct{}
	chromedpCancel context.CancelFunc
}

var webLogin = &webLoginState{
	qrReadyCh: make(chan struct{}, 1),
}

func handleLoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		jsonError(w, 405, "method not allowed")
		return
	}

	webLogin.mu.Lock()
	if webLogin.running {
		webLogin.mu.Unlock()
		jsonError(w, 429, "login already in progress")
		return
	}
	webLogin.running = true
	webLogin.err = ""
	webLogin.qrBytes = nil
	webLogin.qrReady = false
	webLogin.qrReadyCh = make(chan struct{}, 1)
	webLogin.mu.Unlock()

	// Reset config cookie so check doesn't falsely report "completed" from old config
	cfg, _ := utils.LoadConfig()
	cfg.Cookie = ""
	utils.SaveConfig(cfg)

	go func() {
		chromePath := login.LocateChrome()
		if chromePath == "" {
			webLogin.mu.Lock()
			webLogin.err = "Chrome not found"
			webLogin.running = false
			webLogin.mu.Unlock()
			return
		}

		allocCtx, allocCancel := chromedp.NewExecAllocator(
			context.Background(),
			append(
				chromedp.DefaultExecAllocatorOptions[:],
				chromedp.ExecPath(chromePath),
				chromedp.Flag("headless", true),
			)...,
		)
		defer allocCancel()

		ctx, cancel := chromedp.NewContext(allocCtx)
		defer cancel()

		// Phase 1: Capture QR code screenshot, parse content, regenerate larger QR
		var qrScreenshot []byte
		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.Navigate("https://i.qq.com/"),
			chromedp.ActionFunc(func(ctx context.Context) error {
				return nil
			}),
			chromedp.Screenshot(`#login_frame`, &qrScreenshot, chromedp.NodeVisible, chromedp.BySearch),
		})
		if err != nil {
			webLogin.mu.Lock()
			webLogin.err = "QR capture failed: " + err.Error()
			webLogin.running = false
			webLogin.mu.Unlock()
			return
		}

		// Parse QR content from screenshot and generate larger QR code
		largeQR, err := decodeAndRegenerateQR(qrScreenshot)
		if err != nil {
			webLogin.mu.Lock()
			webLogin.err = "QR decode failed: " + err.Error()
			webLogin.running = false
			webLogin.mu.Unlock()
			return
		}

		// Save regenerated large QR to shared state
		webLogin.mu.Lock()
		webLogin.qrBytes = largeQR
		webLogin.qrReady = true
		webLogin.mu.Unlock()

		// Signal QR is ready
		select {
		case webLogin.qrReadyCh <- struct{}{}:
		default:
		}

		// Phase 2: Wait for user to scan and login
		var cookies []*network.Cookie
		err = chromedp.Run(ctx, chromedp.Tasks{
			chromedp.WaitVisible(`#tb_logout`, chromedp.BySearch),
			chromedp.ActionFunc(func(ctx context.Context) error {
				var err error
				cookies, err = network.GetCookies().Do(ctx)
				return err
			}),
		})
		if err != nil {
			webLogin.mu.Lock()
			webLogin.err = "Login wait failed: " + err.Error()
			webLogin.running = false
			webLogin.mu.Unlock()
			return
		}

		// Save cookies
		var cStr string
		for _, v := range cookies {
			cStr = cStr + v.Name + "=" + v.Value + ";"
		}

		cfg, _ := utils.LoadConfig()
		cfg.Cookie = cStr
		cfg.Uin = utils.GetUin(cStr)
		cfg.GTk = fmt.Sprint(utils.GetGTK2(api.PhotoImgApi, utils.GetCookieKey(cStr, "p_skey"), cStr))
		utils.SaveConfig(cfg)
		api.InitApi()

		webLogin.mu.Lock()
		webLogin.running = false
		webLogin.mu.Unlock()
	}()

	jsonResponse(w, []byte(`{"status":"started"}`))
}

func handleLoginQrcode(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, 405, "method not allowed")
		return
	}

	webLogin.mu.Lock()
	ready := webLogin.qrReady
	bytes := webLogin.qrBytes
	webLogin.mu.Unlock()

	if !ready {
		// Check if login already failed
		webLogin.mu.Lock()
		errMsg := webLogin.err
		running := webLogin.running
		webLogin.mu.Unlock()
		if !running && errMsg != "" {
			data, _ := json.Marshal(map[string]interface{}{
				"ready": false,
				"error": errMsg,
			})
			jsonResponse(w, data)
			return
		}
		data, _ := json.Marshal(map[string]interface{}{"ready": false})
		jsonResponse(w, data)
		return
	}

	if len(bytes) == 0 {
		jsonError(w, 404, "no QR code available")
		return
	}

	b64 := base64.StdEncoding.EncodeToString(bytes)
	data, _ := json.Marshal(map[string]interface{}{
		"ready":   true,
		"dataUrl": "data:image/png;base64," + b64,
	})
	jsonResponse(w, data)
}

func handleLoginCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, 405, "method not allowed")
		return
	}

	webLogin.mu.Lock()
	running := webLogin.running
	err := webLogin.err
	qrReady := webLogin.qrReady
	webLogin.mu.Unlock()

	cfg, _ := utils.LoadConfig()
	hasCookie := cfg.Cookie != ""

	data, _ := json.Marshal(map[string]interface{}{
		"running":   running,
		"completed": !running && hasCookie && err == "",
		"error":     err,
		"hasCookie": hasCookie,
		"qrReady":   qrReady,
	})
	jsonResponse(w, data)
}
