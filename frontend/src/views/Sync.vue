<template>
  <div class="min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full">
      <div class="card">
        <div class="text-center mb-8">
          <div class="mb-4">
            <svg v-if="status === 'syncing'" class="animate-spin h-12 w-12 text-primary-600 mx-auto" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            <svg v-else-if="status === 'success'" class="h-12 w-12 text-green-600 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <svg v-else-if="status === 'error'" class="h-12 w-12 text-red-600 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          
          <h1 class="text-2xl font-bold text-gray-900">{{ title }}</h1>
          
          <p class="mt-2 text-gray-600">{{ message }}</p>
          
          <div v-if="diagramCount > 0" class="mt-4 text-sm text-gray-500">
            Found {{ diagramCount }} diagram{{ diagramCount !== 1 ? 's' : '' }} in cloud
          </div>
        </div>

        <div v-if="status === 'error'" class="mt-6 space-y-3">
          <button v-if="dbNotInitialized" @click="openChartDBAndWait" class="btn btn-primary w-full">
            Open ChartDB & Initialize
          </button>
          <button @click="retrySync" class="btn btn-secondary w-full">
            {{ dbNotInitialized ? 'Check Again' : 'Retry Sync' }}
          </button>
          <button @click="goToDashboard" class="btn btn-outline w-full">
            Go to Dashboard
          </button>
        </div>

        <div v-if="status === 'success'" class="mt-6 space-y-3">
          <button @click="goToApplication" class="btn btn-primary w-full">
            Open ChartDB
          </button>
          <button @click="goToDashboard" class="btn btn-secondary w-full">
            Go to Dashboard
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { chartDB } from '../chartdb-client'

  export default {
  name: 'Sync',
  setup() {
    const router = useRouter()
    const status = ref('syncing')
    const title = ref('Syncing Data...')
    const message = ref('Fetching your diagrams from the cloud...')
    const diagramCount = ref(0)
    const dbNotInitialized = ref(false)

    const syncCloudData = async () => {
      try {
        status.value = 'syncing'
        title.value = 'Syncing Data...'
        message.value = 'Checking database...'
        console.log('[Sync] Starting cloud data sync...')
        
        // Check if ChartDB database exists with proper schema
        console.log('[Sync] Checking ChartDB database...')
        let dbReady = false
        try {
          dbReady = await chartDB.checkDatabase()
        } catch (dbErr) {
          console.error('[Sync] Database check failed:', dbErr)
        }
        
        if (!dbReady) {
          console.log('[Sync] ChartDB database not found')
          status.value = 'error'
          title.value = 'ChartDB Not Initialized'
          message.value = 'ChartDB database not found. Please open ChartDB first.'
          dbNotInitialized.value = true
          return
        } else {
          console.log('[Sync] ChartDB database is ready')
          dbNotInitialized.value = false
        }
        
        message.value = 'Fetching your diagrams from the cloud...'
        
        // Get all diagrams from cloud
        const response = await api.pullAllDiagrams()
        console.log('[Sync] Cloud response:', response)
        
        const cloudDiagrams = response.diagrams || []
        diagramCount.value = cloudDiagrams.length
        console.log(`[Sync] Found ${cloudDiagrams.length} diagrams in cloud`)
        
        if (cloudDiagrams.length === 0) {
          status.value = 'success'
          title.value = 'No Cloud Data'
          message.value = 'No diagrams found in cloud. You can create new diagrams in ChartDB.'
          console.log('[Sync] No cloud diagrams found, skipping sync')
          
          // Auto-redirect after 2 seconds
          setTimeout(() => {
            window.location.href = '/'
          }, 2000)
          return
        }
        
        message.value = `Found ${cloudDiagrams.length} diagrams. Replacing local data...`
        
        // Clear local IndexedDB first
        console.log('[Sync] Clearing local IndexedDB...')
        try {
          await chartDB.clearAllDiagrams()
          console.log('[Sync] Local IndexedDB cleared successfully')
        } catch (clearErr) {
          console.warn('[Sync] Error clearing (may be empty):', clearErr)
        }
        
        // Ensure database is still open after clearing
        console.log('[Sync] Ensuring database connection...')
        try {
          await chartDB.ensureOpen()
          console.log('[Sync] Database connection ready')
        } catch (openErr) {
          console.error('[Sync] Failed to ensure database connection:', openErr)
          throw openErr
        }
        
        // Save each cloud diagram to local IndexedDB using JSON format
        for (let i = 0; i < cloudDiagrams.length; i++) {
          const diagram = cloudDiagrams[i]
          message.value = `Syncing diagram ${i + 1} of ${cloudDiagrams.length}: ${diagram.name || 'Untitled'}...`
          console.log(`[Sync] Saving diagram ${i + 1}:`, diagram.id, diagram.name)
          console.log(`[Sync] Diagram data:`, {
            id: diagram.id,
            name: diagram.name,
            databaseType: diagram.databaseType,
            tablesCount: diagram.tables?.length || 0,
            relationshipsCount: diagram.relationships?.length || 0,
            hasTables: !!diagram.tables,
            hasRelationships: !!diagram.relationships
          })
          
          try {
            // Use the new JSON-based save method
            await chartDB.saveDiagramJSON(diagram)
            console.log(`[Sync] Successfully saved diagram ${i + 1}`)
          } catch (saveErr) {
            console.error(`[Sync] Failed to save diagram ${i + 1}:`, saveErr)
            throw saveErr
          }
        }
        
        status.value = 'success'
        title.value = 'Sync Complete!'
        message.value = `Successfully synced ${cloudDiagrams.length} diagram${cloudDiagrams.length !== 1 ? 's' : ''} from cloud.`
        console.log('[Sync] Cloud sync completed successfully')
        
        // Auto-redirect after 2 seconds
        setTimeout(() => {
          window.location.href = '/'
        }, 2000)
      } catch (err) {
        console.error('[Sync] Failed to sync cloud data:', err)
        status.value = 'error'
        title.value = 'Sync Failed'
        message.value = 'Failed to sync data from cloud: ' + (err.message || 'Unknown error')
      }
    }

    const retrySync = () => {
      syncCloudData()
    }

    const skipSync = () => {
      window.location.href = '/'
    }

    const goToApplication = () => {
      window.location.href = '/'
    }

    const goToDashboard = () => {
      router.push('/dashboard')
    }

    const openChartDBAndWait = async () => {
      // Open ChartDB in a popup (this works because it's triggered by user click)
      const chartdbWindow = window.open('/', 'chartdb-init', 'width=1200,height=800')
      
      if (!chartdbWindow || chartdbWindow.closed || typeof chartdbWindow.closed === 'undefined') {
        // Popup was blocked, fallback to redirect
        window.location.href = '/?returnTo=sync'
        return
      }
      
      status.value = 'syncing'
      title.value = 'Waiting for ChartDB...'
      message.value = 'Please wait while ChartDB initializes (max 10 seconds)...'
      
      // Poll for database creation (max 10 seconds)
      let attempts = 0
      const maxAttempts = 20 // 20 * 500ms = 10 seconds
      
      const checkInterval = setInterval(async () => {
        attempts++
        message.value = `Waiting for ChartDB to initialize (${attempts}/${maxAttempts})...`
        
        try {
          const dbReady = await chartDB.checkDatabase()
          if (dbReady) {
            clearInterval(checkInterval)
            try {
              chartdbWindow.close()
            } catch (e) {}
            // Database ready, retry sync
            syncCloudData()
            return
          }
        } catch (e) {
          // Ignore errors during check
        }
        
        if (attempts >= maxAttempts) {
          clearInterval(checkInterval)
          try {
            chartdbWindow.close()
          } catch (e) {}
          
          status.value = 'error'
          title.value = 'ChartDB Not Initialized'
          message.value = 'ChartDB did not initialize in time. Please create a diagram in ChartDB and try again.'
        }
      }, 500)
    }

    onMounted(() => {
      syncCloudData()
    })

    return {
      status,
      title,
      message,
      diagramCount,
      dbNotInitialized,
      retrySync,
      skipSync,
      goToDashboard,
      goToApplication,
      openChartDBAndWait
    }
  }
}
</script>
