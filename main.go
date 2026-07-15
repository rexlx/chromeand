package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	devtoolsURL := "ws://127.0.0.1:9222"
	baseURL := "http://localhost:8081"

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), devtoolsURL)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	// Raw indicator blob data to pass directly to the payload field
	indicatorBlob := "192.168.1.100\nmalicious-domain.io"

	var apiResponse string
	err := chromedp.Run(ctx,
		// Navigate to the front-end to utilize the browser context
		chromedp.Navigate(baseURL+"/app"),

		// Ensure the DOM has finished loading
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// Dispatches the JSON structure natively via fetch using the existing session cookies
		chromedp.Evaluate(fmt.Sprintf(`
			(async function() {
				const response = await fetch('/parse', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify({ blob: %q })
				});

				return await response.text();
			})()
		`, indicatorBlob), &apiResponse),
	)

	if err != nil {
		log.Fatalf("Automation failed: %v", err)
	}

	// Structuralize local document return matrix using the baseURL variable for assets
	fullHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="%[1]s/static/bulma.min.css">
    <link rel="stylesheet" href="%[1]s/static/material-icons.css">
    <title>ThreatPunch Export Matrix</title>
    <style>
        body { background-color: #000000; color: #ffffff; padding: 2rem; }
        .box { border: 1px solid #333333; white-space: pre-wrap; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <div class="box has-background-dark">
            %[2]s
        </div>
    </div>
</body>
</html>`, baseURL, apiResponse)

	outputPath := "search_result.html"
	err = os.WriteFile(outputPath, []byte(fullHTML), 0644)
	if err != nil {
		log.Fatalf("Failed to write HTML file to target storage path: %v", err)
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		log.Fatalf("Failed to resolve local absolute storage path target: %v", err)
	}

	fmt.Printf("Execution successfully completed. Export matrix parsed to location:\nfile://%s\n", absPath)
}
