import { defineStore } from 'pinia'
import { api, clearToken, getToken, setToken, type User } from '@/api/client'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: getToken(),
    user: null as User | null,
  }),
  actions: {
    async login(username: string, password: string) {
      const resp = await api.login(username, password)
      this.token = resp.token
      this.user = resp.user
      setToken(resp.token)
    },
    async loadMe() {
      this.user = await api.me()
    },
    async logout() {
      try {
        await api.logout()
      } finally {
        this.clear()
      }
    },
    clear() {
      this.token = ''
      this.user = null
      clearToken()
    },
  },
})
