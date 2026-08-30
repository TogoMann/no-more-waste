<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api, authState, setSession } from '../../services/api.js'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const profile = ref(null)
const cities = ref([])
const form = ref({ full_name: '', phone: '', address: '', city_id: null })
const passwordForm = ref({ current_password: '', new_password: '' })
const profileMessage = ref('')
const profileError = ref('')
const passwordMessage = ref('')
const passwordError = ref('')

const initials = computed(() => {
  const name = form.value.full_name || '?'
  return name.split(' ').map((part) => part.charAt(0)).slice(0, 2).join('').toUpperCase()
})

function roleLabel(role) {
  return t(`roles.${role}`)
}

onMounted(async () => {
  cities.value = await api.get('/cities')
  profile.value = await api.get('/profile')
  try {
    duesInfo.value = await api.get('/dues')
  } catch (err) {
    duesInfo.value = null
  }
  if (route.query.session_id) {
    await confirmPayment(route.query.session_id)
  } else if (route.query.payment === 'cancelled') {
    duesError.value = t('stripe.cancelled')
    router.replace({ path: '/espace/profil' })
  }
  form.value = {
    full_name: profile.value.full_name,
    phone: profile.value.phone,
    address: profile.value.address,
    city_id: profile.value.city_id
  }
})

async function saveProfile() {
  profileMessage.value = ''
  profileError.value = ''
  try {
    await api.put('/profile', form.value)
    profile.value = await api.get('/profile')
    setSession(authState.token, { ...authState.user, full_name: profile.value.full_name })
    profileMessage.value = t('profile.updated')
  } catch (err) {
    profileError.value = err.message
  }
}

const duesMessage = ref('')
const duesError = ref('')
const duesInfo = ref(null)
const paying = ref(false)

const duesAmount = computed(() => {
  if (!duesInfo.value) return '20,00 €'
  return (duesInfo.value.amount_cents / 100).toFixed(2).replace('.', ',') + ' €'
})

async function payDues() {
  duesMessage.value = ''
  duesError.value = ''
  paying.value = true
  try {
    const session = await api.post('/profile/dues/checkout')
    window.location.href = session.url
  } catch (err) {
    duesError.value = err.message
    paying.value = false
  }
}

async function confirmPayment(sessionId) {
  duesMessage.value = ''
  duesError.value = ''
  try {
    await api.post('/profile/dues/confirm', { session_id: sessionId })
    profile.value = await api.get('/profile')
    duesMessage.value = t('stripe.success')
  } catch (err) {
    duesError.value = err.message
  }
  router.replace({ path: '/espace/profil' })
}

async function savePassword() {
  passwordMessage.value = ''
  passwordError.value = ''
  try {
    await api.put('/profile/password', passwordForm.value)
    passwordForm.value = { current_password: '', new_password: '' }
    passwordMessage.value = t('profile.passwordUpdated')
  } catch (err) {
    passwordError.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('profile.title') }}</h1>
      <p class="page-subtitle">{{ t('profile.subtitle') }}</p>
    </div>

    <div v-if="profile" class="profile-header card">
      <div class="avatar profile-avatar">{{ initials }}</div>
      <div>
        <h2 style="font-size:20px">{{ profile.full_name }}</h2>
        <p style="color:var(--muted);font-size:14px">{{ profile.email }}</p>
        <div style="margin-top:8px;display:flex;gap:8px;flex-wrap:wrap">
          <span class="badge active">{{ roleLabel(profile.role) }}</span>
          <span v-if="profile.city_name" class="skill-tag">📍 {{ profile.city_name }}</span>
          <span class="skill-tag">{{ t('profile.memberSince') }} {{ (profile.created_at || '').slice(0, 10) }}</span>
        </div>
      </div>
    </div>

    <div v-if="profile" class="card dues-card" :class="profile.membership_valid ? 'valid' : 'invalid'">
      <div class="card-title">{{ profile.membership_valid ? '✅' : '⚠️' }} {{ t('dues.title') }}</div>
      <div class="dues-body">
        <div>
          <div class="dues-status">
            {{ profile.membership_valid ? t('dues.valid') : (profile.membership_end_date ? t('dues.expired') : t('dues.never')) }}
          </div>
          <div class="dues-date" v-if="profile.membership_end_date">
            {{ profile.membership_valid ? t('dues.validUntil') : t('dues.expiredSince') }}
            {{ profile.membership_end_date }}
          </div>
          <p v-if="!profile.membership_valid" class="dues-warning">{{ t('dues.warning') }}</p>
        </div>
        <div class="dues-actions">
          <button class="btn accent" :disabled="paying" @click="payDues">
            💳 {{ paying ? t('stripe.redirect') : t('stripe.pay') }} — {{ duesAmount }}
          </button>
          <small class="stripe-hint">🔒 {{ t('stripe.securedBy') }}</small>
          <small class="stripe-hint">{{ t('stripe.testCard') }}</small>
        </div>
      </div>
      <p v-if="duesError" class="error">{{ duesError }}</p>
      <p v-if="duesMessage" class="success">{{ duesMessage }}</p>
    </div>

    <div class="panel-grid">
      <div class="card">
        <div class="card-title">👤 {{ t('profile.personalInfo') }}</div>
        <div class="field">
          <label>{{ t('auth.fullName') }}</label>
          <input v-model="form.full_name" />
        </div>
        <div class="field">
          <label>{{ t('common.phone') }}</label>
          <input v-model="form.phone" />
        </div>
        <div class="field">
          <label>{{ t('common.address') }}</label>
          <input v-model="form.address" />
        </div>
        <div class="field">
          <label>{{ t('profile.city') }}</label>
          <select v-model.number="form.city_id">
            <option :value="null">—</option>
            <option v-for="city in cities" :key="city.id" :value="city.id">{{ city.name }}</option>
          </select>
        </div>
        <p v-if="profileError" class="error">{{ profileError }}</p>
        <p v-if="profileMessage" class="success">{{ profileMessage }}</p>
        <button class="btn" @click="saveProfile">{{ t('common.save') }}</button>
      </div>

      <div class="card">
        <div class="card-title">🔒 {{ t('profile.security') }}</div>
        <div class="field">
          <label>{{ t('profile.currentPassword') }}</label>
          <input v-model="passwordForm.current_password" type="password" />
        </div>
        <div class="field">
          <label>{{ t('profile.newPassword') }}</label>
          <input v-model="passwordForm.new_password" type="password" />
        </div>
        <p v-if="passwordError" class="error">{{ passwordError }}</p>
        <p v-if="passwordMessage" class="success">{{ passwordMessage }}</p>
        <button class="btn secondary" @click="savePassword">{{ t('profile.changePassword') }}</button>
      </div>
    </div>
  </div>
</template>
