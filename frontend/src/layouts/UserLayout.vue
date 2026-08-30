<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { authState, clearSession, hasRole } from '../services/api.js'
import { setLocale } from '../i18n/index.js'
import AppLogo from '../components/AppLogo.vue'

const router = useRouter()
const { t, locale } = useI18n()

const isAdmin = computed(() => hasRole('admin'))
const isMerchant = computed(() => hasRole('merchant', 'admin'))
const isVolunteer = computed(() => hasRole('volunteer', 'admin'))

const initials = computed(() => {
  const name = authState.user?.full_name || '?'
  return name.split(' ').map((part) => part.charAt(0)).slice(0, 2).join('').toUpperCase()
})

function logout() {
  clearSession()
  router.push('/login')
}
</script>

<template>
  <div class="user-shell">
    <header class="user-nav">
      <router-link to="/espace" class="user-nav-brand">
        <AppLogo :size="38" :tagline="t('app.tagline')" light />
      </router-link>
      <nav class="user-nav-links">
        <router-link to="/espace">{{ t('nav.home') }}</router-link>
        <router-link to="/espace/planning">{{ t('nav.events') }}</router-link>
        <router-link to="/espace/services">{{ t('nav.services') }}</router-link>
        <router-link v-if="isMerchant" to="/espace/dons">{{ t('nav.donations') }}</router-link>
        <router-link v-if="isVolunteer" to="/espace/mes-creations">{{ t('nav.myEvents2') }}</router-link>
        <router-link to="/espace/mes-evenements">{{ t('nav.myEvents') }}</router-link>
        <router-link to="/espace/profil">{{ t('nav.profile') }}</router-link>
      </nav>
      <div class="user-nav-actions">
        <div class="lang-switch">
          <button :class="{ active: locale === 'fr' }" @click="setLocale('fr')">FR</button>
          <button :class="{ active: locale === 'en' }" @click="setLocale('en')">EN</button>
        </div>
        <router-link v-if="isAdmin" class="btn small accent" to="/admin">{{ t('nav.adminSpace') }}</router-link>
        <div class="avatar">{{ initials }}</div>
        <button class="btn ghost small" @click="logout">{{ t('nav.logout') }}</button>
      </div>
    </header>
    <main class="user-content">
      <router-view />
    </main>
    <footer class="user-footer">© 2026 NO MORE WASTE — {{ t('app.tagline') }}</footer>
  </div>
</template>
