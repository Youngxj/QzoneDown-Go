package main

import (
	"QzoneDown-Go/api"
	"QzoneDown-Go/utils"
	"QzoneDown-Go/utils/progress"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// 选中的相册ID
var checkPhotos []api.PhotoListPicStruct

// 相册元数据列表
var pList []api.PhotoListPicStruct

// 复选框组件列表
var newCheckBox []fyne.CanvasObject

// 下载按钮组件
var downloadBtn *widget.Button

// 相册图片下载成功数量
var photoDownSuccessNum int32 = 0

// 相册图片下载成功总数量
var photoDownSuccessNumCount int32 = 0

var downAlbumnameBind = binding.NewString()
var downNumberBind = binding.NewString()
var downResultBind = binding.NewString()

var downAlbumnameLabel *widget.Label
var downNumberLabel *widget.Label
var downResultLabel *widget.Label

// 下载进度
var bar progress.Bar

// 相册列表视图
func imagesList(w fyne.Window) fyne.CanvasObject {
	clearSelection()
	newCheckBox = make([]fyne.CanvasObject, 0)

	var errMsgLabelBox *fyne.Container
	var errMsgLabelText = binding.NewString()
	errMsgLabel := widget.NewLabelWithData(errMsgLabelText)
	errMsgLabelBox = container.NewHBox(errMsgLabel)
	errMsgLabelBox.Hide()

	currentPage := 1
	photoList, err := api.GetPhotoList(currentPage)
	if err != nil {
		errMsgLabelBox.Show()
		errMsgLabelText.Set(err.Error() + "\n检查相册失败，请先检查参数配置")
	}

	for k, VFeeds := range photoList.Data.VFeeds {
		VFeeds.Pic.Id = k
		pList = append(pList, VFeeds.Pic)

		// 构建复选框
		newCheckBox = append(newCheckBox, widget.NewCheck(fmt.Sprintf("%s(%d)", VFeeds.Pic.Albumname, VFeeds.Pic.Albumnum), func(value bool) {
			checkPhotos = getCheckPhotos()
			if len(checkPhotos) == 0 {
				downloadBtn.Disable()
				downloadBtn.SetText("请选择要下载的相册")
			} else {
				downloadBtn.Enable()
				downloadBtn.Importance = widget.WarningImportance
				downloadBtn.SetText(fmt.Sprintf("下载中%d个相册", len(checkPhotos)))
			}
		}))
	}

	// 构建网格布局
	grid := container.NewGridWithColumns(6, newCheckBox...)
	// 构建滚动容器
	scrollContainer := container.NewScroll(grid)
	//构建边框布局
	gridBox := container.NewBorder(container.NewVBox(
		selectBtnGroup(w),
		errMsgLabelBox,
	), downBtn(), nil, nil, scrollContainer)
	// 构建垂直分栏布局
	split := container.NewVSplit(gridBox, downImageBox())
	split.Refresh()
	split.Offset = 0.6
	downInit()
	return split
}

// 选择按钮组
func selectBtnGroup(w fyne.Window) fyne.CanvasObject {
	return container.NewHBox(
		widget.NewButton("全选", func() {
			selectAll()
		}),
		widget.NewButton("反选", func() {
			selectInverse()
		}),
		widget.NewButton("清空选择", func() {
			clearSelection()
		}),
		widget.NewButton("刷新列表", func() {
			imagesList(w)
			downloadBtn.Refresh()
		}),
	)
}

// 下载图片的信息框
func downImageBox() fyne.CanvasObject {
	downAlbumnameLabel = widget.NewLabelWithData(downAlbumnameBind)
	downNumberLabel = widget.NewLabelWithData(downNumberBind)
	downResultLabel = widget.NewLabelWithData(downResultBind)
	vbox := container.NewVBox(container.NewHBox(downAlbumnameLabel), downNumberLabel, downResultLabel)
	return vbox
}

// 下载初始化
func downInit() {
	photoDownSuccessNumCount = 0
	downloadBtn.SetText("请选择要下载的相册")
	downloadBtn.Disable()
	downAlbumnameBind.Set("")
	downNumberBind.Set("")
	downResultBind.Set("")
	downAlbumnameLabel.Show()
	downNumberLabel.Show()
}

// 下载完成事件
func downSuccess() {
	//下载完成后隐藏下载进度
	downAlbumnameLabel.Hide()
	downNumberLabel.Hide()
	downResultBind.Set(fmt.Sprintf("下载完成%d个相册，成功下载%d张图片", len(checkPhotos), photoDownSuccessNumCount))
	downloadBtn.SetText("下载完成")
	downloadBtn.Enable()
	photoDownSuccessNumCount = 0
}

// 下载按钮
func downBtn() *widget.Button {
	downloadBtn = widget.NewButtonWithIcon("请选择要下载的相册", theme.DownloadIcon(), func() {
		checkPhotos = getCheckPhotos()
		if len(checkPhotos) == 0 {
			return
		}
		downloadBtn.SetText("下载中...")
		// 下载选中的相册
		go func() {
			var successAlbumname []string
			for _, photo := range checkPhotos {
				bar = progress.Bar{} // 在这里重新初始化bar，否则会出现进度条叠加的情况
				bar.IsGui = true
				bar.NewOptionWithGraph(0, int64(photo.Albumnum), "✨")
				photoDownSuccessNum = 0 // 重置下载成功数量

				downAlbumnameBind.Set(fmt.Sprintf("正在下载相册：%s 共%d张图片", photo.Albumname, photo.Albumnum))
				photos, err := api.GetPhotoImages2(photo.Albumid, photo.Albumnum)
				if err != nil {
					downResultBind.Set(fmt.Sprintf("下载相册：%s 失败", photo.Albumname))
					continue
				}
				for _, photoUrl := range photos {
					err = downloadImage(photoUrl, photo.Albumname)
					if err != nil {
						downResultBind.Set(fmt.Sprintf("下载图片：%s 失败", photoUrl))
						continue
					}
				}
				bar.Finish()
				successAlbumname = append(successAlbumname, photo.Albumname)
			}
			fyne.Do(func() {
				downSuccess()
			})
		}()
	})
	downloadBtn.Disable()
	return downloadBtn
}

// 下载图片
func downloadImage(photoUrl string, Albumname string) (err error) {
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
		downNumberBind.Set(bar.Play(int64(photoDownSuccessNum)))
	}(photoUrl)
	wg.Wait() // 等待所有 goroutine 完成
	return err
}

// 获取选中的相册
func getCheckPhotos() []api.PhotoListPicStruct {
	tempList := make([]api.PhotoListPicStruct, 0, len(newCheckBox))
	for k, check := range newCheckBox {
		if check.(*widget.Check).Checked {
			tempList = append(tempList, pList[k])
		}
	}
	return tempList
}

// 全选功能
func selectAll() {
	// 清空现有选择
	checkPhotos = checkPhotos[:0]

	// 确保循环范围不超过任一切片的长度
	maxLen := len(newCheckBox)
	if len(pList) < maxLen {
		maxLen = len(pList)
	}

	// 勾选所有复选框并更新选择列表（使用安全的循环范围）
	for i := 0; i < maxLen; i++ {
		// 检查索引是否有效
		if i < len(newCheckBox) && i < len(pList) {
			newCheckBox[i].(*widget.Check).SetChecked(true)
			checkPhotos = append(checkPhotos, pList[i])
		}
	}
}

// 反选功能
func selectInverse() {
	// 确保循环范围不超过任一切片的长度
	maxLen := len(newCheckBox)
	if len(pList) < maxLen {
		maxLen = len(pList)
	}
	// 确保循环范围不超过任一切片的长度
	for i := 0; i < maxLen; i++ {
		if i < len(newCheckBox) && i < len(pList) {
			current := newCheckBox[i].(*widget.Check).Checked
			newCheckBox[i].(*widget.Check).SetChecked(!current)
			if !current {
				// 原本未选中，现在选中
				checkPhotos = append(checkPhotos, pList[i])
			} else {
				// 原本选中，现在取消
				checkPhotos = removeElement(checkPhotos, pList[i])
			}
		}
	}
}

// 清空选择
func clearSelection() {
	for _, check := range newCheckBox {
		check.(*widget.Check).SetChecked(false)
	}
	checkPhotos = checkPhotos[:0]
}

// 移除选中
func removeElement(photos []api.PhotoListPicStruct, picStruct api.PhotoListPicStruct) []api.PhotoListPicStruct {
	for i, v := range photos {
		if v.Albumid == picStruct.Albumid {
			return append(photos[:i], photos[i+1:]...)
		}
	}
	return photos
}
