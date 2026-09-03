// Thin fetch wrapper for the admin API. Auth is a cookie set by the server
// on login, so every call just needs credentials included.

class ApiError extends Error {
  constructor(status, message) {
    super(message || `request failed (${status})`)
    this.status = status
  }
}

async function request(method, path, body) {
  const res = await fetch(path, {
    method,
    credentials: 'same-origin',
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new ApiError(res.status, text)
  }
  if (res.status === 204) return null
  const contentType = res.headers.get('content-type') || ''
  if (contentType.includes('application/json')) return res.json()
  return null
}

export const api = {
  ApiError,

  login: (username, password) => request('POST', '/admin/api/login', { username, password }),
  logout: () => request('POST', '/admin/api/logout'),
  session: () => request('GET', '/admin/api/session'),
  version: () => request('GET', '/admin/api/version'),

  getSettings: () => request('GET', '/admin/api/settings'),
  updateSettings: (publicAddr, autoUpdateEnabled) =>
    request('PUT', '/admin/api/settings', { public_addr: publicAddr, auto_update_enabled: autoUpdateEnabled }),
  changeOwnPassword: (currentPassword, newPassword) =>
    request('PUT', '/admin/api/account/password', { current_password: currentPassword, new_password: newPassword }),

  listAdmins: () => request('GET', '/admin/api/admins'),
  createAdmin: (username, password) => request('POST', '/admin/api/admins', { username, password }),
  deleteAdmin: (id) => request('DELETE', `/admin/api/admins/${id}`),

  listReleases: () => request('GET', '/admin/api/releases'),
  deleteRelease: (id) => request('DELETE', `/admin/api/releases/${id}`),
  async uploadRelease(version, notes, file, onProgress) {
    const form = new FormData()
    form.append('version', version)
    form.append('notes', notes)
    form.append('file', file)
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', '/admin/api/releases')
      xhr.upload.onprogress = (e) => {
        if (onProgress && e.lengthComputable) onProgress(e.loaded / e.total)
      }
      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) resolve(JSON.parse(xhr.responseText))
        else reject(new ApiError(xhr.status, xhr.responseText))
      }
      xhr.onerror = () => reject(new ApiError(0, 'network error'))
      xhr.send(form)
    })
  },

  listUsers: () => request('GET', '/admin/api/users'),
  getUser: (id) => request('GET', `/admin/api/users/${id}`),
  createUser: (name) => request('POST', '/admin/api/users', { name }),
  updateUser: (id, { name, enabled, startsAt, expiresAt }) =>
    request('PUT', `/admin/api/users/${id}`, { name, enabled, starts_at: startsAt, expires_at: expiresAt }),
  deleteUser: (id) => request('DELETE', `/admin/api/users/${id}`),
  setUserNodes: (id, nodeIds) => request('PUT', `/admin/api/users/${id}/nodes`, { node_ids: nodeIds }),

  listNodes: () => request('GET', '/admin/api/nodes'),
  createNode: (name, address, tags) => request('POST', '/admin/api/nodes', { name, address, tags }),
  updateNode: (id, { name, address, tags, enabled }) =>
    request('PATCH', `/admin/api/nodes/${id}`, { name, address, tags, enabled }),
  deleteNode: (id) => request('DELETE', `/admin/api/nodes/${id}`),

  statsOverview: () => request('GET', '/admin/api/stats/overview'),
  statsTimeSeries: (hours = 24) => request('GET', `/admin/api/stats/timeseries?hours=${hours}`),
  statsUser: (id) => request('GET', `/admin/api/stats/users/${id}`),
  statsUserSeries: (id, hours = 24) => request('GET', `/admin/api/stats/users/${id}/series?hours=${hours}`),
  statsNode: (id) => request('GET', `/admin/api/stats/nodes/${id}`),

  listAuditLog: (limit = 200) => request('GET', `/admin/api/audit-log?limit=${limit}`),
}
