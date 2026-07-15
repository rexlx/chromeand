package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func main() {
	devtoolsURL := "ws://127.0.0.1:9222"
	targetURL := "https://example.com/dashboard"
	backendEndpoint := "/api/v1/telemetry"

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), devtoolsURL)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 120*time.Second)
	defer cancelTimeout()

	payloadData := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"client":    "automated-uplink-agent",
		"metrics": map[string]interface{}{
			"batchSize": 5000,
			"status":    "active",
		},
	}

	payloadJSON, err := json.Marshal(payloadData)
	if err != nil {
		log.Fatalf("Failed to marshal generic payload: %v", err)
	}

	loadCh := make(chan struct{}, 1)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if _, ok := ev.(*page.EventLoadEventFired); ok {
			select {
			case loadCh <- struct{}{}:
			default:
			}
		}
	})

	var apiResponse string
	err = chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			select {
			case <-loadCh:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}),
		chromedp.Evaluate(fmt.Sprintf(`
            (async function() {
                const pageMetaContext = {
                    title: document.title,
                    url: window.location.href,
                    token: document.querySelector('meta[name="csrf-token"]')?.getAttribute('content') 
                           || document.querySelector('input[name="csrf"]')?.value 
                           || ""
                };

                const outboundPayload = {
                    meta: pageMetaContext,
                    data: %s
                };
                
                try {
                    const response = await fetch('%s', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'X-CSRF-Token': pageMetaContext.token,
                            'X-Requested-With': 'XMLHttpRequest'
                        },
                        body: JSON.stringify(outboundPayload)
                    });
                    
                    return await response.text();
                } catch (e) {
                    return "Transmission failed: " + e.message;
                }
            })()
        `, string(payloadJSON), backendEndpoint), &apiResponse),
	)

	if err != nil {
		log.Fatalf("Generic automation sequence failed: %v", err)
	}

	fmt.Printf("Transmission Sequence Finalized.\nServer Response:\n%s\n", apiResponse)
}
