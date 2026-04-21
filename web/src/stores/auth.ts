import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const TOKEN_KEY = 'auth_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) ?? '')
  // noAuth: server has no AUTH_TOKEN configured, auth is disabled entirely
  const noAuth = ref<boolean>(false)

  const isLoggedIn = computed(() => noAuth.value || token.value.length > 0)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  function setNoAuth() {
    noAuth.value = true
  }

  function clear() {
    token.value = ''
    noAuth.value = false
    localStorage.removeItem(TOKEN_KEY)
  }

  return { token, noAuth, isLoggedIn, setToken, setNoAuth, clear }
})
