package main

import (
	_ "QzoneDown-Go/utils/login"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/cmd/fyne_settings/settings"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	Icolor "image/color"
	"log"
	"net/url"
	"os"
)

func main() {
	a := app.NewWithID("io.youngxj.qzonedown")
	icon, err := loadResourceFromFile("Icon.png")
	if err != nil {
		log.Fatalf("LoadResourceFromFile err: %v", err)
	}
	a.SetIcon(icon) // 设置应用图标
	makeTray(a)     // 创建windows系统托盘菜单
	w := a.NewWindow("QQ空间相册下载器 - QzoneDown")

	w.SetMainMenu(makeMenu(a, w)) // 添加顶部菜单
	w.SetMaster()                 // 关闭窗口时退出程序

	content := container.NewStack()
	title := widget.NewLabel("Component name")
	intro := widget.NewLabel("An introduction would probably go\nhere, as well as a")
	intro.Wrapping = fyne.TextWrapWord

	top := container.NewVBox(title, widget.NewSeparator(), intro)
	setTutorial := func(t Tutorial) {
		title.SetText(t.Title)
		isMarkdown := len(t.Intro) == 0
		if !isMarkdown {
			intro.SetText(t.Intro)
		}

		if isMarkdown {
			top.Hide()
		} else {
			top.Show()
		}

		content.Objects = []fyne.CanvasObject{t.View(w)}
		content.Refresh()
	}

	tutorial := container.NewBorder(
		top, nil, nil, nil, content)

	split := container.NewHSplit(makeNav(setTutorial, true), tutorial)
	split.Offset = 0.2
	w.SetContent(split)

	w.Resize(fyne.NewSize(640, 460))

	w.ShowAndRun()
}

// 创建windows系统托盘菜单
func makeTray(a fyne.App) {
	if desk, ok := a.(desktop.App); ok {
		h := fyne.NewMenuItem("Hello", func() {})
		h.Icon = theme.HomeIcon()
		menu := fyne.NewMenu("Hello World", h)
		h.Action = func() {
			log.Println("System tray menu tapped")
			h.Label = "Welcome"
			menu.Refresh()
		}
		desk.SetSystemTrayMenu(menu)
	}
}

// 创建顶部菜单
func makeMenu(a fyne.App, w fyne.Window) *fyne.MainMenu {
	openSettings := func() {
		w := a.NewWindow("外观设置")
		w.SetContent(settings.NewSettings().LoadAppearanceScreen(w))
		w.Resize(fyne.NewSize(440, 520))
		w.Show()
	}
	showAbout := func() {
		w := a.NewWindow("关于")
		w.SetContent(widget.NewRichTextFromMarkdown(`
# QQ空间相册下载器 (Golang)

## 项目简介

QQ空间相册下载器是一个使用 Go 语言编写的工具，用于下载 QQ 空间中的相册图片。用户可以通过提供必要的认证信息来下载自己或指定QQ相册。

## 功能特性

- 支持下载 QQ 空间中（自己或指定QQ）的相册图片
- 自动处理分页，确保所有图片都能下载
- 支持并发下载，提高下载速度
- 提供下载进度显示
- 配置文件支持，保存用户认证信息
- 支持扫码登录自动获取Cookie（需要安装chrome）
- 自动识别 g_tk 和 uin，无需手动输入
- 支持Gui界面操作


## 使用方法

1. 下载对应操作系统的最新可执行文件（支持Windows、Mac、Linux）[QzoneDown-Go Releases](https://github.com/Youngxj/QzoneDown-Go/releases)。
2. 登录 [QQ空间](https://qzone.qq.com) 并获取你的 cookie。
3. 运行程序并输入你的 cookie，g_tk和uin将自动识别。
4. 按照要求输入，程序会自动下载相册中的图片。
5. 图片下载完成后会按照相册名分类保存在images目录中。
`))
		w.Resize(fyne.NewSize(520, 460))
		w.CenterOnScreen()
		w.Show()
	}
	aboutItem := fyne.NewMenuItem("关于", showAbout)
	settingsItem := fyne.NewMenuItem("外观设置", openSettings)
	settingsShortcut := &desktop.CustomShortcut{KeyName: fyne.KeyComma, Modifier: fyne.KeyModifierShortcutDefault}
	settingsItem.Shortcut = settingsShortcut
	w.Canvas().AddShortcut(settingsShortcut, func(shortcut fyne.Shortcut) {
		openSettings()
	})

	projectHome := fyne.NewMenuItem("项目主页", func() {
		u, _ := url.Parse("https://github.com/Youngxj/QzoneDown-Go")
		_ = a.OpenURL(u)
	})
	projectHome.Icon = theme.HomeIcon()

	qzoneLink := fyne.NewMenuItem("QQ空间", func() {
		u, _ := url.Parse("https://qzone.qq.com/")
		_ = a.OpenURL(u)
	})
	qzoneLink.Icon = theme.MailAttachmentIcon()

	helpMenu := fyne.NewMenu("帮助",
		projectHome,
		qzoneLink,
		fyne.NewMenuItem("检查更新", func() {
			//TODO 检查更新
			u, _ := url.Parse("https://github.com/Youngxj/QzoneDown-Go")
			_ = a.OpenURL(u)
		}))

	// a quit item will be appended to our first (File) menu
	file := fyne.NewMenu("开始")
	device := fyne.CurrentDevice()
	if !device.IsMobile() && !device.IsBrowser() {
		file.Items = append(file.Items, fyne.NewMenuItemSeparator(), settingsItem)
	}
	file.Items = append(file.Items, aboutItem)
	main := fyne.NewMainMenu(
		file,
		helpMenu,
	)
	return main
}

const preferenceCurrentMenu = "currentTutorial"

type forcedVariant struct {
	fyne.Theme

	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) Icolor.Color {
	return f.Theme.Color(name, f.variant)
}

// 创建导航菜单
func makeNav(setTutorial func(tutorial Tutorial), loadPrevious bool) fyne.CanvasObject {
	a := fyne.CurrentApp()

	tree := &widget.Tree{
		ChildUIDs: func(uid string) []string {
			return TutorialIndex[uid]
		},
		IsBranch: func(uid string) bool {
			children, ok := TutorialIndex[uid]

			return ok && len(children) > 0
		},
		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Collection Widgets")
		},
		UpdateNode: func(uid string, branch bool, obj fyne.CanvasObject) {
			t, ok := Tutorials[uid]
			if !ok {
				fyne.LogError("Missing tutorial panel: "+uid, nil)
				return
			}
			obj.(*widget.Label).SetText(t.Title)
		},
		OnSelected: func(uid string) {
			if t, ok := Tutorials[uid]; ok {
				for _, f := range OnChangeFuncs {
					f()
				}
				OnChangeFuncs = nil // Loading a page registers a new cleanup.

				a.Preferences().SetString(preferenceCurrentMenu, uid)
				setTutorial(t)
			}
		},
	}

	if loadPrevious {
		currentPref := a.Preferences().StringWithFallback(preferenceCurrentMenu, "welcome")
		tree.Select(currentPref)
	}

	themes := container.NewGridWithColumns(2,
		widget.NewButton("暗黑", func() {
			a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
		}),
		widget.NewButton("明亮", func() {
			a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
		}),
	)

	return container.NewBorder(nil, themes, nil, nil, tree)
}

// loadResourceFromFile 从文件加载资源
func loadResourceFromFile(path string) (fyne.Resource, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &fyne.StaticResource{
		StaticName:    path,
		StaticContent: content,
	}, nil
}
