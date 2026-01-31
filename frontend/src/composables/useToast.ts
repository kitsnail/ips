import { ref } from 'vue'

const toastVisible = ref(false)
const toastMessage = ref('')
const toastType = ref<'success' | 'error' | 'info'>('success')

export const useToast = () => {
  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    toastMessage.value = message
    toastType.value = type
    toastVisible.value = true

    setTimeout(() => {
      toastVisible.value = false
    }, 3000)
  }

  const hideToast = () => {
    toastVisible.value = false
  }

  return {
    toastVisible,
    toastMessage,
    toastType,
    showToast,
    hideToast,
  }
}
