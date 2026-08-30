<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api, setSession } from '../services/api.js'
import { setLocale } from '../i18n/index.js'
import AuthHero from '../components/AuthHero.vue'

const router = useRouter()
const { t, locale } = useI18n()

const email = ref('')
const password = ref('')
const error = ref('')

async function submit() {
  error.value = ''
  try {
    const data = await api.post('/auth/login', { email: email.value, password: password.value })
    setSession(data.token, data.user)
    router.push(data.user.role === 'admin' ? '/admin' : '/espace')
  } catch (err) {
    error.value = err.message
  }
}

function changeLanguage(code) {
  setLocale(code)
}
</script>

<template>
  <div class="auth-page">
    <AuthHero />
    <div class="auth-form-side">
      <div class="auth-card">
        <div style="display:flex;justify-content:flex-end;margin-bottom:18px">
          <div class="lang-switch">
            <button :class="{ active: locale === 'fr' }" @click="changeLanguage('fr')">FR</button>
            <button :class="{ active: locale === 'en' }" @click="changeLanguage('en')">EN</button>
          </div>
        </div>
        <h2>{{ t('auth.login') }}</h2>
        <p class="sub">{{ t('app.tagline') }}</p>
        <form @submit.prevent="submit">
          <div class="field">
            <label>{{ t('common.email') }}</label>
            <input v-model="email" type="email" required />
          </div>
          <div class="field">
            <label>{{ t('auth.password') }}</label>
            <input v-model="password" type="password" required />
          </div>
          <p v-if="error" class="error">{{ error }}</p>
          <button class="btn" type="submit">{{ t('auth.signIn') }}</button>
        </form>
        <div class="auth-links">
          <span>{{ t('auth.noAccount') }} <router-link to="/register">{{ t('nav.register') }}</router-link></span>
          <router-link to="/apply">{{ t('nav.apply') }} →</router-link>
        </div>
      </div>
    </div>
  </div>
</template>
