<template>
  <div class="min-h-screen flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full">
      <div class="card">
        <div class="text-center mb-8">
          <h1 class="text-3xl font-bold text-primary-600">ChartDB Sync</h1>
          <p class="mt-2 text-gray-600">Sign in to your account</p>
        </div>

        <!-- OIDC Login Button -->
        <div v-if="oidcEnabled" class="mb-6">
          <a
            href="/sync/api/auth/oidc/login"
            class="w-full flex items-center justify-center px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary-500"
          >
            <svg class="h-5 w-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
            </svg>
            Sign in with SSO
          </a>
          <div class="relative mt-6">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-gray-300"></div>
            </div>
            <div class="relative flex justify-center text-sm">
              <span class="px-2 bg-white text-gray-500">Or continue with</span>
            </div>
          </div>
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

          <div v-if="error" class="text-red-600 text-sm text-center">
            {{ error }}
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="btn btn-primary w-full"
          >
            {{ loading ? 'Signing in...' : 'Sign in' }}
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
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'

export default {
  name: 'Login',
  setup() {
    const router = useRouter()
    const email = ref('')
    const password = ref('')
    const error = ref('')
    const loading = ref(false)
    const oidcEnabled = ref(false)

    const checkOIDCEnabled = async () => {
      try {
        const response = await fetch('/sync/api/auth/oidc/enabled')
        const data = await response.json()
        oidcEnabled.value = data.enabled
      } catch (err) {
        console.error('Failed to check OIDC status:', err)
        oidcEnabled.value = false
      }
    }

    const handleLogin = async () => {
      error.value = ''
      loading.value = true

      try {
        await api.login(email.value, password.value)
        
        // Redirect to sync page to pull cloud data
        router.push('/sync')
      } catch (err) {
        error.value = err.message
        loading.value = false
      }
    }

    onMounted(() => {
      checkOIDCEnabled()
    })

    return {
      email,
      password,
      error,
      loading,
      oidcEnabled,
      handleLogin
    }
  }
}
</script>
