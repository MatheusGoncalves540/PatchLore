const TAG = {
  novo: { label: 'NOVO', stroke: '#1d4ed8', fg: '#ffffff', bg: '#1d4ed8', icon: '<path d="M12 5v14M5 12h14"/>' },
  melhoria: { label: 'MELHORIA', stroke: '#0f766e', fg: '#ffffff', bg: '#0f766e', icon: '<path d="M4 17l6-6 4 4 6-6"/><path d="M14 9h6v6"/>' },
  correcao: { label: 'CORREÇÃO', stroke: '#6d28d9', fg: '#3b0764', bg: '#ede9fe', icon: '<path d="M20 6L9 17l-5-5"/>' },
  regra: { label: 'REGRA', stroke: '#b45309', fg: '#7c2d12', bg: '#fde7cf', icon: '<path d="M12 3l9 16H3z"/><path d="M12 10v4M12 17h.01"/>' },
};

const esc = (s) =>
  String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);

export function renderHtml(notes, pr) {
  const cards = notes.itens
    .map((it) => {
      const t = TAG[it.tag];
      return `<div class="card">
  <div class="row"><svg aria-hidden="true" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="${t.stroke}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${t.icon}</svg>
  <span class="tag" style="color:${t.fg};background:${t.bg}">${t.label}</span></div>
  <h2>${esc(it.titulo)}</h2>
  <p>${esc(it.texto)}</p>
</div>`;
    })
    .join('\n');
  return `<!doctype html>
<html lang="pt-BR"><head><meta charset="utf-8"><title>${esc(pr.repo)} #${pr.number}</title>
<style>
body{margin:0;background:#f6f6f3}
#pn{width:960px;box-sizing:border-box;padding:56px;background:#f6f6f3;color:#16181d;font-family:Inter,system-ui,-apple-system,"Segoe UI",sans-serif;display:flex;flex-direction:column;gap:32px}
.head{display:flex;justify-content:space-between;align-items:flex-end;gap:24px}
.eyebrow{font-size:13px;font-weight:600;letter-spacing:.08em;text-transform:uppercase;color:#4b5563}
h1{margin:10px 0;font-size:48px;line-height:1.05;letter-spacing:-.02em}
.sub{margin:0;font-size:17px;color:#374151}
.pill{font-size:13px;font-weight:500;color:#374151;padding:8px 14px;border:1px solid #d4d4cf;border-radius:999px;white-space:nowrap}
.grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}
.card{background:#fff;border:1px solid #e5e5e0;border-radius:16px;padding:28px;display:flex;flex-direction:column;gap:12px}
.row{display:flex;align-items:center;justify-content:space-between}
.tag{font-size:12px;font-weight:600;padding:4px 10px;border-radius:999px}
h2{margin:0;font-size:22px}
p{margin:0;font-size:15px;line-height:1.5;color:#374151}
.foot{display:flex;justify-content:space-between;font-size:13px;color:#4b5563;border-top:1px solid #d4d4cf;padding-top:18px}
</style></head><body>
<div id="pn">
<div class="head"><div>
<span class="eyebrow">${esc(pr.repo.split('/')[1])} · PR #${pr.number}</span>
<h1>O que mudou</h1>
<p class="sub">${esc(notes.titulo)}</p>
</div><span class="pill">${esc(pr.head)} → ${esc(pr.base)}</span></div>
<div class="grid">
${cards}
</div>
<div class="foot"><span>${esc(pr.title)}</span><span>${pr.commits.length} commits · ${pr.files.length} arquivos</span></div>
</div>
</body></html>`;
}

export function renderMarkdown(notes, pr) {
  const items = notes.itens.map((it) => `- **${TAG[it.tag].label} · ${it.titulo}**: ${it.texto}`).join('\n');
  return `### O que mudou: ${notes.titulo}\n\n${items}\n\n<sub>PR #${pr.number} · ${pr.commits.length} commits · ${pr.files.length} arquivos</sub>`;
}
