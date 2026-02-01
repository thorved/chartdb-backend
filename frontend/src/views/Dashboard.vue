<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="flex justify-between items-center mb-8">
      <h1 class="text-2xl font-bold text-gray-900">My Diagrams</h1>
      <div class="flex gap-3">
        <button @click="refreshLocal" class="btn btn-secondary" :disabled="refreshing">
          {{ refreshing ? 'Refreshing...' : 'Refresh' }}
        </button>
      </div>
    </div>

    <!-- Auto-Sync Control Panel -->
    <div class="card mb-6 bg-gradient-to-r from-indigo-50 to-purple-50 border-indigo-200">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="flex items-center gap-3">
          <svg class="w-6 h-6 text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <div>
            <p class="font-medium text-gray-800">Auto-Sync</p>
            <p class="text-sm text-gray-600">Automatically sync diagrams to the server</p>
          </div>
        </div>
        
        <div class="flex items-center gap-4">
          <!-- Sync Status Indicator -->
          <div class="flex items-center gap-2">
            <span :class="['w-3 h-3 rounded-full', syncStatusClass]"></span>
            <span class="text-sm text-gray-600">{{ syncStatusText }}</span>
          </div>

          <!-- Interval Selector -->
          <select v-model="autoSyncInterval" @change="updateAutoSyncInterval" 
                  class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm bg-white focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500">
            <option :value="30000">Every 30 seconds</option>
            <option :value="60000">Every 1 minute</option>
            <option :value="300000">Every 5 minutes</option>
            <option :value="600000">Every 10 minutes</option>
          </select>

          <!-- Auto-Sync Toggle -->
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="autoSyncEnabled" @change="toggleAutoSync" class="sr-only peer">
            <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600"></div>
            <span class="ms-2 text-sm font-medium text-gray-700">{{ autoSyncEnabled ? 'On' : 'Off' }}</span>
          </label>
        </div>
      </div>

      <!-- Last Sync Info -->
      <div v-if="lastSyncTime" class="mt-3 pt-3 border-t border-indigo-200">
        <p class="text-xs text-gray-500">
          Last synced: {{ formatDate(lastSyncTime) }}
          <span v-if="nextSyncTime && autoSyncEnabled" class="ml-2">
            • Next sync in: {{ nextSyncCountdown }}
          </span>
        </p>
      </div>
    </div>

    <!-- Database Status -->
    <div v-if="!dbAvailable" class="card mb-6 border-yellow-300 bg-yellow-50">
      <div class="flex items-center gap-3">
        <svg class="w-6 h-6 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <p class="font-medium text-yellow-800">ChartDB not detected</p>
          <p class="text-sm text-yellow-700">Open ChartDB in this browser first to create diagrams, then return here to sync them.</p>
          <p v-if="availableDbs.length > 0" class="text-xs text-yellow-600 mt-1">
            Available databases: {{ availableDbs.map(db => db.name).join(', ') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="text-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
      <p class="mt-4 text-gray-600">Loading diagrams...</p>
    </div>

    <template v-else>
      <!-- Local Diagrams Section -->
      <div class="mb-8">
        <h2 class="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <svg class="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          Browser Diagrams (IndexedDB)
        </h2>
        
        <div v-if="localDiagrams.length === 0" class="card text-center py-8 bg-gray-50">
          <p class="text-gray-500">No diagrams found in browser. Create one in ChartDB first.</p>
        </div>

        <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div v-for="diagram in localDiagrams" :key="diagram.id" 
               class="card hover:shadow-md transition-shadow border-l-4 border-l-blue-500">
            <div class="flex items-start justify-between">
              <div class="flex-1 min-w-0">
                <h3 class="font-semibold text-gray-900 truncate">{{ diagram.name }}</h3>
                <p class="text-xs text-gray-500 mt-1 font-mono">{{ diagram.id }}</p>
              </div>
              <span v-if="getSyncedDiagram(diagram.id)" 
                    class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                Synced
              </span>
              <span v-else class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-gray-100 text-gray-600">
                Local only
              </span>
            </div>

            <div class="mt-3 text-sm text-gray-500">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                {{ diagram.databaseType || 'Unknown' }}
              </span>
            </div>

            <div class="mt-4 flex gap-2">
              <button @click="pushDiagram(diagram)" 
                      :disabled="pushing === diagram.id"
                      class="btn btn-primary text-sm flex-1">
                <span v-if="pushing === diagram.id">Pushing...</span>
                <span v-else>{{ getSyncedDiagram(diagram.id) ? 'Push Update' : 'Push to Server' }}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Synced Diagrams Section -->
      <div>
        <h2 class="text-lg font-semibold text-gray-800 mb-4 flex items-center gap-2">
          <svg class="w-5 h-5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
          </svg>
          Server Diagrams (Synced)
        </h2>

        <div v-if="syncedDiagrams.length === 0" class="card text-center py-8 bg-gray-50">
          <p class="text-gray-500">No diagrams synced to server yet. Push a diagram to get started.</p>
        </div>

        <div v-else class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          <div v-for="diagram in syncedDiagrams" :key="diagram.id" 
               class="card hover:shadow-md transition-shadow border-l-4 border-l-green-500">
            <div class="flex items-start justify-between">
              <div class="flex-1 min-w-0">
                <h3 class="font-semibold text-gray-900 truncate">{{ diagram.name }}</h3>
                <p class="text-xs text-gray-500 mt-1 font-mono">{{ diagram.diagram_id }}</p>
              </div>
              <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
                v{{ diagram.version }}
              </span>
            </div>

            <div class="mt-3 text-sm text-gray-500">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                {{ diagram.database_type || 'Unknown' }}
              </span>
              <span class="ml-2">{{ diagram.table_count }} tables</span>
            </div>

            <div class="mt-2 text-xs text-gray-400">
              Updated: {{ formatDate(diagram.updated_at) }}
            </div>

            <div class="mt-4 flex gap-2 flex-wrap">
              <button @click="pullDiagram(diagram)" 
                      :disabled="pulling === diagram.diagram_id"
                      class="btn btn-success text-sm flex-1">
                {{ pulling === diagram.diagram_id ? 'Pulling...' : 'Pull to Browser' }}
              </button>
              <button @click="createSnapshot(diagram)" 
                      :disabled="creatingSnapshot === diagram.diagram_id"
                      class="btn btn-primary text-sm" title="Create Version Snapshot">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
                </svg>
              </button>
              <button @click="viewVersions(diagram)" class="btn btn-secondary text-sm">
                History
              </button>
              <button @click="confirmDelete(diagram)" class="btn btn-danger text-sm">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Delete Confirmation Modal -->
    <div v-if="deleteTarget" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
        <h2 class="text-xl font-bold text-gray-900 mb-4">Delete from Server</h2>
        <p class="text-gray-600 mb-6">
          Are you sure you want to delete "<strong>{{ deleteTarget.name }}</strong>" from the server? 
          This will not affect your local browser data.
        </p>
        <div class="flex gap-3 justify-end">
          <button @click="deleteTarget = null" class="btn btn-secondary">Cancel</button>
          <button @click="handleDelete" :disabled="deleting" class="btn btn-danger">
            {{ deleting ? 'Deleting...' : 'Delete from Server' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Versions Modal -->
    <div v-if="versionsTarget" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-xl shadow-xl max-w-2xl w-full mx-4 p-6 max-h-[80vh] overflow-auto">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-xl font-bold text-gray-900">Version History: {{ versionsTarget.name }}</h2>
          <button @click="versionsTarget = null" class="text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-if="versionsLoading" class="text-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto"></div>
        </div>

        <div v-else class="space-y-3">
          <div v-for="version in versions" :key="version.id" 
               class="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
            <div>
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900">Version {{ version.version }}</span>
                <span v-if="version.version === versionsTarget.version" 
                      class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                  Latest
                </span>
              </div>
              <p class="text-sm text-gray-500 mt-1">{{ version.description || 'No description' }}</p>
              <p class="text-xs text-gray-400 mt-1">{{ formatDate(version.created_at) }}</p>
            </div>
            <div class="flex gap-2">
              <button @click="pullVersion(versionsTarget.diagram_id, version.version)" 
                      :disabled="pulling"
                      class="btn btn-secondary text-sm">
                {{ pulling === `${versionsTarget.diagram_id}-${version.version}` ? 'Pulling...' : 'Pull This' }}
              </button>
              <button v-if="version.version !== versionsTarget.version && versions.length > 1"
                      @click="deleteVersion(versionsTarget.diagram_id, version.version)"
                      :disabled="deletingVersion === version.version"
                      class="btn btn-danger text-sm" title="Delete this snapshot">
                <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Success/Error Toast -->
    <div v-if="toast" 
         :class="['fixed bottom-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all', 
                  toast.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white']">
      {{ toast.message }}
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { api } from '../api'
import { chartDB } from '../chartdb-client'
import { diagramFromJSON } from '../utils/diagram-export-import'

export default {
  name: 'Dashboard',
  setup() {
    const loading = ref(true)
    const refreshing = ref(false)
    const dbAvailable = ref(true)
    const availableDbs = ref([])
    const localDiagrams = ref([])
    const syncedDiagrams = ref([])
    
    const pushing = ref(null)
    const pulling = ref(null)
    
    const deleteTarget = ref(null)
    const deleting = ref(false)
    
    const versionsTarget = ref(null)
    const versions = ref([])
    const versionsLoading = ref(false)
    const deletingVersion = ref(null)
    
    const toast = ref(null)

    // Snapshot state
    const creatingSnapshot = ref(null)

    // Auto-sync state
    const autoSyncEnabled = ref(false)
    const autoSyncInterval = ref(60000)
    const lastSyncTime = ref(null)
    const nextSyncTime = ref(null)
    const isSyncing = ref(false)
    const syncStatus = ref('idle') // idle, syncing, success, error
    let autoSyncTimer = null
    let countdownTimer = null

    // Computed properties for auto-sync
    const syncStatusClass = computed(() => {
      switch (syncStatus.value) {
        case 'syncing': return 'bg-yellow-400 animate-pulse'
        case 'success': return 'bg-green-500'
        case 'error': return 'bg-red-500'
        default: return 'bg-gray-400'
      }
    })

    const syncStatusText = computed(() => {
      switch (syncStatus.value) {
        case 'syncing': return 'Syncing...'
        case 'success': return 'Synced'
        case 'error': return 'Error'
        default: return 'Idle'
      }
    })

    const nextSyncCountdown = computed(() => {
      if (!nextSyncTime.value) return ''
      const diff = Math.max(0, nextSyncTime.value - Date.now())
      const seconds = Math.floor(diff / 1000)
      if (seconds >= 60) {
        return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
      }
      return `${seconds}s`
    })

    // Load auto-sync preferences from localStorage
    const loadAutoSyncPreferences = () => {
      autoSyncEnabled.value = localStorage.getItem('chartdb_sync_auto') === 'true'
      autoSyncInterval.value = parseInt(localStorage.getItem('chartdb_sync_interval')) || 60000
    }

    // Save auto-sync preferences to localStorage
    const saveAutoSyncPreferences = () => {
      localStorage.setItem('chartdb_sync_auto', autoSyncEnabled.value)
      localStorage.setItem('chartdb_sync_interval', autoSyncInterval.value)
    }

    // Perform auto-sync for all local diagrams (uses sync endpoint - no version increment)
    const performAutoSync = async () => {
      if (isSyncing.value || !dbAvailable.value) return
      
      isSyncing.value = true
      syncStatus.value = 'syncing'

      try {
        // Sync all local diagrams to server (without creating new versions)
        for (const diagram of localDiagrams.value) {
          const diagramJSON = await chartDB.getDiagramJSON(diagram.id)
          // Use syncDiagram instead of pushDiagram for auto-sync
          await api.syncDiagram(diagramJSON)
        }

        // Reload synced diagrams
        await loadSyncedDiagrams()
        
        syncStatus.value = 'success'
        lastSyncTime.value = new Date().toISOString()
        
        // Schedule next sync time
        if (autoSyncEnabled.value) {
          nextSyncTime.value = Date.now() + autoSyncInterval.value
        }
      } catch (err) {
        console.error('Auto-sync failed:', err)
        syncStatus.value = 'error'
        showToast('Auto-sync failed: ' + err.message, 'error')
      } finally {
        isSyncing.value = false
      }
    }

    // Start auto-sync timer
    const startAutoSync = () => {
      stopAutoSync()
      if (autoSyncEnabled.value) {
        // Initial sync
        performAutoSync()
        
        // Set up interval
        autoSyncTimer = setInterval(performAutoSync, autoSyncInterval.value)
        
        // Set up countdown timer
        countdownTimer = setInterval(() => {
          // Force reactivity update for countdown
          if (nextSyncTime.value) {
            nextSyncTime.value = nextSyncTime.value
          }
        }, 1000)
        
        console.log(`Auto-sync started with interval: ${autoSyncInterval.value}ms`)
      }
    }

    // Stop auto-sync timer
    const stopAutoSync = () => {
      if (autoSyncTimer) {
        clearInterval(autoSyncTimer)
        autoSyncTimer = null
      }
      if (countdownTimer) {
        clearInterval(countdownTimer)
        countdownTimer = null
      }
      nextSyncTime.value = null
      console.log('Auto-sync stopped')
    }

    // Toggle auto-sync on/off
    const toggleAutoSync = () => {
      saveAutoSyncPreferences()
      if (autoSyncEnabled.value) {
        startAutoSync()
        showToast('Auto-sync enabled!')
      } else {
        stopAutoSync()
        syncStatus.value = 'idle'
        showToast('Auto-sync disabled')
      }
    }

    // Update auto-sync interval
    const updateAutoSyncInterval = () => {
      saveAutoSyncPreferences()
      if (autoSyncEnabled.value) {
        startAutoSync() // Restart with new interval
        showToast(`Sync interval updated to ${autoSyncInterval.value / 1000} seconds`)
      }
    }

    const showToast = (message, type = 'success') => {
      toast.value = { message, type }
      setTimeout(() => { toast.value = null }, 3000)
    }

    const loadLocalDiagrams = async () => {
      try {
        // Get list of all databases for debugging
        availableDbs.value = await chartDB.listDatabases()
        console.log('Available databases:', availableDbs.value)
        
        const exists = await chartDB.checkDatabase()
        dbAvailable.value = exists
        
        if (exists) {
          localDiagrams.value = await chartDB.getDiagrams()
          console.log('Loaded diagrams:', localDiagrams.value)
        }
      } catch (err) {
        console.error('Failed to load local diagrams:', err)
        dbAvailable.value = false
      }
    }

    const loadSyncedDiagrams = async () => {
      try {
        syncedDiagrams.value = await api.listDiagrams()
      } catch (err) {
        console.error('Failed to load synced diagrams:', err)
      }
    }

    const loadAll = async () => {
      loading.value = true
      await Promise.all([loadLocalDiagrams(), loadSyncedDiagrams()])
      loading.value = false
    }

    const refreshLocal = async () => {
      refreshing.value = true
      await loadAll()
      refreshing.value = false
      showToast('Refreshed!')
    }

    const getSyncedDiagram = (localId) => {
      return syncedDiagrams.value.find(d => d.diagram_id === localId)
    }

    const formatDate = (dateStr) => {
      return new Date(dateStr).toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    }

    const pushDiagram = async (diagram) => {
      try {
        pushing.value = diagram.id
        
        // Get diagram data in JSON format from IndexedDB
        const diagramJSON = await chartDB.getDiagramJSON(diagram.id)
        
        // Push to server
        await api.pushDiagram(diagramJSON)
        
        // Reload synced diagrams
        await loadSyncedDiagrams()
        
        showToast(`"${diagram.name}" pushed to server!`)
      } catch (err) {
        showToast('Failed to push: ' + err.message, 'error')
      } finally {
        pushing.value = null
      }
    }

    const pullDiagram = async (diagram) => {
      try {
        pulling.value = diagram.diagram_id
        
        // Get diagram data from server
        const diagramData = await api.pullDiagram(diagram.diagram_id)
        
        // Use ChartDB-compatible import function to clone with new IDs
        // This ensures relationships, areas, and all entities are properly remapped
        const diagramJSON = JSON.stringify(diagramData)
        const clonedDiagram = diagramFromJSON(diagramJSON)
        
        // Make sure database is open (will create if deleted)
        await chartDB.reopen()
        
        // Save to IndexedDB using the cloned diagram
        await chartDB.saveDiagramJSON(clonedDiagram)
        
        // Reload local diagrams
        dbAvailable.value = true
        await loadLocalDiagrams()
        
        showToast(`"${diagram.name}" pulled to browser!`)
      } catch (err) {
        console.error('Pull error:', err)
        showToast('Failed to pull: ' + err.message, 'error')
      } finally {
        pulling.value = null
      }
    }

    const pullVersion = async (diagramId, version) => {
      try {
        pulling.value = `${diagramId}-${version}`
        
        // Get specific version from server
        const diagramData = await api.pullDiagram(diagramId, version)
        
        // Use ChartDB-compatible import function to clone with new IDs
        // This ensures relationships, areas, and all entities are properly remapped
        const diagramJSON = JSON.stringify(diagramData)
        const clonedDiagram = diagramFromJSON(diagramJSON)
        
        // Make sure database is open (will create if deleted)
        await chartDB.reopen()
        
        // Save to IndexedDB using the cloned diagram
        await chartDB.saveDiagramJSON(clonedDiagram)
        
        // Reload local diagrams
        dbAvailable.value = true
        await loadLocalDiagrams()
        
        showToast(`Version ${version} pulled to browser!`)
        versionsTarget.value = null
      } catch (err) {
        console.error('Pull version error:', err)
        showToast('Failed to pull version: ' + err.message, 'error')
      } finally {
        pulling.value = null
      }
    }

    const viewVersions = async (diagram) => {
      versionsTarget.value = diagram
      versionsLoading.value = true
      
      try {
        versions.value = await api.getVersions(diagram.diagram_id)
      } catch (err) {
        showToast('Failed to load versions: ' + err.message, 'error')
      } finally {
        versionsLoading.value = false
      }
    }

    // Delete a specific version/snapshot
    const deleteVersion = async (diagramId, version) => {
      try {
        deletingVersion.value = version
        await api.deleteVersion(diagramId, version)
        // Reload versions
        versions.value = await api.getVersions(diagramId)
        showToast(`Version ${version} deleted!`)
      } catch (err) {
        showToast('Failed to delete version: ' + err.message, 'error')
      } finally {
        deletingVersion.value = null
      }
    }

    const confirmDelete = (diagram) => {
      deleteTarget.value = diagram
    }

    const handleDelete = async () => {
      try {
        deleting.value = true
        const diagramId = deleteTarget.value.diagram_id
        
        // Delete from server
        await api.deleteDiagram(diagramId)
        
        // Also delete from local IndexedDB
        try {
          await chartDB.deleteDiagram(diagramId)
          console.log('Deleted from local IndexedDB:', diagramId)
        } catch (localErr) {
          console.warn('Could not delete from local IndexedDB:', localErr)
        }
        
        deleteTarget.value = null
        await loadAll() // Reload both local and server diagrams
        showToast('Deleted from server and browser!')
      } catch (err) {
        showToast('Failed to delete: ' + err.message, 'error')
      } finally {
        deleting.value = false
      }
    }

    // Create a manual snapshot/version of a diagram
    const createSnapshot = async (diagram) => {
      try {
        creatingSnapshot.value = diagram.diagram_id
        await api.createSnapshot(diagram.diagram_id, 'Manual snapshot')
        await loadSyncedDiagrams()
        showToast(`Snapshot created for "${diagram.name}"!`)
      } catch (err) {
        showToast('Failed to create snapshot: ' + err.message, 'error')
      } finally {
        creatingSnapshot.value = null
      }
    }

    onMounted(async () => {
      loadAutoSyncPreferences()
      
      // Check if ChartDB database exists with proper schema
      console.log('[Dashboard] Checking ChartDB database...')
      try {
        const dbReady = await chartDB.checkDatabase()
        if (dbReady) {
          console.log('[Dashboard] ChartDB database is ready')
          dbAvailable.value = true
        } else {
          console.log('[Dashboard] ChartDB database not found or empty')
          dbAvailable.value = false
        }
      } catch (err) {
        console.log('[Dashboard] ChartDB not available:', err.message)
        dbAvailable.value = false
      }
      
      await loadAll()
      
      // Start auto-sync if it was enabled
      if (autoSyncEnabled.value && dbAvailable.value) {
        startAutoSync()
      }
    })

    onUnmounted(() => {
      stopAutoSync()
    })

    return {
      loading,
      refreshing,
      dbAvailable,
      availableDbs,
      localDiagrams,
      syncedDiagrams,
      pushing,
      pulling,
      deleteTarget,
      deleting,
      versionsTarget,
      versions,
      versionsLoading,
      toast,
      // Snapshot
      creatingSnapshot,
      createSnapshot,
      // Delete version
      deletingVersion,
      deleteVersion,
      // Auto-sync
      autoSyncEnabled,
      autoSyncInterval,
      lastSyncTime,
      nextSyncTime,
      syncStatusClass,
      syncStatusText,
      nextSyncCountdown,
      toggleAutoSync,
      updateAutoSyncInterval,
      // Methods
      refreshLocal,
      getSyncedDiagram,
      formatDate,
      pushDiagram,
      pullDiagram,
      pullVersion,
      viewVersions,
      confirmDelete,
      handleDelete
    }
  }
}
</script>
