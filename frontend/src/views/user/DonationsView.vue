<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../../services/api.js'

const { t } = useI18n()

const donations = ref([])
const showForm = ref(false)
const error = ref('')

const emptyForm = () => ({
  title: '', donation_type: 'food', category: '', description: '',
  quantity: 1, expiration_date: '', pickup_address: '', available_from: ''
})
const form = ref(emptyForm())

const statusLabels = {
  pending: 'donations.pending',
  approved: 'donations.approved',
  rejected: 'donations.rejected',
  scheduled: 'donations.scheduled',
  collected: 'donations.collected'
}

function statusClass(status) {
  if (status === 'rejected') return 'rejected'
  if (status === 'pending') return 'pending'
  if (status === 'collected' || status === 'scheduled') return 'approved'
  return 'active'
}

async function load() {
  donations.value = await api.get('/donations/mine')
}

onMounted(load)

async function submit() {
  error.value = ''
  try {
    await api.post('/donations', form.value)
    form.value = emptyForm()
    showForm.value = false
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(donation) {
  if (!confirm(t('common.confirmDelete'))) return
  error.value = ''
  try {
    await api.delete(`/donations/${donation.id}`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('donations.title') }}</h1>
      <p class="page-subtitle">{{ t('donations.subtitle') }}</p>
    </div>

    <div class="info-banner">
      <span class="info-icon">📦</span>
      <p>{{ t('donations.intro') }}</p>
    </div>

    <div class="toolbar">
      <div></div>
      <button class="btn" @click="showForm = !showForm">＋ {{ t('donations.add') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="showForm" class="card">
      <div class="card-title">🎁 {{ t('donations.add') }}</div>
      <div class="form-row">
        <div><label>{{ t('common.name') }}</label><input v-model="form.title" /></div>
        <div>
          <label>{{ t('donations.type') }}</label>
          <select v-model="form.donation_type">
            <option value="food">{{ t('donations.food') }}</option>
            <option value="object">{{ t('donations.object') }}</option>
          </select>
        </div>
        <div><label>{{ t('donations.category') }}</label><input v-model="form.category" /></div>
        <div><label>{{ t('donations.quantity') }}</label><input v-model.number="form.quantity" type="number" min="1" /></div>
      </div>
      <div class="form-row">
        <div v-if="form.donation_type === 'food'">
          <label>{{ t('products.expirationDate') }} *</label>
          <input v-model="form.expiration_date" type="date" />
        </div>
        <div><label>{{ t('donations.availableFrom') }}</label><input v-model="form.available_from" type="date" /></div>
        <div><label>{{ t('donations.pickupAddress') }}</label><input v-model="form.pickup_address" /></div>
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
      <article v-for="donation in donations" :key="donation.id" class="event-card">
        <div class="service-icon">{{ donation.donation_type === 'food' ? '🥗' : '📦' }}</div>
        <div class="event-body">
          <span class="cat-tag">{{ t('donations.' + donation.donation_type) }}</span>
          <h3>{{ donation.title }}</h3>
          <p class="event-desc">{{ donation.description }}</p>
          <div class="event-meta">
            <span>📊 {{ donation.quantity }} · {{ donation.category }}</span>
            <span v-if="donation.expiration_date">⏰ {{ t('expiry.label') }} {{ donation.expiration_date }}</span>
            <span v-if="donation.collection_date">🚚 {{ t('donations.collectionPlanned') }} {{ donation.collection_date }}</span>
          </div>
          <p v-if="donation.review_note" class="review-note">💬 {{ donation.review_note }}</p>
          <div class="event-foot">
            <span class="badge" :class="statusClass(donation.status)">{{ t(statusLabels[donation.status]) }}</span>
            <button
              v-if="donation.status === 'pending' || donation.status === 'rejected'"
              class="btn small ghost"
              @click="remove(donation)"
            >{{ t('common.delete') }}</button>
          </div>
        </div>
      </article>
      <p v-if="!donations.length" class="list-empty">{{ t('donations.noneMine') }}</p>
    </div>
  </div>
</template>
