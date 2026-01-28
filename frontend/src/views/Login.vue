<template>
  <div class="min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full">
      <div class="card">
        <div class="text-center mb-8">
          <h1 class="text-3xl font-bold text-primary-600">ChartDB Sync</h1>
          <p class="mt-2 text-gray-600">Sign in to your account</p>
        </div>

        <form @submit.prevent="handleLogin" class="space-y-6">
          <div>
            <label for="email" class="block text-sm font-medium text-gray-700 mb-1">
              Email
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              required
              class="input"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label for="password" class="block text-sm font-medium text-gray-700 mb-1">
              Password
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              required
              class="input"
              placeholder="••••••••"
            />
          </div>

          <!-- Sync option on login -->
          <div class="flex items-center">
            <input
              id="syncOnLogin"
              v-model="syncOnLogin"
              type="checkbox"
              class="h-4 w-4 text-primary-600 focus:ring-primary-500 border-gray-300 rounded"
            />
            <label for="syncOnLogin" class="ml-2 block text-sm text-gray-700">
              Replace local diagrams with cloud diagrams on login
            </label>
          </div>

          <div v-if="error" class="text-red-600 text-sm text-center">
            {{ error }}
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="btn btn-primary w-full"
          >
            {{ loading ? (syncing ? 'Syncing diagrams...' : 'Signing in...') : 'Sign in' }}
          </button>
        </form>

        <div class="mt-6 text-center">
          <p class="text-gray-600">
            Don't have an account?
            <router-link to="/signup" class="text-primary-600 hover:text-primary-700 font-medium">
              Sign up
            </router-link>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import { chartDB } from '../chartdb-client'

export default {
  name: 'Login',
  setup() {
    const router = useRouter()
    const email = ref('')
    const password = ref('')
    const error = ref('')
    const loading = ref(false)
    const syncing = ref(false)
    const syncOnLogin = ref(true)

    const handleLogin = async () => {
      error.value = ''
      loading.value = true

      try {
        await api.login(email.value, password.value)
        
        // If sync on login is enabled, pull all cloud diagrams and replace local
        if (syncOnLogin.value) {
          syncing.value = true
          try {
            const result = await api.pullAllDiagrams()
            if (result.diagrams && result.diagrams.length > 0) {
              // Reopen/create the database
              await chartDB.reopen()
              
              // Clear all existing local diagrams first
              await chartDB.clearAllDiagrams()
              
              // Save all cloud diagrams to local IndexedDB
              for (const diagramData of result.diagrams) {
                await chartDB.saveDiagramFull(diagramData)
              }
              console.log(`Synced ${result.diagrams.length} diagrams from cloud`)
            }
          } catch (syncErr) {
            console.error('Failed to sync diagrams on login:', syncErr)
            // Don't block login if sync fails
          }
          syncing.value = false
        }
        
        router.push('/dashboard')
      } catch (err) {
        error.value = err.message
      } finally {
        loading.value = false
      }
    }

    return {
      email,
      password,
      error,
      loading,
      syncing,
      syncOnLogin,
      handleLogin
    }
  }
}
</script>
