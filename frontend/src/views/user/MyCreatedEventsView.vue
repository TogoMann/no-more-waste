<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../services/api.js'

const { t } = useI18n()

const events = ref([])
const showForm = ref(false)
const error = ref('')

const eventTypes = ['brocante', 'collecte', 'atelier', 'mission', 'autre']

const emptyForm = () => ({
  title: '', event_type: 'brocante', planning_date: '', start_time: '09:00',
  end_time: '17:00', location: '', description: '', max_participants: 20
})
const form = ref(emptyForm())

function statusClass(status) {
  if (status === 'approved') return 'approved'
  if (status === 'rejected') return 'rejected'
  return 'pending'
}

async function load() {
  events.value = await api.get('/plannings/created')
}

onMounted(load)

async function submit() {
  error.value = ''
  try {
    await api.post('/events', form.value)
    form.value = emptyForm()
    showForm.value = false
    await load()
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('myEventsAdmin.title') }}</h1>
      <p class="page-subtitle">{{ t('myEventsAdmin.subtitle') }}</p>
    </div>

    <div class="info-banner">
      <span class="info-icon">🎪</span>
      <p>{{ t('myEventsAdmin.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <div></div>
      <button class="btn" @click="showForm = !showForm">＋ {{ t('myEventsAdmin.create') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="showForm" class="card">
      <div class="card-title">🎪 {{ t('myEventsAdmin.create') }}</div>
      <div class="form-row">
        <div><label>{{ t('common.name') }}</label><input v-model="form.title" /></div>
        <div>
          <label>{{ t('myEventsAdmin.eventType') }}</label>
          <select v-model="form.event_type">
            <option v-for="type in eventTypes" :key="type" :value="type">{{ t('myEventsAdmin.' + type) }}</option>
          </select>
        </div>
        <div><label>{{ t('common.date') }}</label><input v-model="form.planning_date" type="date" /></div>
        <div><label>{{ t('plannings.start') }}</label><input v-model="form.start_time" type="time" /></div>
        <div><label>{{ t('plannings.end') }}</label><input v-model="form.end_time" type="time" /></div>
        <div><label>{{ t('plannings.maxParticipants') }}</label><input v-model.number="form.max_participants" type="number" min="1" /></div>
      </div>
      <div class="form-row">
        <div><label>{{ t('plannings.location') }}</label><input v-model="form.location" /></div>
      </div>
      <div class="field">
        <label>{{ t('products.description') }}</label>
        <textarea v-model="form.description" rows="3"></textarea>
      </div>
      <div class="inline-actions">
        <button class="btn" @click="submit">{{ t('common.create') }}</button>
        <button class="btn ghost" @click="showForm = false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div class="event-grid">
      <article v-for="event in events" :key="event.id" class="event-card">
        <div class="event-date">
          <span class="event-day">{{ new Date(event.planning_date).getDate() }}</span>
          <span class="event-month">{{ new Date(event.planning_date).toLocaleDateString('fr-FR', { month: 'short' }) }}</span>
        </div>
        <div class="event-body">
          <span class="cat-tag">{{ t('myEventsAdmin.' + (event.event_type || 'mission')) }}</span>
          <h3>{{ event.title }}</h3>
          <p class="event-desc">{{ event.description }}</p>
          <div class="event-meta">
            <span>🕒 {{ event.start_time }} – {{ event.end_time }}</span>
            <span v-if="event.location">📍 {{ event.location }}</span>
            <span>👥 {{ event.participant_count }}/{{ event.max_participants }}</span>
          </div>
          <p v-if="event.review_note" class="review-note">💬 {{ t('myEventsAdmin.reviewNote') }}: {{ event.review_note }}</p>
          <div class="event-foot">
            <span class="badge" :class="statusClass(event.approval_status)">
              {{ t('myEventsAdmin.' + event.approval_status) }}
            </span>
          </div>
        </div>
      </article>
      <p v-if="!events.length" class="list-empty">{{ t('myEventsAdmin.none') }}</p>
    </div>
  </div>
</template>
