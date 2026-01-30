import { createApp, ref } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

// Import views
import Login from './views/Login.vue'
import Signup from './views/Signup.vue'
import Sync from './views/Sync.vue'
import Dashboard from './views/Dashboard.vue'
import DiagramDetail from './views/DiagramDetail.vue'

// Reactive auth state
export const isAuthenticated = ref(false)
export const authChecked = ref(false)

// Check auth status with backend
export async function checkAuth() {
  try {
    const response = await fetch('/sync/api/auth/me', {
      credentials: 'include'
    })
    isAuthenticated.value = response.ok
    authChecked.value = true
    return response.ok
  } catch (err) {
    isAuthenticated.value = false
    authChecked.value = true
    return false
  }
}

// Create router
const router = createRouter({
  history: createWebHistory('/sync/'),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: Login, meta: { guest: true } },
    { path: '/signup', component: Signup, meta: { guest: true } },
    { path: '/sync', component: Sync, meta: { requiresAuth: true } },
    { path: '/dashboard', component: Dashboard, meta: { requiresAuth: true } },
    { path: '/dashboard/:diagramId', component: DiagramDetail, meta: { requiresAuth: true } },
  ]
})

// Navigation guard
router.beforeEach(async (to, from, next) => {
  // Always check auth status
  await checkAuth()
  
  if (to.meta.requiresAuth && !isAuthenticated.value) {
    next('/login')
  } else if (to.meta.guest && isAuthenticated.value) {
    next('/dashboard')
  } else {
    next()
  }
})

const app = createApp(App)
app.use(router)
app.mount('#app')
