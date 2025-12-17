package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
	"github.com/chromedp/chromedp"
)

const htmlOutputFilename = "scraped_output.html"
const screenshotFilename = "screenshot.png"
const urlListFilename = "extracted_urls.txt"
const logFilename = "logs.txt"

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Kullanım: %s <hedef_URL>\n", os.Args[0])
		log.Fatal("Lütfen hedef URL'yi komut satırı argümanı olarak sağlayın.")
	}
	targetURL := os.Args[1]
	operationTime := time.Now().Format("2006-01-02_15-04-05")
	outputDir := fmt.Sprintf("outputs/%s_%s", operationTime, sanitizeURL(targetURL))

	// Klasörü oluştur
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Klasör oluşturulamadı: %v", err)
	}

	logFile, err := os.OpenFile(filepath.Join("outputs", logFilename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Log dosyası açılamadı: %v", err)
	}
	defer logFile.Close()

	log.Printf("İşlem başlatıldı: %s - %s\n", operationTime, targetURL)

	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithLogf(log.Printf),
	)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var htmlContent string
	var screenshotData []byte
	var links []string

	fmt.Printf("Hedef URL'ye bağlanılıyor: %s\n", targetURL)

	err = chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
		chromedp.FullScreenshot(&screenshotData, 90),
		chromedp.Evaluate(`
			let urls = [];
			document.querySelectorAll('a').forEach(link => {
				try {
					let absoluteUrl = new URL(link.href, document.baseURI).href;
					if (absoluteUrl && urls.indexOf(absoluteUrl) === -1) {
						urls.push(absoluteUrl);
					}
				} catch (e) {}
			});
			urls;
		`, &links),
	)

	if err != nil {
		result := "Bilinmeyen hata"

		if err == context.DeadlineExceeded {
			result = "Zaman aşımı"
		} else {
			result = err.Error()
		}

		logToFile(false, logFile, operationTime, targetURL, "Başarısız: "+result)
		cleanup(outputDir)

		log.Printf("Error: İşlem sırasında hata oluştu: %v", err)
		return
	}

	if err := saveFile(outputDir, htmlOutputFilename, []byte(htmlContent)); err != nil {
		log.Printf("Error: HTML içeriği kaydedilemedi: %v", err)
		cleanup(outputDir)
		logToFile(false, logFile, operationTime, targetURL, "HTML içeriği kaydedilemedi")
		return
	}
	logToFile(true, logFile, operationTime, targetURL, fmt.Sprintf("HTML içeriği '%s' dosyasına kaydedildi.", htmlOutputFilename))


	if err := saveFile(outputDir, screenshotFilename, screenshotData); err != nil {
		log.Printf("Error: Ekran görüntüsü kaydedilemedi: %v", err)
		cleanup(outputDir)
		logToFile(false, logFile, operationTime, targetURL, "Ekran görüntüsü kaydedilemedi")
		return
	}
	logToFile(true, logFile, operationTime, targetURL, fmt.Sprintf("Ekran görüntüsü '%s' dosyasına kaydedildi.", screenshotFilename))


	linkContent := ""
	for _, link := range links {
		linkContent += link + "\n"
	}

	if err := saveFile(outputDir, urlListFilename, []byte(linkContent)); err != nil {
		log.Printf("Error: URL listesi kaydedilemedi: %v", err)
		cleanup(outputDir)
		logToFile(false, logFile, operationTime, targetURL, "URL listesi kaydedilemedi")
		return
	}
	logToFile(true, logFile, operationTime, targetURL, fmt.Sprintf("%d adet URL '%s' dosyasına kaydedildi.", len(links), urlListFilename))

	logToFile(true, logFile, operationTime, targetURL, "İşlem başarılı")
}

func saveFile(outputDir, filename string, data []byte) error {
	fullPath := filepath.Join(outputDir, filename)
	return ioutil.WriteFile(fullPath, data, 0644)
}

func cleanup(outputDir string) {
	if err := os.RemoveAll(outputDir); err != nil {
		log.Printf("Error: Klasör silinemedi: %v", err)
	}
}

func logToFile(status bool, logFile *os.File, operationTime, targetURL, message string) error {
	statusMsg := "OK"
	if status {
		statusMsg = "ERR"
	}

	logMessage := fmt.Sprintf(
		"%s | %s - %s - %s\n",
		statusMsg,
		operationTime,
		targetURL,
		message,
	)

	_, err := logFile.WriteString(logMessage)
	return err
}

func sanitizeURL(url string) string {
	return filepath.Base(url)
}
