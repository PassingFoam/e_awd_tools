// 前端统一请求封装 + 登录态存储
//
// 管理端鉴权用 cookie 会话（HttpOnly + SameSite=Strict，由服务端 Set-Cookie 下发），
// 因此 fetch 只需 credentials:'include' 即可自动带 cookie，WebSocket 也自动带。
// 401 时清登录态并派发 '0e7-unauth' 事件，App.vue 监听后切回登录页。

const TOKEN_KEY = '0e7_admin_logged_in'

export function setLoggedIn(v: boolean): void {
  localStorage.setItem(TOKEN_KEY, v ? '1' : '0')
}

export function getLoggedIn(): boolean {
  return localStorage.getItem(TOKEN_KEY) === '1'
}

export function clearLoggedIn(): void {
  localStorage.removeItem(TOKEN_KEY)
}

// 统一请求：带 cookie；401 清登录态并通知 App 回登录页。
export async function request(url: string, opts: RequestInit = {}): Promise<Response> {
  const res = await fetch(url, { ...opts, credentials: 'include' })
  if (res.status === 401) {
    clearLoggedIn()
    window.dispatchEvent(new CustomEvent('0e7-unauth'))
    throw new Error('未登录或会话已过期')
  }
  return res
}
