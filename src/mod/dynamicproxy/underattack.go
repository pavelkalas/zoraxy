package dynamicproxy

/*
	underattack.go

	Interstitial stránka pro režim "Under attack". Když má proxy endpoint
	zapnutý UnderAttack, místo cílového webu se návštěvníkovi zobrazí tahle
	stránka (HTTP 503).
*/

import (
	"net/http"
	"strings"
)

const underAttackPageHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="robots" content="noindex">
    <title>Under Attack | %s</title>
    <style>
        body {
            margin: 0;
            font-family: -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background: linear-gradient(135deg, #1b1c2b 0%, #2b2d42 100%);
            color: #e9ecef;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
        }
        .card {
            max-width: 460px;
            width: 92vw;
            background: rgba(255,255,255,0.04);
            border: 1px solid rgba(255,255,255,0.12);
            border-radius: 16px;
            padding: 2.2em;
            text-align: center;
            box-shadow: 0 20px 60px rgba(0,0,0,0.35);
        }
        .shield {
            font-size: 3em;
            margin-bottom: 0.2em;
        }
        h1 {
            font-size: 1.7em;
            margin: 0.2em 0 0.4em;
            letter-spacing: 0.02em;
        }
        p {
            line-height: 1.6;
            opacity: 0.9;
        }
        .host {
            font-family: ui-monospace, "SF Mono", Menlo, Consolas, monospace;
            background: rgba(255,255,255,0.08);
            padding: 0.15em 0.5em;
            border-radius: 6px;
        }
        .pulse {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            background: #ff6b6b;
            box-shadow: 0 0 0 0 rgba(255,107,107,0.6);
            animation: pulse 1.6s infinite;
            margin-right: 0.4em;
        }
        @keyframes pulse {
            0% { box-shadow: 0 0 0 0 rgba(255,107,107,0.6); }
            70% { box-shadow: 0 0 0 14px rgba(255,107,107,0); }
            100% { box-shadow: 0 0 0 0 rgba(255,107,107,0); }
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="shield">&#128737;&#65039;</div>
        <h1>Under Attack</h1>
        <p><span class="pulse"></span><strong>This site is currently under attack.</strong><br>
        Please be patient — we are working on recovery.</p>
        <p style="opacity:0.6; font-size:0.9em;">Domain: <span class="host">%s</span></p>
    </div>
</body>
</html>`

func serveUnderAttackPage(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	html := strings.ReplaceAll(underAttackPageHTML, "%s", host)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(html))
}
