<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../services/api.js'

const { t, locale } = useI18n()

const services = ref([])
const profile = ref(null)
const categoryFilter = ref('')
const error = ref('')
const busyId = ref(null)

const categories = ['cuisine', 'bricolage', 'electricite', 'plomberie', 'reparation', 'vehicule', 'gardiennage']

const categoryIcons = {
  cuisine: '🍳',
  bricolage: '🔨',
  electricite: '💡',
  plomberie: '🚰',
  reparation: '🔧',
  vehicule: '🚗',
  gardiennage: '🏠'
}

const duesValid = computed(() => profile.value && profile.value.membership_valid)

function isPast(dateTime) {
  return new Date(dateTime.replace(' ', 'T')) < new Date()
}

const visibleServices = computed(() =>
  services.value
    .filter((service) => !categoryFilter.value || service.category === categoryFilter.value)
    .filter((service) => !isPast(service.date_time))
)

function placesLeft(service) {
  return Math.max(0, service.max_capacity - service.subscriber_count)
}

function fillPercent(service) {
  return Math.min(100, (service.subscriber_count / service.max_capacity) * 100)
}

function fillTone(service) {
  const ratio = service.subscriber_count / service.max_capacity
  if (ratio >= 1) return 'danger'
  if (ratio >= 0.7) return 'warn'
  return ''
}

function formatDateTime(value) {
  const parsed = new Date(value.replace(' ', 'T'))
  return parsed.toLocaleDateString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    weekday: 'long', day: 'numeric', month: 'long'
  }) + ' · ' + parsed.toLocaleTimeString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    hour: '2-digit', minute: '2-digit'
  })
}

async function load() {
  services.value = await api.get('/services')
  profile.value = await api.get('/profile')
}

onMounted(load)

async function toggleSubscription(service) {
  error.value = ''
  busyId.value = service.id
  try {
    if (service.subscribed) {
      await api.delete(`/services/${service.id}/subscribe`)
    } else {
      await api.post(`/services/${service.id}/subscribe`)
    }
    await load()
  } catch (err) {
    error.value = err.message
  } finally {
    busyId.value = null
  }
}

async function payDues() {
  error.value = ''
  try {
    const session = await api.post('/profile/dues/checkout')
    window.location.href = session.url
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('services.title') }}</h1>
      <p class="page-subtitle">{{ t('services.subtitle') }}</p>
    </div>

    <div v-if="profile && !duesValid" class="dues-alert">
      <div class="dues-alert-icon">⚠️</div>
      <div class="dues-alert-body">
        <strong>{{ t('dues.expired') }}</strong>
        <p>{{ t('dues.warning') }}</p>
      </div>
      <button class="btn accent" @click="payDues">{{ t('dues.pay') }} — {{ t('dues.amount') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div class="category-filters">
      <button class="cat-chip" :class="{ active: categoryFilter === '' }" @click="categoryFilter = ''">
        {{ t('services.all') }}
      </button>
      <button
        v-for="category in categories"
        :key="category"
        class="cat-chip"
        :class="{ active: categoryFilter === category }"
        @click="categoryFilter = category"
      >
        {{ categoryIcons[category] }} {{ t('services.' + category) }}
      </button>
    </div>

    <div class="event-grid">
      <article
        v-for="service in visibleServices"
        :key="service.id"
        class="event-card service-card"
        :class="{ joined: service.subscribed, locked: !duesValid }"
      >
        <div class="service-icon">{{ categoryIcons[service.category] }}</div>
        <div class="event-body">
          <span class="cat-tag">{{ t('services.' + service.category) }}</span>
          <h3>{{ service.title }}</h3>
          <p class="event-desc">{{ service.description }}</p>
          <div class="event-meta">
            <span>🗓️ {{ formatDateTime(service.date_time) }}</span>
            <span v-if="service.location">📍 {{ service.location }}</span>
          </div>
          <div class="capacity-bar">
            <div class="capacity-fill" :class="fillTone(service)" :style="{ width: fillPercent(service) + '%' }"></div>
          </div>
          <div class="event-foot">
            <span class="capacity-text">
              {{ service.subscriber_count }}/{{ service.max_capacity }} {{ t('services.subscribed') }}
              <em v-if="placesLeft(service) > 0">· {{ placesLeft(service) }} {{ t('services.placesLeft') }}</em>
            </span>
            <button v-if="!duesValid" class="btn small ghost" disabled>{{ t('services.duesRequired') }}</button>
            <button
              v-else-if="service.subscribed"
              class="btn small ghost"
              :disabled="busyId === service.id"
              @click="toggleSubscription(service)"
            >{{ t('services.unsubscribe') }}</button>
            <button v-else-if="service.status !== 'open'" class="btn small ghost" disabled>
              {{ t('services.closed') }}
            </button>
            <button v-else-if="placesLeft(service) === 0" class="btn small ghost" disabled>
              {{ t('services.full') }}
            </button>
            <button
              v-else
              class="btn small"
              :disabled="busyId === service.id"
              @click="toggleSubscription(service)"
            >{{ t('services.subscribe') }}</button>
          </div>
        </div>
      </article>
      <p v-if="!visibleServices.length" class="list-empty">{{ t('services.none') }}</p>
    </div>
  </div>
</template>
