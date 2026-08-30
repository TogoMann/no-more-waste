<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, authState } from '../../services/api.js'

const { t, locale } = useI18n()

const plannings = ref([])
const myPlannings = ref([])

const firstName = computed(() => (authState.user?.full_name || '').split(' ')[0])

function isFuture(date) {
  return new Date(date) >= new Date(new Date().toDateString())
}

const upcoming = computed(() =>
  plannings.value.filter((planning) => isFuture(planning.planning_date))
    .sort((a, b) => a.planning_date.localeCompare(b.planning_date))
)

const nextEvent = computed(() => upcoming.value[0] || null)

function formatDate(value) {
  return new Date(value).toLocaleDateString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    weekday: 'long', day: 'numeric', month: 'long'
  })
}

onMounted(async () => {
  plannings.value = await api.get('/plannings')
  myPlannings.value = await api.get('/plannings/mine')
})
</script>

<template>
  <div>
    <section class="dash-hero">
      <div class="hero-eyebrow">{{ t('app.tagline') }}</div>
      <h1>{{ t('events.welcomeTitle') }}, {{ firstName }} 👋</h1>
      <p>{{ t('events.welcomeLead') }}</p>
      <div class="hero-actions">
        <router-link class="hero-btn" to="/espace/planning">🗓️ {{ t('events.browse') }}</router-link>
        <router-link class="hero-btn outline" to="/espace/profil">👤 {{ t('events.myProfile') }}</router-link>
      </div>
      <div class="emoji-badge">🌍</div>
    </section>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-top"><div class="stat-icon">🗓️</div></div>
        <div class="stat-value">{{ upcoming.length }}</div>
        <div class="stat-label">{{ t('events.upcoming') }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-top"><div class="stat-icon blue">🤝</div></div>
        <div class="stat-value">{{ myPlannings.length }}</div>
        <div class="stat-label">{{ t('events.participations') }}</div>
      </div>
      <div class="stat-card" style="grid-column: span 2">
        <div class="stat-top"><div class="stat-icon amber">⭐</div></div>
        <div v-if="nextEvent">
          <div class="stat-value" style="font-size:19px">{{ nextEvent.title }}</div>
          <div class="stat-label">{{ t('events.nextEvent') }} · {{ formatDate(nextEvent.planning_date) }}</div>
        </div>
        <div v-else class="stat-label">{{ t('events.noEvents') }}</div>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <h3>🗓️ {{ t('events.title') }}</h3>
        <router-link class="panel-link" to="/espace/planning">{{ t('dashboard.viewAll') }} →</router-link>
      </div>
      <div class="list">
        <div v-for="event in upcoming.slice(0, 5)" :key="event.id" class="list-item">
          <div class="list-avatar">🌱</div>
          <div class="list-body">
            <div class="list-name">{{ event.title }}</div>
            <div class="list-meta">{{ formatDate(event.planning_date) }} · {{ event.location }}</div>
          </div>
          <span class="badge" :class="event.joined ? 'approved' : 'pending'">
            {{ event.participant_count }}/{{ event.max_participants }}
          </span>
        </div>
        <div v-if="!upcoming.length" class="list-empty">{{ t('events.noEvents') }}</div>
      </div>
    </div>
  </div>
</template>
