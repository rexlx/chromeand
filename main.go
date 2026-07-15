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

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), devtoolsURL)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// 120-second threshold matching your environment parameters
	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	// Define the raw JSON payload data you want to submit to your endpoint
	bulkPayloadJSON := `{"iocs": ["192.168.1.100", "malicious-domain.io"]}`

	var apiResponse string
	err := chromedp.Run(ctx,
		// Navigate to your application front-end to establish/use the authenticated session context
		chromedp.Navigate("http://localhost:8081/app"),

		// Wait strictly for the page body to assure the execution runtime is active
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// Hit your backend route natively using fetch; browser cookies are automatically attached
		chromedp.Evaluate(fmt.Sprintf(`
            (async function() {
                const response = await fetch('/api/v1/bulk-search', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(%s)
                });

                return await response.text();
            })()
        `, bulkPayloadJSON), &apiResponse),
	)

	if err != nil {
		log.Fatalf("Automation stalled or failed: %v", err)
	}

	// Structuralize local document return matrix
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
        .box { border: 1px solid #333333; white-space: pre-wrap; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <div class="box has-background-dark">
            %s
        </div>
    </div>
</body>
</html>`, apiResponse)

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
