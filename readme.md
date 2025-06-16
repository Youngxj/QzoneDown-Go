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

1. 下载对应操作系统的最新可执行文件[QzoneDown-Go](https://github.com/Youngxj/QzoneDown-Go/releases)。
2. 登录 [QQ空间](https://qzone.qq.com) 并获取你的 cookie、g_tk 和 uin。
3. 运行程序并输入你的 cookie、g_tk 和 uin。
4. 选择要下载的相册及图片。
5. 下载完成后打开`images`目录查看下载的图片。

## 开发与调试

### 环境要求

- Go 语言环境 (建议使用 Go 1.16 或更高版本)

### 安装步骤

1. 克隆项目到本地：
   ```bash
   git clone https://github.com/Youngxj/QzoneDown-Go.git
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

- `github.com/jedib0t/go-pretty/v6/table` - 表格输出
- `github.com/fatih/color` - 文字颜色输出
- `github.com/cheggaaa/pb/v3` - 进度条显示

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