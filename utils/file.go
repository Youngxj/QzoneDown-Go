package utils

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	DownloadLinksDir = "download_links"
	DownloadImgsDir  = "download_imgs"
)

// EnsureDir 检查并创建目录
func EnsureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("无法创建目录 %s: %w", dir, err)
		}
	}
	return nil
}

// ReadLinksFromFile 从指定文件中读取链接，每行一个链接
func ReadLinksFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var links []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		links = append(links, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return links, nil
}

// DownloadFileWithAlbum 带相册名的下载函数
func DownloadFileWithAlbum(url string, albumName string) error {
	albumDir := filepath.Join(DownloadImgsDir, albumName)
	if err := EnsureDir(albumDir); err != nil {
		return err
	}
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", response.StatusCode)
	}

	fileName := filepath.Base(url)
	data := []byte(fileName)
	fileNameMd5 := filepath.Join(albumDir, fmt.Sprintf("%x", md5.Sum(data))+".jpg")

	file, err := os.Create(fileNameMd5)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
}