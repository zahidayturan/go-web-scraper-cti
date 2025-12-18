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

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const (
	htmlOutputFilename = "scraped_output.html"
	screenshotFilename = "screenshot.png"
	urlListFilename    = "extracted_urls.txt"
	logFilename        = "logs.txt"
	targetsFilename    = "targets.txt"
)

func main() {
	var targets []string

	if len(os.Args) > 1 {
		targets = append(targets, os.Args[1])
	} else {
		file, err := os.Open(targetsFilename)
		if err != nil {
			fmt.Println("Kullanım: program <hedef_url> veya targets.txt dosyası olmalı")
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
			fmt.Println("targets.txt boş.")
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

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var (
		htmlContent   string
		screenshotData []byte
		links         []string
		statusCode    int64 = -1
	)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if resp, ok := ev.(*network.EventResponseReceived); ok {
			if resp.Type == network.ResourceTypeDocument {
				statusCode = int64(resp.Response.Status)
			}
		}
	})

	err = chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return network.Enable().Do(ctx)
		}),
		chromedp.Navigate(targetURL),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
		chromedp.FullScreenshot(&screenshotData, 90),
		chromedp.Evaluate(`
			let urls = [];
			document.querySelectorAll('a').forEach(link => {
				try {
					let u = new URL(link.href, document.baseURI).href;
					if (!urls.includes(u)) urls.push(u);
				} catch (e) {}
			});
			urls;
		`, &links),
	)

	if err != nil {
		logToFile(false, logFile, operationTime, targetURL,
			fmt.Sprintf("Chromedp hatası: %s | HTTP Status: %d", err.Error(), statusCode))
		cleanup(outputDir)
		return
	}

	if err := saveFile(outputDir, htmlOutputFilename, []byte(htmlContent)); err != nil {
		logToFile(false, logFile, operationTime, targetURL,
			fmt.Sprintf("HTML kaydedilemedi | HTTP Status: %d", statusCode))
		cleanup(outputDir)
		return
	}

	if err := saveFile(outputDir, screenshotFilename, screenshotData); err != nil {
		logToFile(false, logFile, operationTime, targetURL,
			fmt.Sprintf("Screenshot kaydedilemedi | HTTP Status: %d", statusCode))
		cleanup(outputDir)
		return
	}

	if err := saveFile(outputDir, urlListFilename, []byte(strings.Join(links, "\n"))); err != nil {
		logToFile(false, logFile, operationTime, targetURL,
			fmt.Sprintf("URL listesi kaydedilemedi | HTTP Status: %d", statusCode))
		cleanup(outputDir)
		return
	}

	logToFile(true, logFile, operationTime, targetURL,
		fmt.Sprintf("İşlem başarılı | HTTP Status: %d", statusCode))
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
	return strings.NewReplacer(
		"https://", "",
		"http://", "",
		"/", "_",
		":", "_",
	).Replace(url)
}