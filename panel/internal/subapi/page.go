package subapi

import (
	_ "embed"
	"html/template"
	"net/http"
)

//go:embed assets/logo.png
var logoPNG []byte

func handleLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(logoPNG)
}

// subPageData is what the browser-facing landing page renders — a human
// opening their sub_url in a browser instead of an app shouldn't see raw
// JSON.
type subPageData struct {
	Found     bool
	Name      string
	ExpiresAt string
	NodeCount int
	// DeepLink opens directly in the app, once it registers the rookery://
	// scheme. template.URL (not string) so html/template doesn't sanitize
	// the custom scheme down to "#ZgotmplZ" — safe here since we build the
	// whole value ourselves from panel data, not from unsanitized input.
	DeepLink template.URL
}

var subPageTmpl = template.Must(template.New("sub").Parse(`<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<link rel="icon" type="image/png" href="/sub-assets/logo.png" />
<title>Rookery — {{if .Found}}{{.Name}}{{else}}Ссылка не найдена{{end}}</title>
<style>
  :root {
    --bg: #0b0d11; --surface: #171a21; --surface-2: #1e222b; --border: #2a2f3a;
    --text: #e5e7eb; --muted: #8b93a1; --accent: #4f7cff; --danger: #e0453f; --ok: #22c55e;
  }
  * { box-sizing: border-box; }
  html, body { margin: 0; min-height: 100%; background: var(--bg); color: var(--text);
    font-family: -apple-system, "Segoe UI", Roboto, sans-serif; }
  body { display: flex; align-items: center; justify-content: center; padding: 24px; }
  .card { width: 100%; max-width: 420px; background: var(--surface); border: 1px solid var(--border);
    border-radius: 20px; padding: 28px; text-align: center;
    animation: rise .4s cubic-bezier(.2,.8,.2,1); }
  @keyframes rise { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }
  .badge { width: 64px; height: 64px; border-radius: 18px; margin: 0 auto 16px; overflow: hidden; }
  .badge img { width: 100%; height: 100%; object-fit: cover; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .sub { color: var(--muted); font-size: 13px; margin: 0 0 20px; }
  .pill { display: inline-flex; align-items: center; gap: 6px; border-radius: 999px; padding: 4px 12px;
    font-size: 12px; font-weight: 500; margin-bottom: 20px; }
  .pill.ok { background: rgba(34,197,94,.15); color: var(--ok); }
  .pill.off { background: rgba(224,69,63,.15); color: var(--danger); }
  .pill .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
  .stat { display: flex; justify-content: space-between; padding: 10px 0; border-top: 1px solid var(--border); font-size: 13px; }
  .stat span:first-child { color: var(--muted); }
  a.primary, button { display: block; width: 100%; margin-top: 20px; box-sizing: border-box; cursor: pointer; border: none;
    border-radius: 12px; padding: 13px; font-size: 14px; font-weight: 600; background: var(--accent); color: #fff;
    text-decoration: none; transition: filter .15s, transform .1s; }
  a.primary:hover, button:hover { filter: brightness(1.1); }
  a.primary:active, button:active { transform: scale(.98); }
  .secondary { margin-top: 10px; background: transparent; color: var(--muted); font-weight: 500; font-size: 13px; padding: 8px; }
  .secondary.copied { color: var(--ok); }
  .hint { color: var(--muted); font-size: 12px; margin-top: 14px; line-height: 1.5; }
</style>
</head>
<body>
<div class="card">
  {{if .Found}}
    <div class="badge"><img src="/sub-assets/logo.png" alt="Rookery" /></div>
    <h1>{{.Name}}</h1>
    <p class="sub">Подписка Rookery</p>
    <div class="pill ok"><span class="dot"></span>Активна</div>
    <div class="stat"><span>Серверов доступно</span><span>{{.NodeCount}}</span></div>
    {{if .ExpiresAt}}<div class="stat"><span>Действует до</span><span>{{.ExpiresAt}}</span></div>{{end}}
    <a class="primary" href="{{.DeepLink}}">Установить в приложение</a>
    <button class="secondary" id="copy-btn" onclick="copyLink()">Скопировать ссылку вместо этого</button>
    <p class="hint">Если приложение Rookery установлено, ссылка выше подключит подписку автоматически. Иначе — скопируйте ссылку и вставьте её в приложении вручную.</p>
  {{else}}
    <div class="badge"><img src="/sub-assets/logo.png" alt="Rookery" /></div>
    <h1>Ссылка недействительна</h1>
    <p class="sub">Подписка отключена, истекла или не существует. Обратитесь к администратору.</p>
  {{end}}
</div>
<script>
function copyLink() {
  navigator.clipboard.writeText(window.location.href).then(() => {
    const btn = document.getElementById('copy-btn');
    btn.textContent = 'Скопировано';
    btn.classList.add('copied');
    setTimeout(() => { btn.textContent = 'Скопировать ссылку вместо этого'; btn.classList.remove('copied'); }, 1500);
  });
}
</script>
</body>
</html>
`))

func renderSubPage(w http.ResponseWriter, data subPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !data.Found {
		w.WriteHeader(http.StatusNotFound)
	}
	subPageTmpl.Execute(w, data)
}
