<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('auth.welcomeBack') }}</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('auth.signInToAccount') }}</p>
      </div>
      <form class="space-y-5" @submit.prevent="handleLogin">
        <div>
          <label class="input-label" for="email">{{ t('auth.emailLabel') }}</label>
          <input id="email" v-model="email" class="input" type="email" autocomplete="email" required autofocus :disabled="loading" />
        </div>
        <div>
          <label class="input-label" for="password">{{ t('auth.passwordLabel') }}</label>
          <input id="password" v-model="password" class="input" type="password" autocomplete="current-password" required :disabled="loading" />
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
        <button class="btn btn-primary w-full" type="submit" :disabled="loading || !agreement.canSubmit.value">{{ loading ? t('auth.signingIn') : t('auth.signIn') }}</button>
      </form>
    </div>
    <template #footer>
      <p class="text-gray-500 dark:text-dark-400">{{ t('auth.dontHaveAccount') }} <router-link class="font-medium text-primary-600 dark:text-primary-400" to="/register">{{ t('auth.signUp') }}</router-link></p>
    </template>
  </AuthLayout>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { AuthLayout } from '@/components/layout'
import { useAuthStore, useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import LoginAgreementPrompt from '@/components/auth/LoginAgreementPrompt.vue'
import { useLoginAgreement } from '@/composables/useLoginAgreement'
const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const app = useAppStore()
const email = ref('')
const password = ref('')
const loading = ref(false)
const agreement = useLoginAgreement()
async function handleLogin() {
  if (!agreement.canSubmit.value) return
  loading.value = true
  try {
    await auth.login({ email: email.value, password: password.value })
    const redirect = route.query.redirect
    const destination = typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
      ? redirect : auth.isAdmin ? '/admin/dashboard' : '/dashboard'
    await router.push(destination)
  } catch (error) {
    app.showError(extractApiErrorMessage(error, t('auth.loginFailed')))
  } finally {
    loading.value = false
  }
}
</script>
