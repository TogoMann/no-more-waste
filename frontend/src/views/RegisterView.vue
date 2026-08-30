<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api, setSession } from '../services/api.js'
import AuthHero from '../components/AuthHero.vue'

const router = useRouter()
const { t } = useI18n()

const form = ref({
  full_name: '', email: '', password: '', role: 'member',
  company_name: '', siret: '', phone: '', address: '', city_id: null
})
const error = ref('')
const cities = ref([])
const siretState = ref({ checking: false, verified: false, message: '', error: '' })

const isMerchant = computed(() => form.value.role === 'merchant')

onMounted(async () => {
  try {
    cities.value = await api.get('/cities')
  } catch (err) {
    cities.value = []
  }
})

async function verifySiret() {
  siretState.value = { checking: true, verified: false, message: '', error: '' }
  try {
    const company = await api.get(`/siret/verify?siret=${encodeURIComponent(form.value.siret)}`)
    form.value.siret = company.siret
    if (company.name) {
      form.value.company_name = company.name
    }
    if (company.address && !form.value.address) {
      form.value.address = company.address
    }
    if (company.city) {
      const match = cities.value.find(
        (city) => city.name.toLowerCase() === company.city.toLowerCase()
      )
      if (match) {
        form.value.city_id = match.id
      }
    }
    siretState.value = {
      checking: false,
      verified: true,
      message: company.verified ? company.name : t('siret.valid'),
      error: ''
    }
  } catch (err) {
    siretState.value = { checking: false, verified: false, message: '', error: err.message }
  }
}

async function submit() {
  error.value = ''
  try {
    const data = await api.post('/auth/register', form.value)
    setSession(data.token, data.user)
    router.push('/espace')
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div class="auth-page">
    <AuthHero />
    <div class="auth-form-side">
      <div class="auth-card">
        <h2>{{ t('auth.register') }}</h2>
        <p class="sub">{{ t('app.title') }}</p>
        <form @submit.prevent="submit">
          <div class="field">
            <label>{{ t('auth.fullName') }}</label>
            <input v-model="form.full_name" required />
          </div>
          <div class="field">
            <label>{{ t('common.email') }}</label>
            <input v-model="form.email" type="email" required />
          </div>
          <div class="field">
            <label>{{ t('auth.password') }}</label>
            <input v-model="form.password" type="password" required />
          </div>
          <div class="field">
            <label>{{ t('auth.role') }}</label>
            <select v-model="form.role">
              <option value="member">{{ t('roles.member') }}</option>
              <option value="merchant">{{ t('roles.merchant') }}</option>
              <option value="volunteer">{{ t('roles.volunteer') }}</option>
            </select>
          </div>
          <template v-if="isMerchant">
            <div class="field">
              <label>{{ t('siret.label') }} *</label>
              <div class="siret-row">
                <input v-model="form.siret" placeholder="35247171800010" maxlength="20" />
                <button type="button" class="btn secondary small" :disabled="siretState.checking" @click="verifySiret">
                  {{ siretState.checking ? t('siret.checking') : t('siret.verify') }}
                </button>
              </div>
              <small class="field-hint">{{ t('siret.hint') }}</small>
              <p v-if="siretState.error" class="error">{{ siretState.error }}</p>
              <p v-if="siretState.verified" class="success">✅ {{ siretState.message }}</p>
            </div>
            <div class="field">
              <label>{{ t('siret.company') }} *</label>
              <input v-model="form.company_name" required />
            </div>
          </template>
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
          <p v-if="error" class="error">{{ error }}</p>
          <button class="btn" type="submit">{{ t('auth.signUp') }}</button>
        </form>
        <div class="auth-links">
          <span>{{ t('auth.haveAccount') }} <router-link to="/login">{{ t('nav.login') }}</router-link></span>
        </div>
      </div>
    </div>
  </div>
</template>
