<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <div class="mb-6">
      <router-link to="/dashboard" class="text-primary-600 hover:text-primary-700 flex items-center gap-1">
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        Back to Dashboard
      </router-link>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="text-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
    </div>

    <template v-else-if="diagram">
      <!-- Diagram Info -->
      <div class="card mb-8">
        <div class="flex items-start justify-between">
          <div>
            <h1 class="text-2xl font-bold text-gray-900">{{ diagram.name }}</h1>
            <p class="mt-2 text-gray-500">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                {{ diagram.database_type || 'Unknown' }}
              </span>
              <span class="ml-2">Current version: v{{ diagram.version }}</span>
            </p>
          </div>
          <div class="flex gap-2">
            <button @click="pullLatest" :disabled="pulling" class="btn btn-success">
              {{ pulling ? 'Pulling...' : 'Pull to Browser' }}
            </button>
          </div>
        </div>

        <div class="mt-4 grid grid-cols-2 gap-4 text-sm text-gray-600">
          <div>
            <span class="font-medium">Created:</span> {{ formatDate(diagram.created_at) }}
          </div>
          <div>
            <span class="font-medium">Updated:</span> {{ formatDate(diagram.updated_at) }}
          </div>
        </div>
      </div>

      <!-- Version History -->
      <div class="card">
        <h2 class="text-xl font-bold text-gray-900 mb-4">Version History</h2>
        
        <div v-if="versionsLoading" class="text-center py-8">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto"></div>
        </div>

        <div v-else-if="versions.length === 0" class="text-center py-8 text-gray-500">
          No version history available.
        </div>

        <div v-else class="space-y-3">
          <div v-for="version in versions" :key="version.id" 
               class="flex items-center justify-between p-4 bg-gray-50 rounded-lg hover:bg-gray-100 transition-colors">
            <div>
              <div class="flex items-center gap-2">
                <span class="font-medium text-gray-900">Version {{ version.version }}</span>
                <span v-if="version.version === diagram.version" 
                      class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                  Latest
                </span>
              </div>
              <p class="text-sm text-gray-500 mt-1">
                {{ version.description || 'No description' }}
              </p>
              <p class="text-xs text-gray-400 mt-1">
                {{ formatDate(version.created_at) }}
              </p>
            </div>
            <button @click="pullVersion(version.version)" 
                    :disabled="pulling"
                    class="btn btn-secondary text-sm">
              {{ pulling === version.version ? 'Pulling...' : 'Pull This Version' }}
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- Toast -->
    <div v-if="toast" 
         :class="['fixed bottom-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 transition-all', 
                  toast.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white']">
      {{ toast.message }}
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import { chartDB } from '../chartdb-client'
import { diagramFromJSON } from '../utils/diagram-export-import'

export default {
  name: 'DiagramDetail',
  setup() {
    const route = useRoute()
    const diagramId = route.params.diagramId
    
    const diagram = ref(null)
    const versions = ref([])
    const loading = ref(true)
    const versionsLoading = ref(true)
    const pulling = ref(null)
    const toast = ref(null)

    const showToast = (message, type = 'success') => {
      toast.value = { message, type }
      setTimeout(() => { toast.value = null }, 3000)
    }

    const loadDiagram = async () => {
      try {
        diagram.value = await api.getDiagram(diagramId)
      } catch (err) {
        console.error('Failed to load diagram:', err)
      } finally {
        loading.value = false
      }
    }

    const loadVersions = async () => {
      try {
        versions.value = await api.getVersions(diagramId)
      } catch (err) {
        console.error('Failed to load versions:', err)
      } finally {
        versionsLoading.value = false
      }
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

    const pullLatest = async () => {
      try {
        pulling.value = 'latest'
        
        // Get diagram data from server
        const diagramData = await api.pullDiagram(diagramId)
        
        // Use ChartDB-compatible import function to clone with new IDs
        // This ensures relationships, areas, and all entities are properly remapped
        const diagramJSON = JSON.stringify(diagramData)
        const clonedDiagram = diagramFromJSON(diagramJSON)
        
        // Save to IndexedDB using the cloned diagram
        await chartDB.saveDiagramJSON(clonedDiagram)
        
        showToast('Diagram pulled to browser!')
      } catch (err) {
        showToast('Failed to pull: ' + err.message, 'error')
      } finally {
        pulling.value = null
      }
    }

    const pullVersion = async (version) => {
      try {
        pulling.value = version
        
        // Get specific version from server
        const diagramData = await api.pullDiagram(diagramId, version)
        
        // Use ChartDB-compatible import function to clone with new IDs
        // This ensures relationships, areas, and all entities are properly remapped
        const diagramJSON = JSON.stringify(diagramData)
        const clonedDiagram = diagramFromJSON(diagramJSON)
        
        // Save to IndexedDB using the cloned diagram
        await chartDB.saveDiagramJSON(clonedDiagram)
        
        showToast(`Version ${version} pulled to browser!`)
      } catch (err) {
        showToast('Failed to pull: ' + err.message, 'error')
      } finally {
        pulling.value = null
      }
    }

    onMounted(() => {
      loadDiagram()
      loadVersions()
    })

    return {
      diagram,
      versions,
      loading,
      versionsLoading,
      pulling,
      toast,
      formatDate,
      pullLatest,
      pullVersion
    }
  }
}
</script>
