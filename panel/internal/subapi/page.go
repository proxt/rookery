package subapi

import (
	"html/template"
	"net/http"
)

// subPageData is what the browser-facing landing page renders — a human
// opening their sub_url in a browser instead of an app shouldn't see raw
// JSON.
type subPageData struct {
	Found     bool
	Name      string
	ExpiresAt string
	NodeCount int
	SubURL    string
}

var subPageTmpl = template.Must(template.New("sub").Parse(`<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
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
  .badge { width: 56px; height: 56px; border-radius: 16px; background: linear-gradient(135deg, var(--accent), #7c5cff);
    display: flex; align-items: center; justify-content: center; margin: 0 auto 16px; font-size: 24px; }
  h1 { font-size: 18px; margin: 0 0 4px; }
  .sub { color: var(--muted); font-size: 13px; margin: 0 0 20px; }
  .pill { display: inline-flex; align-items: center; gap: 6px; border-radius: 999px; padding: 4px 12px;
    font-size: 12px; font-weight: 500; margin-bottom: 20px; }
  .pill.ok { background: rgba(34,197,94,.15); color: var(--ok); }
  .pill.off { background: rgba(224,69,63,.15); color: var(--danger); }
  .pill .dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
  .stat { display: flex; justify-content: space-between; padding: 10px 0; border-top: 1px solid var(--border); font-size: 13px; }
  .stat span:first-child { color: var(--muted); }
  button { width: 100%; margin-top: 20px; cursor: pointer; border: none; border-radius: 12px;
    padding: 13px; font-size: 14px; font-weight: 600; background: var(--accent); color: #fff;
    transition: filter .15s, transform .1s; }
  button:hover { filter: brightness(1.1); }
  button:active { transform: scale(.98); }
  button.copied { background: var(--ok); }
  .hint { color: var(--muted); font-size: 12px; margin-top: 14px; line-height: 1.5; }
</style>
</head>
<body>
<div class="card">
  {{if .Found}}
    <div class="badge">🔗</div>
    <h1>{{.Name}}</h1>
    <p class="sub">Подписка Rookery</p>
    <div class="pill ok"><span class="dot"></span>Активна</div>
    <div class="stat"><span>Серверов доступно</span><span>{{.NodeCount}}</span></div>
    {{if .ExpiresAt}}<div class="stat"><span>Действует до</span><span>{{.ExpiresAt}}</span></div>{{end}}
    <button id="copy-btn" onclick="copyLink()">Скопировать ссылку</button>
    <p class="hint">Вставьте эту ссылку в приложении Rookery, вкладка «Профили» → «Добавить по ссылке».</p>
  {{else}}
    <div class="badge">⚠️</div>
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
    setTimeout(() => { btn.textContent = 'Скопировать ссылку'; btn.classList.remove('copied'); }, 1500);
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
