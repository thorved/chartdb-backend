<template>
  <div class="min-h-screen bg-gray-100">
    <nav v-if="isAuthenticated" class="bg-white shadow-sm">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <router-link to="/dashboard" class="text-xl font-bold text-primary-600">
              ChartDB Sync
            </router-link>
          </div>
          <div class="flex items-center gap-4">
            <span class="text-gray-600">{{ user?.name || user?.email }}</span>
            <button @click="logout" class="btn btn-secondary text-sm">
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>
    <router-view />
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from './api'
import { chartDB } from './chartdb-client'
import { isAuthenticated, checkAuth } from './main'

export default {
  name: 'App',
  setup() {
    const router = useRouter()
    const user = ref(null)

    const loadUser = async () => {
      try {
        user.value = await api.getCurrentUser()
      } catch (err) {
        user.value = null
      }
    }

    const logout = async () => {
      // Clear local diagrams
      try {
        await chartDB.clearAllDiagrams()
      } catch (err) {
        console.error('Failed to clear local diagrams:', err)
      }
      
      // Call logout API
      try {
        await api.logout()
      } catch (err) {
        console.error('Logout API error:', err)
      }
      
      // Update local state
      user.value = null
      isAuthenticated.value = false
      
      // Redirect to login
      router.push('/login')
    }

    onMounted(async () => {
      await checkAuth()
      if (isAuthenticated.value) {
        await loadUser()
      }
    })

    return {
      user,
      isAuthenticated,
      logout
    }
  }
}
</script>
