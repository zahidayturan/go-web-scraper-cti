# Web Page Scraper

This application is a simple web scraping tool built with **Go** and **chromedp**.

It visits given target URLs, captures:
- Full HTML content
- A full-page screenshot
- All unique links found on the page

Each run is saved into a timestamped folder under the `outputs/` directory and all actions are logged.


## Usage

### 1. Run with a single URL
```bash
go run main.go https://google.com
```

### 2. Run with multiple URLs
Create a **targets.txt** file in the project root:
```bash
https://google.com
https://github.com
```
Then run:
```bash
go run main.go
```

## Output

The application stores the results of each run in a dedicated folder under the `outputs` directory, named using the execution timestamp and the target URL (`outputs/timestamp_url/`). Inside this folder, the scraped page’s full HTML content is saved as `scraped_output.html`, a full-page screenshot is saved as `screenshot.png`, and all unique links found on the page are saved in `extracted_urls.txt`. Additionally, a global log file (`outputs/logs.txt`) is maintained to record the outcome of every operation. If any step of the process fails, all files created for that run are automatically deleted, the corresponding output folder is removed, and the failure is recorded in the log file.
