package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	devtoolsURL := "ws://127.0.0.1:9222"

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), devtoolsURL)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	var resultHTML string
	err := chromedp.Run(ctx,
		chromedp.Navigate("http://localhost:8081/app"),

		chromedp.WaitVisible(`#sidebarSearch`, chromedp.ByID),
		chromedp.Click(`#sidebarSearch`, chromedp.ByID),

		chromedp.WaitVisible(`#userSearch`, chromedp.ByID),

		chromedp.Evaluate(`
            (function() {
                const textarea = document.getElementById('userSearch');
                textarea.value = "192.168.1.100\nmalicious-domain.io";
                
                textarea.dispatchEvent(new Event('input', { bubbles: true }));
                textarea.dispatchEvent(new Event('change', { bubbles: true }));
            })()
        `, nil),

		chromedp.WaitVisible(`#searchButton`, chromedp.ByID),
		chromedp.Click(`#searchButton`, chromedp.ByID),

		chromedp.Evaluate(`
            (function() {
                const btn = document.getElementById('searchButton');
                if (btn) {
                    btn.focus();
                    btn.click();
                }
            })()
        `, nil),

		chromedp.WaitVisible(`#iocSelectionArea, #matchBox`, chromedp.ByID),
		chromedp.Sleep(5*time.Second),

		chromedp.InnerHTML(`#matchBox`, &resultHTML, chromedp.ByID),
	)

	if err != nil {
		log.Fatalf("Automation stalled or failed: %v", err)
	}

	fullHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bulma@1.0.2/css/bulma.min.css">
    <link rel="stylesheet" href="https://fonts.googleapis.com/icon?family=Material+Icons">
    <title>ThreatPunch Export Matrix</title>
    <style>
        body { background-color: #000000; color: #ffffff; padding: 2rem; }
        .box { border: 1px solid #333333; }
    </style>
</head>
<body>
    <div class="container">
        <div class="box has-background-dark">
            %s
        </div>
    </div>
</body>
</html>`, resultHTML)

	outputPath := "search_result.html"
	err = os.WriteFile(outputPath, []byte(fullHTML), 0644)
	if err != nil {
		log.Fatalf("Failed to write HTML file to target storage path: %v", err)
	}

	backendURL := "http://localhost:8080/api/backend-endpoint"
	req, err := http.NewRequestWithContext(ctx, "POST", backendURL, bytes.NewBuffer([]byte(fullHTML)))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/html")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to send data to backend: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Fatalf("Backend returned unexpected status: %s", resp.Status)
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		log.Fatalf("Failed to resolve local absolute storage path target: %v", err)
	}

	fmt.Printf("Execution successfully completed. Export matrix parsed to location:\nfile://%s\n", absPath)
}
