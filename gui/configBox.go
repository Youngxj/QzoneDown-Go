package main

import (
	"QzoneDown-Go/api"
	"QzoneDown-Go/utils"
	"QzoneDown-Go/utils/login"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"os"
	"time"
)

// 定义全局变量存储表单控件引用
var (
	uinEntry    *widget.Entry
	gtkEntry    *widget.Entry
	cookieEntry *widget.Entry
)

// 检查相册是否可用
func checkPhoto() bool {
	_, err := api.TestReq()
	if err != nil {
		return false
	}
	return true
}

func configBox(w fyne.Window) fyne.CanvasObject {
	// 创建一个状态标签用于显示结果信息
	str := binding.NewString() // 状态标签绑定的字符串
	//qrcodeLabel := binding.NewString() // 二维码标签绑定的字符串
	// 创建一个图片组件，初始显示空白
	qrcodeLabel := canvas.NewImageFromImage(nil)
	qrcodeLabel.SetMinSize(fyne.NewSize(400, 300)) // 设置图片显示区域大小
	qrcodeLabel.FillMode = canvas.ImageFillContain // 保持比例填充
	qrcodeLabel.Hide()

	var getQrcodeBtn *widget.Button
	var qrcodeCheck = false
	getQrcodeBtn = widget.NewButton("获取二维码", func() {
		getQrcodeBtn.Disable()
		getQrcodeBtn.SetText("正在获取二维码，请稍候...")
		str.Set("正在获取二维码，请稍候...")
		// 在goroutine中异步执行
		go func() {
			//生成唯一文件名
			login.QrcodeSrc = fmt.Sprintf("login_qrcode_%d.jpg", time.Now().UnixNano())
			//循环异步检查二维码文件是否存在
			go func() {
				for {
					fmt.Println("文件检查中")
					if _, err := os.Stat(login.QrcodeSrc); err == nil {
						qrcodeCheck = true
						getQrcodeBtn.SetText("二维码图片加载成功，请扫码登录")
						str.Set("二维码图片加载成功，请扫码登录")

						img, err := login.GetLoginCode(login.QrcodeSrc, "qrObject")

						if err != nil {
							fyne.CurrentApp().SendNotification(&fyne.Notification{
								Title:   "错误",
								Content: err.Error(),
							})
							continue
						}
						qrcodeLabel.Image = img.Image(300)
						qrcodeLabel.Refresh() // 刷新图片组件
						qrcodeLabel.Show()
						break
					}
					time.Sleep(time.Second)
				}
			}()

			// 检查是否完成扫码
			go func() {
				for {
					//检查配置是否完成
					GlobalConfig, _ := utils.LoadConfig()
					if qrcodeCheck && !utils.FileExists(login.QrcodeSrc) && GlobalConfig.GTk != "" && GlobalConfig.Uin != "" && GlobalConfig.Cookie != "" {
						getQrcodeBtn.Enable()
						getQrcodeBtn.SetText("获取二维码") // 恢复按钮名
						str.Set("已完成")
						fyne.CurrentApp().SendNotification(&fyne.Notification{
							Title:   "成功",
							Content: "已登录",
						})
						qrcodeLabel.Hide()
						break
					}
					time.Sleep(time.Second)
				}
			}()

			//获取cookie任务
			go func() {
				err := login.GetClientCookie()
				if err != nil {
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "错误",
						Content: err.Error(),
					})
					return
				}
			}()
		}()
	})
	getQrcodeBtn.Hide()

	// 顶部描述内容
	descriptionText := widget.NewRichTextFromMarkdown(`
登录 [https://qzone.qq.com/](https://qzone.qq.com/) 可以手动获取Cookie等参数

如果你希望快速获取也可以使用扫码登录(需要安装[Chrome浏览器](https://www.google.cn/chrome/) )
`)

	keyFormBox := keyForm(w) // 手动输入的表单

	getKeyType := widget.NewRadioGroup([]string{"手动输入", "扫码登录"}, func(s string) {
		fmt.Println("s", s)
		if s == "手动输入" {
			keyFormBox.Refresh()
			keyFormBox.Show()
			qrcodeLabel.Hide()
			getQrcodeBtn.Hide()
			updateFormFromConfig() // 更新配置值
		} else if s == "扫码登录" {
			keyFormBox.Hide()
			//重置按钮、文本
			getQrcodeBtn.SetText("获取二维码")
			str.Set("")
			getQrcodeBtn.Show()
		}
	})
	getKeyType.Selected = "手动输入"
	getKeyType.Horizontal = true

	// 先创建按钮变量，初始文本为"验证"
	verifyBtn := widget.NewButton("验证", nil)
	// 验证按钮事件
	verifyBtn.OnTapped = func() {
		verifyBtn.Disable()
		verifyBtn.SetText("验证中")
		// 在回调中可以通过verifyBtn引用自身
		photoCount, err := api.TestReq()

		var t, msg string
		if err != nil {
			t = "验证失败"
			msg = fmt.Sprintf("%s", err.Error())
		} else {
			t = "验证成功"
			msg = fmt.Sprintf("已获取 %d 个相册", photoCount)
		}
		// 显示成功信息到界面
		str.Set(msg)
		// 成功时修改按钮文本
		verifyBtn.SetText(t)

		fyne.CurrentApp().SendNotification(&fyne.Notification{
			Title:   t,
			Content: msg,
		})
		// 刷新按钮以确保文本更新
		verifyBtn.Refresh()

		go func() {
			// 三秒后恢复按钮文本
			time.Sleep(2 * time.Second)
			fyne.Do(func() {
				verifyBtn.Enable()
				verifyBtn.SetText("验证")
			})
		}()

	}
	return container.NewVBox(
		descriptionText,
		getKeyType,
		keyFormBox, getQrcodeBtn, verifyBtn, widget.NewLabelWithData(str), qrcodeLabel)
}

// 新增更新方法：从配置重新加载值
func updateFormFromConfig() {
	GlobalConfig, err := utils.LoadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		return
	}

	// 更新控件值
	if GlobalConfig.Uin != "" {
		uinEntry.Text = GlobalConfig.Uin
	}
	if GlobalConfig.GTk != "" {
		gtkEntry.Text = GlobalConfig.GTk
	}
	if GlobalConfig.Cookie != "" {
		cookieEntry.Text = GlobalConfig.Cookie
	}

	// 刷新UI显示
	uinEntry.Refresh()
	gtkEntry.Refresh()
	cookieEntry.Refresh()
}

// 创建表单并初始化控件
func keyForm(_ fyne.Window) fyne.CanvasObject {
	// 初始化控件并保存引用
	uinEntry = widget.NewEntry()
	uinEntry.SetPlaceHolder("QQ号")

	gtkEntry = widget.NewEntry()
	gtkEntry.SetPlaceHolder("294****274")

	cookieEntry = widget.NewMultiLineEntry()
	cookieEntry.SetPlaceHolder("media_p_skey=xxxxxxx;media_p_uin=xxxxxxxxxxxxx;xxxxxxxxxxxxx")

	// 初始加载配置
	updateFormFromConfig()

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Gtk", Widget: gtkEntry, HintText: "GTK"},
			{Text: "Uin", Widget: uinEntry, HintText: "uin 也就是你的QQ号"},
			{Text: "Cookie", Widget: cookieEntry, HintText: "全部Cookie"},
		},
		SubmitText: "提交",
		CancelText: "取消",
		OnCancel: func() {
			fmt.Println("Cancelled")
		},
		OnSubmit: func() {
			fmt.Println("Form submitted")
			GlobalConfig, _ := utils.LoadConfig()
			// 保存配置逻辑保持不变
			GlobalConfig.Cookie = cookieEntry.Text
			GlobalConfig.Uin = uinEntry.Text
			GlobalConfig.GTk = gtkEntry.Text
			err := utils.SaveConfig(GlobalConfig)
			if err != nil {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "错误",
					Content: err.Error(),
				})
			} else {
				fyne.CurrentApp().SendNotification(&fyne.Notification{
					Title:   "成功",
					Content: "配置已保存",
				})
			}
		},
	}
	return form
}
