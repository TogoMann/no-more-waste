<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../services/api.js'

const { t } = useI18n()

const events = ref([])
const statusFilter = ref('pending')
const error = ref('')
const notes = ref({})

function statusClass(status) {
  if (status === 'approved') return 'approved'
  if (status === 'rejected') return 'rejected'
  return 'pending'
}

async function load() {
  const query = statusFilter.value ? `?approval_status=${statusFilter.value}` : ''
  events.value = await api.get(`/plannings${query}`)
}

onMounted(load)

async function review(event, status) {
  error.value = ''
  try {
    await api.patch(`/plannings/${event.id}/review`, { status, review_note: notes.value[event.id] || '' })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

const pendingCount = computed(() => events.value.filter((e) => e.approval_status === 'pending').length)
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('myEventsAdmin.adminTitle') }}</h1>
      <p class="page-subtitle">{{ t('myEventsAdmin.adminSubtitle') }}</p>
    </div>

    <div class="toolbar">
      <select v-model="statusFilter" style="max-width:240px" @change="load">
        <option value="pending">{{ t('myEventsAdmin.pending') }}</option>
        <option value="approved">{{ t('myEventsAdmin.approved') }}</option>
        <option value="rejected">{{ t('myEventsAdmin.rejected') }}</option>
        <option value="">{{ t('common.status') }}: —</option>
      </select>
      <span v-if="statusFilter === 'pending' && pendingCount" class="stat-chip warn">
        {{ pendingCount }} {{ t('myEventsAdmin.pending') }}
      </span>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

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
            <span v-if="event.creator_name">👤 {{ t('myEventsAdmin.proposedBy') }} {{ event.creator_name }}</span>
          </div>
          <div v-if="event.approval_status === 'pending'" class="field" style="margin-bottom:10px">
            <input v-model="notes[event.id]" :placeholder="t('donations.reviewNote')" />
          </div>
          <div class="event-foot">
            <span class="badge" :class="statusClass(event.approval_status)">
              {{ t('myEventsAdmin.' + event.approval_status) }}
            </span>
            <div v-if="event.approval_status === 'pending'" class="inline-actions">
              <button class="btn small" @click="review(event, 'approved')">{{ t('myEventsAdmin.approve') }}</button>
              <button class="btn small danger" @click="review(event, 'rejected')">{{ t('myEventsAdmin.reject') }}</button>
            </div>
          </div>
        </div>
      </article>
      <p v-if="!events.length" class="list-empty">{{ t('myEventsAdmin.noPending') }}</p>
    </div>
  </div>
</template>
