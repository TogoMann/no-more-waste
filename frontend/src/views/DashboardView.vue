<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, authState, hasRole } from '../services/api.js'

const { t, locale } = useI18n()

const stats = ref(null)
const reminders = ref([])
const pendingVolunteers = ref([])
const tours = ref([])

const isStaff = computed(() => hasRole('admin', 'volunteer'))
const firstName = computed(() => (authState.user?.full_name || '').split(' ')[0])

const todayLabel = computed(() =>
  new Date().toLocaleDateString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    weekday: 'long', day: 'numeric', month: 'long'
  })
)

const deliveredCount = computed(() => tours.value.filter((tour) => tour.status === 'delivered').length)
const upcomingTours = computed(() =>
  tours.value.filter((tour) => tour.status !== 'delivered').slice(0, 5)
)

const expiringProducts = computed(() => (stats.value && stats.value.expiring_products) || [])

function expiryLabel(product) {
  if (product.days_to_expiry === null || product.days_to_expiry === undefined) return ''
  if (product.days_to_expiry < 0) return t('expiry.expired')
  if (product.days_to_expiry === 0) return t('expiry.today')
  return t('expiry.expiresIn', { days: product.days_to_expiry })
}

function expiryTone(product) {
  if (product.days_to_expiry === null || product.days_to_expiry === undefined) return 'amber'
  return product.days_to_expiry < 0 ? 'danger' : 'amber'
}

function initials(name) {
  return (name || '?').split(' ').map((part) => part.charAt(0)).slice(0, 2).join('').toUpperCase()
}

onMounted(async () => {
  stats.value = await api.get('/dashboard')
  try {
    reminders.value = await api.get('/merchants/reminders')
  } catch (error) {
    reminders.value = []
  }
  try {
    tours.value = await api.get('/tours')
  } catch (error) {
    tours.value = []
  }
  if (isStaff.value) {
    try {
      pendingVolunteers.value = await api.get('/volunteers?status=pending')
    } catch (error) {
      pendingVolunteers.value = []
    }
  }
})
</script>

<template>
  <div>
    <section class="dash-hero">
      <div class="hero-eyebrow">{{ todayLabel }}</div>
      <h1>{{ t('dashboard.greeting') }}, {{ firstName }} 👋</h1>
      <p>{{ t('dashboard.heroLead') }}</p>
      <div class="hero-actions">
        <router-link class="hero-btn" to="/admin/tours">🚚 {{ t('dashboard.actionTours') }}</router-link>
        <router-link class="hero-btn outline" to="/admin/products">📦 {{ t('dashboard.actionProducts') }}</router-link>
      </div>
      <div class="emoji-badge">🌍</div>
    </section>

    <div v-if="stats" class="stats-grid">
      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon">🏪</div>
          <span v-if="reminders.length" class="stat-chip danger">{{ reminders.length }} {{ t('dashboard.toRenew') }}</span>
        </div>
        <div class="stat-value">{{ stats.merchants }}</div>
        <div class="stat-label">{{ t('dashboard.merchants') }}</div>
      </div>

      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon blue">📦</div>
          <span class="stat-chip">{{ stats.total_stock }} {{ t('dashboard.unitsStock') }}</span>
        </div>
        <div class="stat-value">{{ stats.products }}</div>
        <div class="stat-label">{{ t('dashboard.products') }}</div>
      </div>

      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon amber">🚚</div>
          <span class="stat-chip">{{ deliveredCount }} {{ t('dashboard.deliveredCount') }}</span>
        </div>
        <div class="stat-value">{{ stats.tours }}</div>
        <div class="stat-label">{{ t('dashboard.tours') }}</div>
      </div>

      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon violet">🤝</div>
          <span v-if="stats.volunteers_pending" class="stat-chip warn">{{ stats.volunteers_pending }} {{ t('dashboard.pendingCount') }}</span>
        </div>
        <div class="stat-value">{{ stats.volunteers_active }}</div>
        <div class="stat-label">{{ t('dashboard.volunteersActive') }}</div>
      </div>
    </div>

    <section v-if="stats" class="panel waste-panel">
      <div class="panel-head">
        <h3>🚨 {{ t('expiry.widgetTitle') }}</h3>
        <router-link class="panel-link" to="/admin/products">{{ t('dashboard.viewAll') }} →</router-link>
      </div>
      <p class="page-subtitle" style="margin-top:-8px;margin-bottom:14px">
        {{ t('expiry.subtitle') }} ·
        <strong style="color:var(--danger)">{{ stats.expired_count }} {{ t('expiry.expiredCount') }}</strong> ·
        <strong style="color:var(--accent-ink)">{{ stats.expiring_count }} {{ t('expiry.critical') }}</strong>
      </p>
      <div class="list">
        <div v-for="product in expiringProducts.slice(0, 6)" :key="product.id" class="list-item waste-item">
          <div class="list-avatar" :class="expiryTone(product)">
            {{ product.days_to_expiry !== null && product.days_to_expiry < 0 ? '⛔' : '⏰' }}
          </div>
          <div class="list-body">
            <div class="list-name">{{ product.name }}</div>
            <div class="list-meta">
              {{ product.quantity }} · {{ product.expiration_date }}
              <template v-if="product.container_name"> · {{ product.container_name }} {{ product.shelf_code }}</template>
            </div>
          </div>
          <span class="badge" :class="product.days_to_expiry !== null && product.days_to_expiry < 0 ? 'rejected' : 'pending'">
            {{ expiryLabel(product) }}
          </span>
        </div>
        <div v-if="!expiringProducts.length" class="list-empty">{{ t('expiry.none') }}</div>
      </div>
    </section>

    <div class="panel-grid">
      <div class="panel">
        <div class="panel-head">
          <h3>🔔 {{ t('dashboard.remindersPanel') }}</h3>
          <router-link class="panel-link" to="/admin/merchants">{{ t('dashboard.viewAll') }} →</router-link>
        </div>
        <div class="list">
          <div v-for="item in reminders.slice(0, 5)" :key="item.merchant_id" class="list-item">
            <div class="list-avatar" :class="item.days_left < 0 ? 'danger' : 'amber'">🏪</div>
            <div class="list-body">
              <div class="list-name">{{ item.company_name }}</div>
              <div class="list-meta">
                {{ item.days_left < 0 ? t('dashboard.expired') : t('dashboard.expiresIn', { days: item.days_left }) }}
              </div>
            </div>
            <span class="badge" :class="item.days_left < 0 ? 'expired' : 'pending'">{{ item.membership_end }}</span>
          </div>
          <div v-if="!reminders.length" class="list-empty">{{ t('dashboard.nothing') }}</div>
        </div>
      </div>

      <div v-if="isStaff" class="panel">
        <div class="panel-head">
          <h3>⏳ {{ t('dashboard.pendingPanel') }}</h3>
          <router-link class="panel-link" to="/admin/volunteers">{{ t('dashboard.viewAll') }} →</router-link>
        </div>
        <div class="list">
          <div v-for="volunteer in pendingVolunteers.slice(0, 5)" :key="volunteer.id" class="list-item">
            <div class="list-avatar blue">{{ initials(volunteer.full_name) }}</div>
            <div class="list-body">
              <div class="list-name">{{ volunteer.full_name }}</div>
              <div class="list-meta">{{ volunteer.skills.map((skill) => skill.name).join(', ') || volunteer.email }}</div>
            </div>
            <span class="badge pending">{{ t('volunteers.pending') }}</span>
          </div>
          <div v-if="!pendingVolunteers.length" class="list-empty">{{ t('dashboard.nothing') }}</div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-head">
          <h3>🗺️ {{ t('dashboard.upcomingPanel') }}</h3>
          <router-link class="panel-link" to="/admin/tours">{{ t('dashboard.viewAll') }} →</router-link>
        </div>
        <div class="list">
          <div v-for="tour in upcomingTours" :key="tour.id" class="list-item">
            <div class="list-avatar">🚚</div>
            <div class="list-body">
              <div class="list-name">{{ tour.label }}</div>
              <div class="list-meta">{{ tour.destination }} · {{ tour.scheduled_date }}</div>
            </div>
            <span class="badge" :class="tour.status">{{ tour.status }}</span>
          </div>
          <div v-if="!upcomingTours.length" class="list-empty">{{ t('dashboard.nothing') }}</div>
        </div>
      </div>
    </div>

    <p v-if="!stats" class="loading-state">{{ t('common.loading') }}</p>
  </div>
</template>
