package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"context"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/proxy"
)

func init() {
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子
}

// 从文件中读取每一行内容到字符串切片
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err // 返回错误信息
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text()) // 将每行文本添加到切片中
	}

	return lines, scanner.Err()
}

// 保存结果 URL 到指定文件中
func saveURLsToFile(filename, query string, urls []string) {
	// 打开文件
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("打开文件时出错: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// 过滤掉包含 google.com 和 site: 的 URL
	filteredURLs := []string{}
	for _, u := range urls {
		if !strings.Contains(u, "google.com") && !strings.Contains(u, "site:") {
			filteredURLs = append(filteredURLs, u)
		}
	}

	// 写入过滤后的 URL
	for _, u := range filteredURLs {
		_, _ = writer.WriteString(u + "\n") // 写入每个 URL
	}

	writer.Flush() // 刷新缓冲区，确保写入文件
}

// 创建 SOCKS5 代理客户端
func createSocks5ProxyClient(gate, proxyUser, proxyPass string) (*http.Client, error) {
	auth := &proxy.Auth{
		User:     proxyUser, // 代理用户名
		Password: proxyPass, // 代理密码
	}
	dialer, err := proxy.SOCKS5("tcp", gate, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建 SOCKS5 代理时出错: %v", err)
	}

	// 配置 HTTP 客户端使用 SOCKS5
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 设置请求超时时间
	}
	return client, nil
}

// 使用代理访问谷歌搜索并获取结果 URL
func fetchGoogleSearchResults(client *http.Client, query string, userAgents []string, page int) ([]string, error) {
	// 修改 Google 搜索 URL，支持翻页功能
	searchURL := fmt.Sprintf("https://www.google.com/search?q=%s&start=%d&num=10000", url.QueryEscape(query), page*100)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	// 设置随机 User-Agent
	ua := userAgents[rand.Intn(len(userAgents))]
	req.Header.Set("User-Agent", ua)

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取搜索结果时出错: %v", err)
	}
	defer response.Body.Close()

	// 使用 goquery 解析 HTML
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 时出错: %v", err)
	}

	var urls []string
	urlMap := make(map[string]bool) // 用于去重
	doc.Find("a[href]").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists && strings.HasPrefix(href, "/url?q=") {
			re := regexp.MustCompile(`^/url\?q=(.+?)&`)
			match := re.FindStringSubmatch(href)
			if len(match) == 2 {
				decodedURL, err := url.QueryUnescape(match[1])
				if err != nil {
					decodedURL = match[1] // 解码失败时使用原始 URL
				}
				if !urlMap[decodedURL] { // 检查 URL 是否已保存
					urlMap[decodedURL] = true
					urls = append(urls, decodedURL)
				}
			}
		}
	})

	return urls, nil
}

// 执行多任务搜索，使用 SOCKS5 代理
func testSocks5(concurrency int) {
	// 代理账号信息
	username := "" // 代理用户名
	password := ""     // 代理密码
	country := ""          // 国家代码
	gate := "" // 代理网关地址
	wg := sync.WaitGroup{}

	// 读取 User-Agent 和关键词文件
	userAgents, err := readLines("ua.txt")
	if err != nil {
		log.Fatalf("读取 ua.txt 文件时出错: %v", err)
	}
	keywords, err := readLines("keyword.txt")
	if err != nil {
		log.Fatalf("读取 keyword.txt 文件时出错: %v", err)
	}

	// 遍历关键词，为每个关键词启动并发任务
	for _, keyword := range keywords {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()

			// 构造代理用户名，包含随机会话 ID
			proxyUser := fmt.Sprintf("%s-country-%s-sid-%d", username, country, rand.Int()%100000)

			// 创建 SOCKS5 客户端
			client, err := createSocks5ProxyClient(gate, proxyUser, password)
			if err != nil {
				log.Printf("创建 SOCKS5 客户端时出错: %v", err)
				return
			}

			// 按照并发数进行翻页搜索，最大翻页次数为5
			var allUrls []string
			for page := 0; page < 5; page++ {
				urls, err := fetchGoogleSearchResults(client, query, userAgents, page)
				if err != nil {
					log.Printf("获取关键词 %s 的第 %d 页搜索结果时出错: %v", query, page+1, err)
					break
				}
				allUrls = append(allUrls, urls...)
			}

			// 保存结果到文件
			saveURLsToFile("urls.txt", query, allUrls)
		}(keyword) // 将关键词作为参数传递
	}

	wg.Wait() // 等待所有任务完成
}

func main() {
	// 输出版权信息
	fmt.Println("=====================================")
	fmt.Println("   Google采集工具  By 小黑")
	fmt.Println("   “我不是黑子，我只是路过的批评家。”")
	fmt.Println("   “这也能吹？小黑子笑疯了。”")
	fmt.Println("   “别骂了别骂了，小黑子快坚持不住了！”")
	fmt.Println("   https://t.me/xxxh72 ")
	fmt.Println("=====================================")
	fmt.Println()
	// 通过命令行参数获取并发数
	concurrency := flag.Int("t", 10, "设置并发任务数")
	flag.Parse()

	// 开始执行多任务搜索
	testSocks5(*concurrency)
}