import { ref, computed } from 'vue'
import type { User } from '@/types/api'

const user = ref<User | null>(null)

export const useAuth = () => {
  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  const loadUser = () => {
    const stored = localStorage.getItem('ips_user')
    if (stored) {
      try {
        user.value = JSON.parse(stored)
      } catch (e) {
        console.error('Failed to parse user from localStorage', e)
      }
    }
  }

  const setUser = (userData: User) => {
    user.value = userData
    localStorage.setItem('ips_user', JSON.stringify(userData))
  }

  const clearUser = () => {
    user.value = null
    localStorage.removeItem('ips_user')
  }

  return {
    user,
    isAuthenticated,
    isAdmin,
    loadUser,
    setUser,
    clearUser,
  }
}
