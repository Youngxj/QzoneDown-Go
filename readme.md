# QQ空间相册下载器 (Golang)

## 项目简介
QQ空间相册下载器是一个使用 Go 语言编写的工具，用于下载 QQ 空间中的相册图片。用户可以通过提供必要的认证信息来下载自己的相册。

## 功能特性
- 支持下载 QQ 空间中的相册图片
- 自动处理分页，确保所有图片都能下载
- 支持并发下载，提高下载速度
- 提供下载进度显示
- 配置文件支持，保存用户认证信息

## 使用方法
1. 登录 [QQ空间](https://qzone.qq.com) 并获取你的 cookie、g_tk 和 uin。
2. 运行程序并输入你的 cookie、g_tk 和 uin。
3. 程序会自动下载相册中的图片。

## 安装与运行
### 环境要求
- Go 语言环境 (建议使用 Go 1.16 或更高版本)

### 安装步骤
1. 克隆项目到本地：
   ```bash
   git clone https://github.com/Youngxj/DzoneDown-Go.git
   cd qzone-down
   ```

### 包依赖管理
1. 初始化Go模块：
   ```bash
   go mod init qzone-down
   ```

2. 安装依赖包：
   ```bash
   go mod tidy
   ```

项目使用以下主要依赖包：
- `github.com/spf13/viper` - 配置文件管理
- `github.com/schollz/progressbar/v3` - 进度条显示
- `github.com/pkg/errors` - 错误处理

### 运行
#### 直接运行

在代码目录下执行：
    ```bash
    go run .
    ```
#### 编译成可执行文件

编译：
    ```bash
    go build -o qzone-down.exe
    ```

运行：
    ```bash
    ./qzone-down.exe
    ```