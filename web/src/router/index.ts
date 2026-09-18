import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppsView from '@/views/AppsView.vue'
import LoginView from '@/views/LoginView.vue'
import UsersView from '@/views/UsersView.vue'
import APIKeysView from '@/views/APIKeysView.vue'
import AuditLogsView from '@/views/AuditLogsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/apps' },
    { path: '/login', component: LoginView },
    { path: '/apps', component: AppsView },
    { path: '/users', component: UsersView },
    { path: '/api-keys', component: APIKeysView },
    { path: '/audit-logs', component: AuditLogsView },
  ],
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (to.path === '/login') return true
  if (!auth.token) return '/login'
  if (!auth.user) {
    try {
      await auth.loadMe()
    } catch {
      auth.clear()
      return '/login'
    }
  }
  if ((to.path === '/users' || to.path === '/api-keys' || to.path === '/audit-logs') && auth.user?.role !== 'admin') return '/apps'
  return true
})

export default router
