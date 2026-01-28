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
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

export default {
  name: 'App',
  setup() {
    const router = useRouter()
    const route = useRoute()
    const user = ref(null)

    const isAuthenticated = computed(() => {
      return !!localStorage.getItem('chartdb_sync_token')
    })

    const loadUser = () => {
      const userData = localStorage.getItem('chartdb_sync_user')
      if (userData) {
        user.value = JSON.parse(userData)
      }
    }

    const logout = () => {
      localStorage.removeItem('chartdb_sync_token')
      localStorage.removeItem('chartdb_sync_user')
      user.value = null
      router.push('/login')
    }

    onMounted(loadUser)
    watch(route, loadUser)

    return {
      user,
      isAuthenticated,
      logout
    }
  }
}
</script>
