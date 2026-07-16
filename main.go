package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func main() {
	devtoolsURL := "ws://127.0.0.1:9222"
	baseURL := "http://neo.nullferatu.com:8081"

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), devtoolsURL)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	// Raw indicator blob data to pass directly to the payload field
	indicatorBlob := "'8.8.8.8' malicious-domain.io"

	type parseResult struct {
		OK     bool   `json:"ok"`
		Status int    `json:"status"`
		Body   string `json:"body"`
	}

	var fetchResult parseResult

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

                return {
                    ok: response.ok,
                    status: response.status,
                    body: await response.text()
                };
            })()
        `, indicatorBlob), &fetchResult, chromedp.EvalAsValue, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
	)

	if err != nil {
		fmt.Println(indicatorBlob)
		log.Fatalf("Automation failed: %v", err)
	}

	if !fetchResult.OK {
		log.Fatalf("Backend returned non-OK response: status=%d body=%s", fetchResult.Status, fetchResult.Body)
	}

	var parsed any
	if err := json.Unmarshal([]byte(fetchResult.Body), &parsed); err != nil {
		log.Fatalf("Failed to parse backend JSON body: %v", err)
	}

	prettyJSON, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		log.Fatalf("Failed to format backend JSON body: %v", err)
	}

	// Escape content before embedding in HTML so raw JSON cannot break markup.
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
        .box { border: 1px solid #333333; white-space: pre-wrap; font-family: monospace; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="title has-text-info mb-5">ThreatPunch Raw Output</h1>
        <div class="box has-background-dark">
            %[2]s
        </div>
    </div>
</body>
</html>`, baseURL, html.EscapeString(string(prettyJSON)))

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
