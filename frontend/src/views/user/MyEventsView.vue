<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../services/api.js'

const { t, locale } = useI18n()

const plannings = ref([])
const error = ref('')

function isPast(date) {
  return new Date(date) < new Date(new Date().toDateString())
}

const upcoming = computed(() =>
  plannings.value.filter((planning) => !isPast(planning.planning_date))
    .sort((a, b) => a.planning_date.localeCompare(b.planning_date))
)

const past = computed(() =>
  plannings.value.filter((planning) => isPast(planning.planning_date))
    .sort((a, b) => b.planning_date.localeCompare(a.planning_date))
)

function formatDate(value) {
  return new Date(value).toLocaleDateString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    weekday: 'long', day: 'numeric', month: 'long', year: 'numeric'
  })
}

async function load() {
  plannings.value = await api.get('/plannings/mine')
}

onMounted(load)

async function leave(event) {
  error.value = ''
  try {
    await api.delete(`/plannings/${event.id}/join`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('events.myTitle') }}</h1>
      <p class="page-subtitle">{{ t('events.mySubtitle') }}</p>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div class="card">
      <div class="card-title">🗓️ {{ t('events.upcoming') }}</div>
      <div class="list">
        <div v-for="event in upcoming" :key="event.id" class="list-item">
          <div class="list-avatar">🌱</div>
          <div class="list-body">
            <div class="list-name">{{ event.title }}</div>
            <div class="list-meta">
              {{ formatDate(event.planning_date) }} · {{ event.start_time }}–{{ event.end_time }}
              <template v-if="event.location"> · {{ event.location }}</template>
            </div>
          </div>
          <button class="btn small ghost" @click="leave(event)">{{ t('events.leave') }}</button>
        </div>
        <div v-if="!upcoming.length" class="list-empty">{{ t('events.noMyEvents') }}</div>
      </div>
    </div>

    <div v-if="past.length" class="card">
      <div class="card-title">✅ {{ t('events.past') }}</div>
      <div class="list">
        <div v-for="event in past" :key="event.id" class="list-item">
          <div class="list-avatar blue">✓</div>
          <div class="list-body">
            <div class="list-name">{{ event.title }}</div>
            <div class="list-meta">{{ formatDate(event.planning_date) }}</div>
          </div>
          <span class="badge inactive">{{ t('events.passed') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
