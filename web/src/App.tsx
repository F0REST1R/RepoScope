import { FormEvent, useEffect, useMemo, useState } from 'react'
import { api, APIError } from './api'
import type { Contributor, Favorite, Issue, RateLimit, Release, Repository, SearchResult } from './types'

const nf = new Intl.NumberFormat('ru-RU')
const compact = new Intl.NumberFormat('ru-RU', { notation: 'compact', maximumFractionDigits: 1 })
const date = (v?: string) => v ? new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium' }).format(new Date(v)) : '—'
type Route = { kind: 'home' } | { kind: 'repo'; owner: string; repo: string }
function currentRoute(): Route { const m = location.pathname.match(/^\/repositories\/([^/]+)\/([^/]+)$/); return m ? { kind: 'repo', owner: decodeURIComponent(m[1]), repo: decodeURIComponent(m[2]) } : { kind: 'home' } }
function navigate(path: string) { history.pushState({}, '', path); dispatchEvent(new PopStateEvent('popstate')) }

export default function App() {
  const [route, setRoute] = useState<Route>(currentRoute)
  const [favorites, setFavorites] = useState<Favorite[]>([])
  const [rate, setRate] = useState<RateLimit>()
  useEffect(() => { const fn = () => setRoute(currentRoute()); addEventListener('popstate', fn); return () => removeEventListener('popstate', fn) }, [])
  const refreshFavorites = () => api.favorites().then(x => setFavorites(x.items)).catch(() => undefined)
  useEffect(() => { refreshFavorites(); api.rate().then(setRate).catch(() => undefined) }, [])
  return <div className="app-shell">
    <header className="topbar"><button className="brand" onClick={() => navigate('/')} aria-label="На главную"><span>RepoScope<small>repository intelligence</small></span></button><nav><a href="#favorites" onClick={(e) => { e.preventDefault(); navigate('/'); setTimeout(() => document.getElementById('favorites')?.scrollIntoView(), 50) }}>Избранное <span className="count">{favorites.length}</span></a><a href="/docs/openapi.yaml">API</a></nav></header>
    <main>{route.kind === 'home' ? <Home favorites={favorites} refreshFavorites={refreshFavorites} rate={rate} /> : <Details {...route} favorites={favorites} refreshFavorites={refreshFavorites} />}</main>
    <footer><span>RepoScope · учебный full-stack проект на Go</span><span>{rate ? `GitHub search: ${rate.search.remaining}/${rate.search.limit}` : 'Лимит GitHub загружается…'}</span></footer>
  </div>
}

function Home({ favorites, refreshFavorites, rate }: { favorites: Favorite[]; refreshFavorites: () => void; rate?: RateLimit }) {
  const [query, setQuery] = useState('web framework')
  const [language, setLanguage] = useState('')
  const [sort, setSort] = useState('stars')
  const [page, setPage] = useState(1)
  const [result, setResult] = useState<SearchResult>()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const search = async (nextPage = 1) => { if (query.trim().length < 2) { setError('Введите хотя бы два символа.'); return } setLoading(true); setError(undefined); try { const p = new URLSearchParams({ q: query.trim(), sort, order: 'desc', page: String(nextPage), per_page: '12' }); if (language) p.set('language', language); setResult(await api.search(p)); setPage(nextPage) } catch (e) { setError(message(e)) } finally { setLoading(false) } }
  useEffect(() => { void search(1) }, []) // initial meaningful result
  const submit = (e: FormEvent) => { e.preventDefault(); void search(1) }
  return <>
    <section className="hero"><div className="eyebrow">OPEN-SOURCE RADAR</div><h1>Найдите репозиторий,<br/><em>достойный вашего времени.</em></h1><p>Поиск, сигналы активности и ключевые данные GitHub — в одном спокойном рабочем пространстве.</p>
      <form className="search-panel" onSubmit={submit}><label className="search-input"><span aria-hidden="true">⌕</span><span className="sr-only">Поисковый запрос</span><input value={query} onChange={e => setQuery(e.target.value)} placeholder="Например: distributed database" /></label><label><span className="sr-only">Язык</span><select value={language} onChange={e => setLanguage(e.target.value)}><option value="">Все языки</option><option>Go</option><option>TypeScript</option><option>Python</option><option>Rust</option><option>Java</option></select></label><label><span className="sr-only">Сортировка</span><select value={sort} onChange={e => setSort(e.target.value)}><option value="stars">По звёздам</option><option value="updated">По обновлению</option><option value="forks">По форкам</option></select></label><button className="primary" disabled={loading}>{loading ? 'Ищем…' : 'Исследовать'}</button></form>
      <div className="hero-meta"><span><i className="pulse"/> Публичные данные GitHub</span>{rate && <span>{rate.core.remaining} запросов API доступно</span>}</div>
    </section>
    <section className="content-section" aria-live="polite"><div className="section-title"><div><span className="eyebrow">РЕЗУЛЬТАТЫ</span><h2>{result ? `${nf.format(result.total_count)} репозиториев` : 'Подборка репозиториев'}</h2></div>{result && <span>Страница {result.page} из {result.total_pages || 1}</span>}</div>
      {error && <State kind="error" title="Не удалось выполнить поиск" text={error} action={() => search(page)} />}
      {loading && <Skeleton />}
      {!loading && !error && result?.items.length === 0 && <State kind="empty" title="Ничего не найдено" text="Попробуйте убрать фильтр языка или изменить формулировку." />}
      {!loading && !error && result && <div className="repo-grid">{result.items.map(r => <RepoCard key={r.id} repo={r} saved={favorites.some(f => f.full_name.toLowerCase() === r.full_name.toLowerCase())} refresh={refreshFavorites} />)}</div>}
      {result && result.total_pages > 1 && <div className="pagination"><button disabled={page <= 1 || loading} onClick={() => search(page - 1)}>← Назад</button><span>{page}</span><button disabled={page >= result.total_pages || loading} onClick={() => search(page + 1)}>Вперёд →</button></div>}
    </section>
    <Favorites items={favorites} refresh={refreshFavorites} />
  </>
}

function RepoCard({ repo, saved, refresh }: { repo: Repository; saved: boolean; refresh: () => void }) {
  const [busy, setBusy] = useState(false)
  const toggle = async () => { setBusy(true); try { saved ? await api.removeFavorite(repo.owner.login, repo.name) : await api.addFavorite(repo.owner.login, repo.name); refresh() } catch (e) { alert(message(e)) } finally { setBusy(false) } }
  return <article className="repo-card"><div className="card-top"><span className="owner-label">{repo.owner.login}</span><button className={`save ${saved ? 'saved' : ''}`} disabled={busy} onClick={toggle} aria-label={saved ? 'Удалить из избранного' : 'Добавить в избранное'}>{saved ? '★' : '☆'}</button></div><button className="title-link" onClick={() => navigate(`/repositories/${encodeURIComponent(repo.owner.login)}/${encodeURIComponent(repo.name)}`)}><span>{repo.owner.login} /</span> {repo.name}</button><p>{repo.description || 'Описание не добавлено автором.'}</p><div className="topics">{repo.topics?.slice(0, 3).map(t => <span key={t}>{t}</span>)}</div><div className="card-stats"><span>★ {compact.format(repo.stargazers_count)}</span><span>⑂ {compact.format(repo.forks_count)}</span><span className="language"><i/>{repo.language || '—'}</span></div></article>
}

function Details({ owner, repo, favorites, refreshFavorites }: { owner: string; repo: string; favorites: Favorite[]; refreshFavorites: () => void }) {
  const [data, setData] = useState<Repository>(); const [langs, setLangs] = useState<Record<string, number>>({}); const [contributors, setContributors] = useState<Contributor[]>([]); const [issues, setIssues] = useState<Issue[]>([]); const [releases, setReleases] = useState<Release[]>([]); const [error, setError] = useState<string>(); const [loading, setLoading] = useState(true)
  useEffect(() => { setLoading(true); Promise.all([api.repository(owner, repo), api.languages(owner, repo), api.contributors(owner, repo), api.issues(owner, repo), api.releases(owner, repo)]).then(([r,l,c,i,rel]) => { setData(r);setLangs(l.languages);setContributors(c.items);setIssues(i.items);setReleases(rel.items) }).catch(e => setError(message(e))).finally(() => setLoading(false)) }, [owner, repo])
  if (loading) return <section className="content-section detail-wrap"><Skeleton /></section>
  if (error || !data) return <section className="content-section detail-wrap"><State kind="error" title="Репозиторий недоступен" text={error || 'Данные не получены.'} action={() => location.reload()} /></section>
  const saved = favorites.some(f => f.full_name.toLowerCase() === data.full_name.toLowerCase())
  const total = Object.values(langs).reduce((a,b) => a+b,0)
  const toggle = async () => { try { saved ? await api.removeFavorite(owner,repo) : await api.addFavorite(owner,repo);refreshFavorites() } catch(e){alert(message(e))} }
  return <section className="detail-wrap"><button className="back" onClick={() => navigate('/')}>← К поиску</button><div className="detail-hero"><div><div className="eyebrow">REPOSITORY PROFILE</div><h1>{data.owner.login} <span>/</span> {data.name}</h1><p>{data.description || 'Описание не добавлено.'}</p><div className="detail-actions"><a className="primary" href={data.html_url} target="_blank" rel="noreferrer">Открыть на GitHub ↗</a><button className="secondary" onClick={toggle}>{saved ? '★ В избранном' : '☆ В избранное'}</button></div></div><div className="health"><div className="score" style={{'--score': `${data.health_score ?? 0}%`} as React.CSSProperties}><strong>{data.health_score}</strong><small>/ 100</small></div><span>Repo Health</span><em>{data.health_summary}</em></div></div>
    <div className="metric-grid"><Metric label="Звёзды" value={nf.format(data.stargazers_count)} /><Metric label="Форки" value={nf.format(data.forks_count)} /><Metric label="Открытые задачи" value={nf.format(data.open_issues_count)} /><Metric label="Обновлён" value={date(data.pushed_at)} /></div>
    <div className="detail-grid"><div className="detail-main"><Panel title="Языки"><div className="language-bar">{Object.entries(langs).map(([k,v],i) => <i key={k} style={{width:`${total ? v/total*100 : 0}%`,background:`hsl(${210+i*54} 75% 58%)`}} />)}</div><div className="language-legend">{Object.entries(langs).slice(0,6).map(([k,v]) => <span key={k}><i/>{k} <b>{total ? Math.round(v/total*100) : 0}%</b></span>)}</div></Panel><Panel title="Последние issues" count={issues.length}>{issues.length ? <div className="list">{issues.map(i => <a key={i.number} href={i.html_url} target="_blank" rel="noreferrer"><span className={`state ${i.state}`}>{i.state === 'open' ? '○' : '●'}</span><span><strong>{i.title}</strong><small>#{i.number} · {i.user.login} · {date(i.updated_at)}</small></span></a>)}</div> : <EmptyLine text="Открытые issues не найдены" />}</Panel></div>
      <aside><Panel title="О проекте"><dl><dt>Лицензия</dt><dd>{data.license?.spdx_id || 'Не указана'}</dd><dt>Основной язык</dt><dd>{data.language || '—'}</dd><dt>Ветка</dt><dd>{data.default_branch}</dd><dt>Статус</dt><dd>{data.archived ? 'Архив' : 'Активен'}</dd></dl></Panel><Panel title="Участники" count={contributors.length}><div className="people">{contributors.map(c => <a key={c.login} href={c.html_url} title={`${c.login}: ${c.contributions}`} target="_blank" rel="noreferrer"><img src={c.avatar_url} alt={c.login} /></a>)}</div></Panel><Panel title="Релизы" count={releases.length}>{releases.length ? <div className="releases">{releases.map(r => <a key={r.id} href={r.html_url} target="_blank" rel="noreferrer"><strong>{r.name || r.tag_name}</strong><small>{date(r.published_at)}</small></a>)}</div> : <EmptyLine text="Релизов пока нет" />}</Panel></aside></div>
  </section>
}

function Favorites({items,refresh}:{items:Favorite[];refresh:()=>void}) { return <section className="favorites" id="favorites"><div className="section-title"><div><span className="eyebrow">ВАША КОЛЛЕКЦИЯ</span><h2>Избранные репозитории</h2></div></div>{items.length ? <div className="favorite-list">{items.map(f => <article key={f.id}><button onClick={() => navigate(`/repositories/${encodeURIComponent(f.owner)}/${encodeURIComponent(f.repo)}`)}><strong>{f.full_name}</strong><span>{f.description || 'Без описания'}</span></button><span>★ {compact.format(f.stars)}</span><button className="remove" onClick={() => api.removeFavorite(f.owner,f.repo).then(refresh)} aria-label={`Удалить ${f.full_name}`}>×</button></article>)}</div> : <State kind="empty" title="Коллекция пока пуста" text="Отмечайте репозитории звездой — они сохранятся в PostgreSQL." />}</section> }
function Panel({title,count,children}:{title:string;count?:number;children:React.ReactNode}) { return <section className="panel"><header><h2>{title}</h2>{count !== undefined && <span>{count}</span>}</header>{children}</section> }
function Metric({label,value}:{label:string;value:string}) { return <div className="metric"><span>{label}</span><strong>{value}</strong></div> }
function EmptyLine({text}:{text:string}) { return <p className="empty-line">{text}</p> }
function State({kind,title,text,action}:{kind:'error'|'empty';title:string;text:string;action?:()=>void}) { return <div className={`state-box ${kind}`} role={kind==='error'?'alert':undefined}><span>{kind==='error'?'!':'◇'}</span><h3>{title}</h3><p>{text}</p>{action&&<button onClick={action}>Повторить</button>}</div> }
function Skeleton() { return <div className="repo-grid skeleton" aria-label="Загрузка"><i/><i/><i/><i/><i/><i/></div> }
function message(e: unknown) { return e instanceof APIError ? `${e.message}${e.requestId ? ` (request ID: ${e.requestId})` : ''}` : e instanceof Error ? e.message : 'Неизвестная ошибка' }
