# Google Scraper (Go)

一个使用 Go 语言编写的高性能谷歌搜索采集器，支持代理池、并发控制、关键词搜索等功能。

## ✨ 功能特色

- 支持 SOCKS5/HTTP 代理
- 支持从 `keyword.txt` 读取关键词批量搜索
- 支持自定义 User-Agent 列表（`ua.txt`）
- 支持多线程并发采集（`-t` 参数）
- 支持关键词批量搜索
- 可自定义 User-Agent
- 自动翻页采集搜索结果
- 可保存为 txt
- 错误重试与超时控制

## 📦 安装与运行
```bash
go run main.go -t 5
```
## 📂 文件结构
 ```plaintext
google-scraper-go/
├── main.go
├── keyword.txt # 关键词列表，每行一个关键词
├── ua.txt # User-Agent 列表，每行一个
├── urls.txt # 默认输出文件
├── README.md
```
### 克隆项目
```bash
git clone https://github.com/shaoyu521/google-scraper-go.git
cd google-scraper-go
go build -o googles.exe 打包exe
```
