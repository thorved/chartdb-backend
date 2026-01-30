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

        <div v-if="status === 'error'" class="mt-6">
          <button @click="retrySync" class="btn btn-primary w-full">
            Retry Sync
          </button>
          <button @click="skipSync" class="btn btn-secondary w-full mt-3">
            Skip & Continue
          </button>
        </div>

        <div v-if="status === 'success'" class="mt-6">
          <button @click="goToDashboard" class="btn btn-primary w-full">
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

    const syncCloudData = async () => {
      try {
        status.value = 'syncing'
        title.value = 'Syncing Data...'
        message.value = 'Fetching your diagrams from the cloud...'
        console.log('[Sync] Starting cloud data sync...')
        
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
            router.push('/dashboard')
          }, 2000)
          return
        }
        
        message.value = `Found ${cloudDiagrams.length} diagrams. Replacing local data...`
        
        // Clear local IndexedDB first
        console.log('[Sync] Clearing local IndexedDB...')
        await chartDB.clearAllDiagrams()
        console.log('[Sync] Local IndexedDB cleared successfully')
        
        // Reopen database after clearing
        console.log('[Sync] Reopening database...')
        await chartDB.reopen()
        console.log('[Sync] Database reopened successfully')
        
        // Save each cloud diagram to local IndexedDB
        for (let i = 0; i < cloudDiagrams.length; i++) {
          const diagram = cloudDiagrams[i]
          message.value = `Syncing diagram ${i + 1} of ${cloudDiagrams.length}: ${diagram.name || 'Untitled'}...`
          console.log(`[Sync] Saving diagram ${i + 1}:`, diagram.id, diagram.name)
          
          try {
            // Ensure all required fields are present with correct types
            const diagramData = {
              id: diagram.id,
              name: diagram.name || 'Untitled',
              databaseType: diagram.databaseType || 'generic',
              databaseEdition: diagram.databaseEdition,
              tables: Array.isArray(diagram.tables) ? diagram.tables : [],
              relationships: Array.isArray(diagram.relationships) ? diagram.relationships : [],
              dependencies: Array.isArray(diagram.dependencies) ? diagram.dependencies : [],
              areas: Array.isArray(diagram.areas) ? diagram.areas : [],
              notes: Array.isArray(diagram.notes) ? diagram.notes : [],
              customTypes: Array.isArray(diagram.customTypes) ? diagram.customTypes : []
            }
            
            await chartDB.saveDiagramFull(diagramData)
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
          router.push('/dashboard')
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
      router.push('/dashboard')
    }

    const goToDashboard = () => {
      router.push('/dashboard')
    }

    onMounted(() => {
      syncCloudData()
    })

    return {
      status,
      title,
      message,
      diagramCount,
      retrySync,
      skipSync,
      goToDashboard
    }
  }
}
</script>
