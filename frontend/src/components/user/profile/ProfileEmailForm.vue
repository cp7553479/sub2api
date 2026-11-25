<template>
  <form class="card space-y-4 p-6" @submit.prevent="updateEmail">
    <label class="input-label" for="account-email">{{ t('auth.emailLabel') }}</label>
    <input id="account-email" v-model="email" class="input" type="email" required autocomplete="email" />
    <button type="submit" class="btn btn-primary" :disabled="loading">{{ t('profile.updateProfile') }}</button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { authAPI } from '@/api'
import { useAppStore, useAuthStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
const { t } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const email = ref(auth.user?.email || '')
const loading = ref(false)
async function updateEmail() {
  loading.value = true
  try {
    const updated = await authAPI.updateAccount({ email: email.value })
    await auth.refreshUser()
    app.showSuccess(updated.email?.toLowerCase() === email.value.trim().toLowerCase()
      ? t('profile.updateSuccess') : t('auth.emailConfirmation'))
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('profile.updateFailed')))
  } finally {
    loading.value = false
  }
}
</script>
