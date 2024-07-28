package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	DownloadList []string `yaml:"downloadList"`
	Datachunk    int64    `yaml:"datachunk"`
	Timelapse    int      `yaml:"timelapse"`
	TrafficFile  string   `yaml:"trafficFile"`
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func loadTraffic(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer file.Close()

	var totalTraffic int64
	_, err = fmt.Fscanf(file, "%d", &totalTraffic)
	if err != nil {
		return 0, err
	}

	return totalTraffic, nil
}

func saveTraffic(filename string, totalTraffic int64) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "%d", totalTraffic)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	totalTraffic, err := loadTraffic(config.TrafficFile)
	if err != nil {
		log.Fatalf("加载流量记录失败: %v", err)
	}

	log.Printf("上次运行总消耗流量: %d MB\n", totalTraffic)

	i := 0
	for {
		i = i + 1
		log.Println("第", i, "轮下载开始")
		for n, url := range config.DownloadList {
			timeSleep := time.Duration(rand.Intn(10)) * time.Second
			log.Println("第", n, "个下载结束，等待", timeSleep)
			time.Sleep(timeSleep)
			log.Println("开始下载：", url)
			resp, err := http.Get(url)
			if err != nil {
				log.Println("Get failed:", err)
			} else {
				defer resp.Body.Close()
				contentLength := resp.ContentLength / 1024 / 1024

				file := io.Discard

				var alreadyDown int64

				for range time.Tick(time.Duration(config.Timelapse) * time.Second) {
					n, err := io.CopyN(file, resp.Body, config.Datachunk)
					if err != nil {
						if err == io.EOF {
							log.Println(url, "下载完成")
						} else {
							log.Println("写入失败:", err)
						}
						break
					}
					alreadyDown = alreadyDown + n/1024/1024
					totalTraffic += n / 1024 / 1024

					log.Println("已下载" + fmt.Sprint(alreadyDown) + "兆，完成百分之" + fmt.Sprint(alreadyDown*10/contentLength*10))
				}
			}
		}
		err = saveTraffic(config.TrafficFile, totalTraffic)
		if err != nil {
			log.Fatalf("保存流量记录失败: %v", err)
		}
		log.Printf("当前总消耗流量: %d MB\n", totalTraffic)
	}
}
