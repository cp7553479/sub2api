<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('auth.createAccount') }}</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('auth.signUpToStart', { siteName: app.siteName }) }}</p>
      </div>
      <p v-if="confirmationRequired" role="status" class="rounded-xl bg-primary-50 p-4 text-sm text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ t('auth.emailConfirmation') }}</p>
      <form v-else class="space-y-5" @submit.prevent="handleRegister">
        <div>
          <label class="input-label" for="email">{{ t('auth.emailLabel') }}</label>
          <input id="email" v-model="email" class="input" type="email" autocomplete="email" required autofocus :disabled="loading" />
        </div>
        <div>
          <label class="input-label" for="password">{{ t('auth.passwordLabel') }}</label>
          <input id="password" v-model="password" class="input" type="password" autocomplete="new-password" required minlength="6" :disabled="loading" />
          <p class="input-hint">{{ t('auth.passwordHint') }}</p>
        </div>
        <LoginAgreementPrompt
          v-if="agreement.enabled.value"
          :accepted="agreement.accepted.value"
          :documents="agreement.documents.value"
          :mode="agreement.mode.value"
          :updated-at="agreement.updatedAt.value"
          :visible="agreement.visible.value"
          @accept="agreement.accept"
          @reject="agreement.reject"
          @open="agreement.visible.value = true"
        />
        <button class="btn btn-primary w-full" type="submit" :disabled="loading || !agreement.canSubmit.value">{{ loading ? t('auth.processing') : t('auth.signUp') }}</button>
      </form>
    </div>
    <template #footer>
      <p class="text-gray-500 dark:text-dark-400">{{ t('auth.alreadyHaveAccount') }} <router-link class="font-medium text-primary-600 dark:text-primary-400" to="/login">{{ t('auth.signIn') }}</router-link></p>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { AuthLayout } from '@/components/layout'
import { useAuthStore, useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import { useLoginAgreement } from '@/composables/useLoginAgreement'
const { t } = useI18n()
const router = useRouter()
const auth = useAuthStore()
const app = useAppStore()
const email = ref('')
const password = ref('')
const loading = ref(false)
const agreement = useLoginAgreement()
const confirmationRequired = ref(false)
async function handleRegister() {
  if (!agreement.canSubmit.value) return
  loading.value = true
  try {
    const user = await auth.register({ email: email.value, password: password.value })
    password.value = ''
    if (!user) confirmationRequired.value = true
    else await router.push(auth.isAdmin ? '/admin/dashboard' : '/dashboard')
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('auth.registrationFailed')))
  } finally {
    loading.value = false
  }
}
</script>
