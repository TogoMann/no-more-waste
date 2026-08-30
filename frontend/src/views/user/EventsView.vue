<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../services/api.js'

const { t, locale } = useI18n()

const plannings = ref([])
const error = ref('')
const busyId = ref(null)

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

function placesLeft(event) {
  return Math.max(0, event.max_participants - event.participant_count)
}

async function load() {
  plannings.value = await api.get('/plannings')
}

onMounted(load)

async function toggleJoin(event) {
  error.value = ''
  busyId.value = event.id
  try {
    if (event.joined) {
      await api.delete(`/plannings/${event.id}/join`)
    } else {
      await api.post(`/plannings/${event.id}/join`)
    }
    await load()
  } catch (err) {
    error.value = err.message
  } finally {
    busyId.value = null
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('events.title') }}</h1>
      <p class="page-subtitle">{{ t('events.subtitle') }}</p>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div class="event-grid">
      <article v-for="event in upcoming" :key="event.id" class="event-card" :class="{ joined: event.joined }">
        <div class="event-date">
          <span class="event-day">{{ new Date(event.planning_date).getDate() }}</span>
          <span class="event-month">
            {{ new Date(event.planning_date).toLocaleDateString(locale === 'en' ? 'en-GB' : 'fr-FR', { month: 'short' }) }}
          </span>
        </div>
        <div class="event-body">
          <h3>{{ event.title }}</h3>
          <p class="event-desc">{{ event.description }}</p>
          <div class="event-meta">
            <span>🕒 {{ event.start_time }} – {{ event.end_time }}</span>
            <span v-if="event.location">📍 {{ event.location }}</span>
          </div>
          <div class="capacity-bar">
            <div
              class="capacity-fill"
              :style="{ width: Math.min(100, (event.participant_count / event.max_participants) * 100) + '%' }"
            ></div>
          </div>
          <div class="event-foot">
            <span class="capacity-text">
              {{ event.participant_count }}/{{ event.max_participants }} {{ t('events.spotsTaken') }}
              <em v-if="placesLeft(event) > 0">· {{ placesLeft(event) }} {{ t('events.placesLeft') }}</em>
            </span>
            <button
              v-if="event.joined"
              class="btn small ghost"
              :disabled="busyId === event.id"
              @click="toggleJoin(event)"
            >{{ t('events.leave') }}</button>
            <button
              v-else-if="placesLeft(event) === 0"
              class="btn small ghost"
              disabled
            >{{ t('events.full') }}</button>
            <button
              v-else
              class="btn small"
              :disabled="busyId === event.id"
              @click="toggleJoin(event)"
            >{{ t('events.join') }}</button>
          </div>
        </div>
      </article>
      <p v-if="!upcoming.length" class="list-empty">{{ t('events.noEvents') }}</p>
    </div>

    <div v-if="past.length" style="margin-top:34px">
      <h2 class="section-title">{{ t('events.past') }}</h2>
      <div class="event-grid">
        <article v-for="event in past.slice(0, 6)" :key="event.id" class="event-card muted">
          <div class="event-date">
            <span class="event-day">{{ new Date(event.planning_date).getDate() }}</span>
            <span class="event-month">
              {{ new Date(event.planning_date).toLocaleDateString(locale === 'en' ? 'en-GB' : 'fr-FR', { month: 'short' }) }}
            </span>
          </div>
          <div class="event-body">
            <h3>{{ event.title }}</h3>
            <div class="event-meta"><span>{{ formatDate(event.planning_date) }}</span></div>
            <div class="event-foot">
              <span class="badge inactive">{{ t('events.passed') }}</span>
            </div>
          </div>
        </article>
      </div>
    </div>
  </div>
</template>
