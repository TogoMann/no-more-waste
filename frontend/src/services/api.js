import { reactive } from 'vue'

const API_BASE = '/api'

export const authState = reactive({
  token: localStorage.getItem('token') || '',
  user: JSON.parse(localStorage.getItem('user') || 'null')
})

export function setSession(token, user) {
  authState.token = token
  authState.user = user
  localStorage.setItem('token', token)
  localStorage.setItem('user', JSON.stringify(user))
}

export function clearSession() {
  authState.token = ''
  authState.user = null
  localStorage.removeItem('token')
  localStorage.removeItem('user')
}

export function isAuthenticated() {
  return Boolean(authState.token)
}

export function hasRole(...roles) {
  return authState.user && roles.includes(authState.user.role)
}

async function request(method, path, body) {
  const headers = {}
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  if (authState.token) {
    headers['Authorization'] = `Bearer ${authState.token}`
  }
  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined
  })
  if (response.status === 401) {
    clearSession()
  }
  if (!response.ok) {
    let message = 'request failed'
    try {
      const data = await response.json()
      message = data.error || message
    } catch (error) {
      message = response.statusText
    }
    throw new Error(message)
  }
  if (response.status === 204) {
    return null
  }
  return response.json()
}

export const api = {
  get: (path) => request('GET', path),
  post: (path, body) => request('POST', path, body),
  put: (path, body) => request('PUT', path, body),
  patch: (path, body) => request('PATCH', path, body),
  delete: (path) => request('DELETE', path)
}

export async function downloadFile(path, filename) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: `Bearer ${authState.token}` }
  })
  if (!response.ok) {
    throw new Error('download failed')
  }
  const blob = await response.blob()
  const url = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  window.URL.revokeObjectURL(url)
}
