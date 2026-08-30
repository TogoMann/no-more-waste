<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { authState, clearSession } from '../services/api.js'
import { setLocale } from '../i18n/index.js'
import AppLogo from '../components/AppLogo.vue'

const router = useRouter()
const { t, locale } = useI18n()

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
  <div class="app-shell">
    <aside class="sidebar">
      <div class="sidebar-brand">
        <AppLogo :size="42" :tagline="t('nav.adminSpace')" light />
      </div>

      <div class="nav-section">{{ t('nav.groupGeneral') }}</div>
      <nav>
        <router-link class="nav-link" to="/admin"><span class="nav-icon">📊</span><span>{{ t('nav.dashboard') }}</span></router-link>
        <router-link class="nav-link" to="/admin/merchants"><span class="nav-icon">🏪</span><span>{{ t('nav.merchants') }}</span></router-link>
        <router-link class="nav-link" to="/admin/containers"><span class="nav-icon">🏬</span><span>{{ t('nav.containers') }}</span></router-link>
        <router-link class="nav-link" to="/admin/products"><span class="nav-icon">📦</span><span>{{ t('nav.products') }}</span></router-link>
        <router-link class="nav-link" to="/admin/donations"><span class="nav-icon">🎁</span><span>{{ t('nav.donationsAdmin') }}</span></router-link>
        <router-link class="nav-link" to="/admin/collections"><span class="nav-icon">🛻</span><span>{{ t('nav.collections') }}</span></router-link>
        <router-link class="nav-link" to="/admin/tours"><span class="nav-icon">🚚</span><span>{{ t('nav.tours') }}</span></router-link>
      </nav>

      <div class="nav-section">{{ t('nav.groupTeam') }}</div>
      <nav>
        <router-link class="nav-link" to="/admin/volunteers"><span class="nav-icon">🤝</span><span>{{ t('nav.volunteers') }}</span></router-link>
        <router-link class="nav-link" to="/admin/plannings"><span class="nav-icon">🗓️</span><span>{{ t('nav.plannings') }}</span></router-link>
        <router-link class="nav-link" to="/admin/event-review"><span class="nav-icon">✅</span><span>{{ t('nav.eventReview') }}</span></router-link>
        <router-link class="nav-link" to="/admin/services"><span class="nav-icon">🎓</span><span>{{ t('nav.services') }}</span></router-link>
        <router-link class="nav-link" to="/admin/users"><span class="nav-icon">👤</span><span>{{ t('nav.users') }}</span></router-link>
      </nav>

      <div class="sidebar-footer">
        <router-link class="nav-link" to="/espace"><span class="nav-icon">↩️</span><span>{{ t('nav.userSpace') }}</span></router-link>
      </div>
    </aside>

    <div class="main">
      <header class="topbar">
        <div class="user-chip">
          <div class="avatar">{{ initials }}</div>
          <div class="user-meta">
            <div class="user-name">{{ authState.user?.full_name }}</div>
            <div class="user-role">{{ t('nav.adminSpace') }}</div>
          </div>
        </div>
        <div class="topbar-actions">
          <div class="lang-switch">
            <button :class="{ active: locale === 'fr' }" @click="setLocale('fr')">FR</button>
            <button :class="{ active: locale === 'en' }" @click="setLocale('en')">EN</button>
          </div>
          <button class="btn ghost" @click="logout">{{ t('nav.logout') }}</button>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>
