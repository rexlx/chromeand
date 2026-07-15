# 1. Fetch the active tabs and get the WebSocket Debug URL
$ChromeInfo = Invoke-RestMethod -Uri "http://127.0.0.1:9222/json"
$TargetPage = $ChromeInfo | Where-Object { $_.url -like "*https://neo.nullferatu.com:8081/app*" } | Select-Object -First 1
$WsUrl = $TargetPage.webSocketDebuggerUrl

# 2. Create the WebSocket Connection
$WS = [System.Net.WebSockets.ClientWebSocket]::new()
$CT = [System.Threading.CancellationToken]::None
$ConnectTask = $WS.ConnectAsync([Uri]$WsUrl, $CT)
$ConnectTask.Wait()

# Helper function to send raw JSON payloads to CDP
function Send-CdpCommand ($Method, $Params) {
    $Payload = @{
        id     = [Guid]::NewGuid().ToString()
        method = $Method
        params = $Params
    } | ConvertTo-Json -Depth 10
    
    $Bytes = [System.Text.Encoding]::UTF8.GetBytes($Payload)
    $Buffer = [ArraySegment[byte]]::new($Bytes)
    $SendTask = $WS.SendAsync($Buffer, [System.Net.WebSockets.WebSocketMessageType]::Text, $true, $CT)
    $SendTask.Wait()
}

# 3. Target the DOM elements and automate the UI
# Click the sidebar search item using the Runtime domain to evaluate JS
Send-CdpCommand "Runtime.evaluate" @{ expression = "document.getElementById('sidebarSearch').click();" }

# Wait a brief moment for the JS framework to render the input field
Start-Sleep -Seconds 1

# Focus and type into the newly rendered input element
Send-CdpCommand "Runtime.evaluate" @{ expression = "
    let input = document.querySelector('#matchBox input, #mainSection input');
    input.value = 'malware indicator';
    input.dispatchEvent(new Event('input', { bubbles: true }));
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', keyCode: 13, bubbles: true }));
" }

# Clean up
$CloseTask = $WS.CloseAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, "", $CT)
$CloseTask.Wait()