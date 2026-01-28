const API_BASE = '/sync/api'

class ApiService {
  constructor() {
    this.token = localStorage.getItem('chartdb_sync_token')
  }

  setToken(token) {
    this.token = token
    localStorage.setItem('chartdb_sync_token', token)
  }

  clearToken() {
    this.token = null
    localStorage.removeItem('chartdb_sync_token')
    localStorage.removeItem('chartdb_sync_user')
  }

  async request(endpoint, options = {}) {
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers
    }

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`
    }

    const response = await fetch(`${API_BASE}${endpoint}`, {
      ...options,
      headers
    })

    if (response.status === 401) {
      this.clearToken()
      window.location.href = '/sync/login'
      throw new Error('Unauthorized')
    }

    const data = await response.json()

    if (!response.ok) {
      throw new Error(data.error || 'Request failed')
    }

    return data
  }

  // Auth endpoints
  async signup(email, password, name) {
    const data = await this.request('/auth/signup', {
      method: 'POST',
      body: JSON.stringify({ email, password, name })
    })
    this.setToken(data.token)
    localStorage.setItem('chartdb_sync_user', JSON.stringify(data.user))
    return data
  }

  async login(email, password) {
    const data = await this.request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password })
    })
    this.setToken(data.token)
    localStorage.setItem('chartdb_sync_user', JSON.stringify(data.user))
    return data
  }

  async getCurrentUser() {
    return this.request('/auth/me')
  }

  // Diagram endpoints
  async listDiagrams() {
    return this.request('/diagrams')
  }

  async getDiagram(diagramId) {
    return this.request(`/diagrams/${diagramId}`)
  }

  async pushDiagram(diagramData) {
    return this.request('/diagrams/push', {
      method: 'POST',
      body: JSON.stringify(diagramData)
    })
  }

  // Sync diagram without creating new version (for auto-sync)
  async syncDiagram(diagramData) {
    return this.request('/diagrams/sync', {
      method: 'POST',
      body: JSON.stringify(diagramData)
    })
  }

  // Pull all diagrams for login sync
  async pullAllDiagrams() {
    return this.request('/diagrams/pull-all')
  }

  // Create a manual snapshot/version of a diagram
  async createSnapshot(diagramId, description = '') {
    return this.request(`/diagrams/${diagramId}/snapshot`, {
      method: 'POST',
      body: JSON.stringify({ description })
    })
  }

  async pullDiagram(diagramId, version = null) {
    const query = version ? `?version=${version}` : ''
    return this.request(`/diagrams/pull/${diagramId}${query}`)
  }

  async deleteDiagram(diagramId) {
    return this.request(`/diagrams/${diagramId}`, {
      method: 'DELETE'
    })
  }

  async getVersions(diagramId) {
    return this.request(`/diagrams/${diagramId}/versions`)
  }

  // Delete a specific version/snapshot
  async deleteVersion(diagramId, version) {
    return this.request(`/diagrams/${diagramId}/versions/${version}`, {
      method: 'DELETE'
    })
  }
}

export const api = new ApiService()
