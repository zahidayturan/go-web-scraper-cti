package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const htmlOutputFilename = "scraped_output.html"
const screenshotFilename = "screenshot.png"
const urlListFilename = "extracted_urls.txt"
const logFilename = "logs.txt"
const targetsFilename = "targets.txt"

func main() {
	var targets []string

	if len(os.Args) > 1 {
		targets = append(targets, os.Args[1])
	} else {
		file, err := os.Open(targetsFilename)
		if err != nil {
			fmt.Println("Kullanım: program <hedef_url> veya ana dizinde targets.txt dosyası olmalı")
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				targets = append(targets, line)
			}
		}

		if len(targets) == 0 {
			fmt.Println("targets.txt bulundu ancak içinde URL yok.")
			return
		}
	}
	for _, url := range targets {
		processURL(url)
	}
}

func processURL(targetURL string) {
	operationTime := time.Now().Format("2006-01-02_15-04-05")
	outputDir := fmt.Sprintf("outputs/%s_%s", operationTime, sanitizeURL(targetURL))

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Klasör oluşturulamadı: %v", err)
		return
	}

	logFile, err := os.OpenFile(filepath.Join("outputs", logFilename),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Log dosyası açılamadı: %v", err)
		cleanup(outputDir)
		return
	}
	defer logFile.Close()

	log.Printf("İşlem başlatıldı: %s - %s", operationTime, targetURL)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var htmlContent string
	var screenshotData []byte
	var links []string

	err = chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
		chromedp.FullScreenshot(&screenshotData, 90),
		chromedp.Evaluate(`
			let urls = [];
			document.querySelectorAll('a').forEach(link => {
				try {
					let u = new URL(link.href, document.baseURI).href;
					if (urls.indexOf(u) === -1) urls.push(u);
				} catch (e) {}
			});
			urls;
		`, &links),
	)

	if err != nil {
		logToFile(false, logFile, operationTime, targetURL, err.Error())
		cleanup(outputDir)
		return
	}

	if err := saveFile(outputDir, htmlOutputFilename, []byte(htmlContent)); err != nil {
		logToFile(false, logFile, operationTime, targetURL, "HTML kaydedilemedi")
		cleanup(outputDir)
		return
	}

	if err := saveFile(outputDir, screenshotFilename, screenshotData); err != nil {
		logToFile(false, logFile, operationTime, targetURL, "Screenshot kaydedilemedi")
		cleanup(outputDir)
		return
	}

	linkContent := strings.Join(links, "\n")
	if err := saveFile(outputDir, urlListFilename, []byte(linkContent)); err != nil {
		logToFile(false, logFile, operationTime, targetURL, "URL listesi kaydedilemedi")
		cleanup(outputDir)
		return
	}

	logToFile(true, logFile, operationTime, targetURL, "İşlem başarılı")
}

func saveFile(outputDir, filename string, data []byte) error {
	return ioutil.WriteFile(filepath.Join(outputDir, filename), data, 0644)
}

func cleanup(outputDir string) {
	_ = os.RemoveAll(outputDir)
}

func logToFile(status bool, logFile *os.File, operationTime, targetURL, message string) error {
	statusMsg := "ERR"
	if status {
		statusMsg = "OK"
	}
	_, err := logFile.WriteString(
		fmt.Sprintf("%s | %s - %s - %s\n", statusMsg, operationTime, targetURL, message),
	)
	return err
}

func sanitizeURL(url string) string {
	return strings.NewReplacer("https://", "", "http://", "", "/", "_").Replace(url)
}
