import { computed, ref, watch } from 'vue'
import { useAppStore } from '@/stores/app'

const STORAGE_KEY = 'sub2api_login_agreement_consent'

/** Site document acknowledgement is local UI state, separate from platform authentication. */
export function useLoginAgreement() {
  const app = useAppStore()
  const settings = computed(() => app.cachedPublicSettings)
  const documents = computed(() => settings.value?.login_agreement_documents?.filter((document) => document.title.trim()) || [])
  const enabled = computed(() => settings.value?.login_agreement_enabled === true && documents.value.length > 0)
  const mode = computed(() => settings.value?.login_agreement_mode === 'checkbox' ? 'checkbox' : 'modal')
  const updatedAt = computed(() => settings.value?.login_agreement_updated_at || '')
  const revision = computed(() => settings.value?.login_agreement_revision || `${updatedAt.value}:${documents.value.map((document) => `${document.id}:${document.title}`).join('|')}`)
  const accepted = ref(false)
  const visible = ref(false)
  const canSubmit = computed(() => app.publicSettingsLoaded && (!enabled.value || accepted.value))

  watch([enabled, revision, mode], () => {
    let storedRevision: string | undefined
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored) {
      try {
        storedRevision = (JSON.parse(stored) as { revision?: string }).revision
      } catch {
        localStorage.removeItem(STORAGE_KEY)
      }
    }
    accepted.value = !enabled.value || storedRevision === revision.value
    visible.value = enabled.value && !accepted.value && mode.value === 'modal'
  }, { immediate: true })

  function accept() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ revision: revision.value, accepted_at: new Date().toISOString() }))
    accepted.value = true
    visible.value = false
  }
  function reject() {
    localStorage.removeItem(STORAGE_KEY)
    accepted.value = false
    visible.value = false
  }
  return { documents, enabled, mode, updatedAt, accepted, visible, canSubmit, accept, reject }
}
