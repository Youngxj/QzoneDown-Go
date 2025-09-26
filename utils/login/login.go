package login

import (
	"QzoneDown-Go/utils"
	"context"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/fatih/color"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	goQrcode "github.com/skip2/go-qrcode"
	"image"
	_ "image/jpeg" // 支持 JPEG 格式
	_ "image/png"  // 支持 PNG 格式
	"os"
	"runtime"
)

var loginURL = "https://i.qq.com/"        // 登录URL
var loginQrcode []byte                    // 登录二维码
var QrcodeSrc string = "login_qrcode.jpg" // 登录二维码图片地址

// GetClientCookie 获取客户端Cookie
func GetClientCookie() error {
	chromePath := LocateChrome()
	fmt.Println("chromePath", chromePath)
	if chromePath == "" {
		return fmt.Errorf("未找到Chrome浏览器，请安装Chrome浏览器后重试，或者手动配置cookie参数")
	}
	color.Cyan("正在初始化浏览器...")
	ctx, cancel := chromedp.NewExecAllocator(
		context.Background(),
		// 以默认配置的数组为基础，覆写headless参数
		// 当然也可以根据自己的需要进行修改，这个flag是浏览器的设置
		append(
			chromedp.DefaultExecAllocatorOptions[:],
			chromedp.ExecPath(chromePath),
			chromedp.Flag("headless", true),
		)...,
	)
	defer cancel()

	c, cancel1 := chromedp.NewContext(ctx)
	defer cancel1()

	err := chromedp.Run(c, myTasks())
	if err != nil {
		return err
	}
	return err
}

// 登录任务
//
//	@return chromedp.Tasks
func myTasks() chromedp.Tasks {
	color.Cyan("正在加载登录二维码，请稍等")
	return chromedp.Tasks{
		//chromedp.EmulateViewport(1920, 2000), // 模拟视口大小，防止元素被隐藏
		chromedp.Navigate(loginURL), // 打开登录页面
		//chromedp.WaitVisible(`#qrlogin_img`, chromedp.BySearch), // 等待二维码出现
		chromedp.Screenshot(`#login_frame`, &loginQrcode, chromedp.NodeVisible), // 获取二维码图片地址
		chromedp.ActionFunc(func(ctx context.Context) (err error) {
			// 1. 保存文件
			if err = os.WriteFile(QrcodeSrc, loginQrcode, 0755); err != nil {
				return
			}
			_, err = GetLoginCode(QrcodeSrc, "")
			if err != nil {
				return err
			}
			return
		}),
		chromedp.WaitVisible(`#tb_logout`, chromedp.BySearch), // 等待退出登录按钮出现
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := os.Remove(QrcodeSrc)
			if err != nil {
				color.Red("删除文件失败:%s", err)
				return err
			}
			// 获取cookie
			cookies, err := network.GetCookies().Do(ctx)
			// 将cookie拼接成header请求中cookie字段的模式
			var c string
			for _, v := range cookies {
				c = c + v.Name + "=" + v.Value + ";"
			}
			//log.Println(c)
			if err != nil {
				return err
			}
			color.Green("获取cookie成功")
			err = setCookie(c)
			if err != nil {
				return err
			}
			return nil
		}),
	}
}

// GetLoginCode 获取登录二维码
func GetLoginCode(qrSrc string, outputType string) (qr *goQrcode.QRCode, err error) {
	// 1. 打开二维码图片
	file, err := os.Open(qrSrc) // 替换为你的二维码图片路径
	if err != nil {
		return nil, fmt.Errorf("无法打开图片:%s", err)
	}
	defer file.Close()

	// 2. 解码图片
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("无法解码图片:%s", err)
	}

	// 3. 创建二维码解码器
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil, fmt.Errorf("无法创建二维码位图:%s", err)
	}

	// 4. 识别二维码
	qrReader := qrcode.NewQRCodeReader()
	result, err := qrReader.Decode(bmp, nil)
	if err != nil {
		return nil, fmt.Errorf("二维码识别失败:%s", err)
	}

	// 5. 用结果来获取go-qrcode对象
	qr, err = goQrcode.New(result.GetText(), goQrcode.Medium)
	if err != nil {
		return nil, err
	}

	// 6. 输出到标准输出流
	fmt.Println(qr.ToSmallString(false))
	color.Cyan("请使用手机QQ扫描二维码登录，自动获取Cookie")
	if outputType == "qrObject" {
		return qr, nil
	}

	return nil, err
}

// 设置cookie
//
//	@param cookie
//	@return error
func setCookie(cookie string) error {
	config, _ := utils.LoadConfig()
	config.Cookie = cookie
	config.Uin = utils.GetUin(cookie)
	config.GTk = fmt.Sprint(utils.GetGTK2("", utils.GetCookieKey(cookie, "p_skey"), cookie))
	err := utils.SaveConfig(config)
	return err
}

// LocateChrome 检测chrome安装路径
//
//	@return string
func LocateChrome() string {
	if path, ok := os.LookupEnv("LORCACHROME"); ok {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	var paths []string
	switch runtime.GOOS {
	case "darwin":
		paths = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
		}
	case "windows":
		paths = []string{
			os.Getenv("LocalAppData") + "/Google/Chrome/Application/chrome.exe",
			os.Getenv("ProgramFiles") + "/Google/Chrome/Application/chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "/Google/Chrome/Application/chrome.exe",
			os.Getenv("LocalAppData") + "/Chromium/Application/chrome.exe",
			os.Getenv("ProgramFiles") + "/Chromium/Application/chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "/Chromium/Application/chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "/Microsoft/Edge/Application/msedge.exe",
			os.Getenv("ProgramFiles") + "/Microsoft/Edge/Application/msedge.exe",
		}
	default:
		paths = []string{
			"/usr/bin/google-chrome-stable",
			"/usr/bin/google-chrome",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		return path
	}
	return ""
}
