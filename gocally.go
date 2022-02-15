package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/debug"
	"os"
	"strings"
	"sync"
)

var (
	baseUrl = "https://so.gushiwen.cn/"
	// 存放诗词链接的数据管道
	chanPoetry        chan map[string]string
	chanPoetryContent chan map[string]string
	waitGroup         sync.WaitGroup
	// 用于监控协程
	chanTask    chan string
	chanSubTask chan string
	// 存放诗词的目录
	mainDir string
)

func readContent() {
	for key := range chanPoetryContent {
		// 获取链接
		href := baseUrl + key["href"]
		// 文件名称
		fileName := key["title"]
		dirName := key["text"]
		s := colly.NewCollector(colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36"), colly.MaxDepth(1), colly.Debugger(&debug.LogDebugger{}))
		s.OnHTML("div[class='left'] ", func(e *colly.HTMLElement) {
			name := e.ChildText("div[class='sons'][id='sonsyuanwen'] > div[class='cont'] > h1")
			author := e.ChildText("div[class='sons'][id='sonsyuanwen'] > div[class='cont'] > p[class='source'] > a")
			text := e.ChildText("div[class='sons'][id='sonsyuanwen'] > div[class='cont'] > div[class='contson']")
			content := name + "\r\n" + author
			text = strings.Replace(text, "。", "。\r\n", 99999)
			text = strings.Replace(text, "；", "；\r\n", 99999)
			text = strings.Replace(text, ")", ")\r\n", 99999)
			path := mainDir + "/诗词/" + dirName
			if len(fileName) > 0 {
				path = path + "/" + fileName
			}
			if len(name) == 0 {
				return
			} else {
				// 创建目录
				makeDir(path)
				file, _ := os.OpenFile(path+"/"+name+".txt", os.O_CREATE|os.O_WRONLY, 0644)
				file.WriteString(content + "\r\n")
				file.WriteString(text + "\r\n")
				// 关闭资源
				file.Close()
			}

		})
		s.Visit(href)
	}
	waitGroup.Done()
}

// 创建目录
func makeDir(path string) bool {
	if _, err := os.Stat(path); err == nil {
		fmt.Println("path exists 1", path)
	} else {
		err := os.MkdirAll(path, 0711)
		if err != nil {
			//log.Println("Error creating directory")
			//log.Println(err)
			return true
		}
	}
	return false
}


// 任务统计协程
func CheckDealHref() {
	var count int
	for {
		dealHref := <- chanTask
		fmt.Printf("%s 完成了任务\n", dealHref)
		count++
		if count == 30 {
			close(chanPoetry)
			break
		}
	}
	waitGroup.Done()
}

// 子链接任务统计协程
func CheckSubDealHref() {
	var count int
	for {
		url := <-chanSubTask
		fmt.Printf("%s 完成了爬取子链接任务\n", url)
		count++
		if count == 30 {
			close(chanPoetryContent)
			break
		}
	}
	waitGroup.Done()
}

// 首页链接
func mainHref() {
	// 采集器
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36"),
		colly.MaxDepth(1),
		colly.Debugger(&debug.LogDebugger{}))
	// 诗词列表，回调函数
	c.OnHTML("div[class='right'] > div[class='sons'] > div[class='cont']", func(e *colly.HTMLElement) {
		//
		e.ForEach("a", func(i int, item *colly.HTMLElement) {
			attr := item.Attr("href")
			text := item.Text
			m := make(map[string]string, 10)
			m["title"] = text
			m["href"] = attr
			// 把连接放进通道内
			chanPoetry <- m
		})
	})
	// 访问网址
	err := c.Visit(baseUrl)
	if err != nil {
		fmt.Println(err.Error())
	}
	waitGroup.Done()
}

// 处理链接的方法
func dealHref() {
	for poetry := range chanPoetry {
		href := poetry["href"]
		text := poetry["title"]
		s := colly.NewCollector(colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.163 Safari/537.36"), colly.MaxDepth(1), colly.Debugger(&debug.LogDebugger{}))
		s.OnHTML("div[class='left'] > div[class='sons'] ", func(e *colly.HTMLElement) {
			e.ForEach("div[class='typecont']", func(i int, item *colly.HTMLElement) {
				//诗词链接
				title := item.ChildText("div[class='bookMl']")
				item.ForEach("span", func(i int, element *colly.HTMLElement) {
					subHref := element.ChildAttr("a", "href")
					// 读取内容
					// 实际内容的通道
					m := make(map[string]string, 10)
					m["title"] = title
					m["href"] = subHref
					m["text"] = text
					chanPoetryContent <- m
				})

			})
		})
		s.Visit(href)
	}
	waitGroup.Done()
}

func main() {
	// 输入存放资料的目录
	for {
		fmt.Println("请存放诗词的目录：")
		fmt.Scanln(&mainDir)
		if len(mainDir) == 0 {
			fmt.Println("未检测到内容，请重新输入")
		} else {
			if _, err := os.Stat(mainDir); err != nil {
				fmt.Println("未查询到该目录，自动创建中......")
				makeDir(mainDir)
				fmt.Println("创建完成，任务开启")
				break
			}
			break
		}
	}
	// 1.初始化管道
	chanPoetry = make(chan map[string]string, 1000000)
	chanPoetryContent = make(chan map[string]string, 1000000)
	chanTask = make(chan string, 30)
	chanSubTask = make(chan string, 30)
	// 向通道内添加链接
	waitGroup.Add(1)
	go mainHref()
	// 读取链接
	for i := 0; i < 30; i++ {
		waitGroup.Add(1)
		go dealHref()
	}
	// 任务统计协程，统计30个任务是否都完成，完成则关闭管道
	waitGroup.Add(1)
	go CheckDealHref()

	// 读取子链接
	for i := 0; i < 30; i++ {
		waitGroup.Add(1)
		go readContent()
	}
	// 任务统计协程，统计30个任务是否都完成，完成则关闭管道
	waitGroup.Add(1)
	go CheckSubDealHref()
	waitGroup.Wait()
}
