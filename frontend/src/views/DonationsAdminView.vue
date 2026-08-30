<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../services/api.js'

const { t } = useI18n()

const donations = ref([])
const collections = ref([])
const drivers = ref([])
const statusFilter = ref('')
const error = ref('')
const reviewing = ref(null)

const reviewForm = ref({ scheduled_date: '', driver_id: null, collection_id: null, review_note: '' })

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
  return 'approved'
}

const pendingCount = computed(() => donations.value.filter((d) => d.status === 'pending').length)

async function load() {
  const query = statusFilter.value ? `?status=${statusFilter.value}` : ''
  donations.value = await api.get(`/donations${query}`)
  collections.value = await api.get('/collections')
  drivers.value = await api.get('/volunteers?status=approved')
}

onMounted(load)

function openReview(donation) {
  reviewing.value = donation
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  reviewForm.value = {
    scheduled_date: donation.available_from || tomorrow.toISOString().slice(0, 10),
    driver_id: null,
    collection_id: null,
    review_note: ''
  }
}

async function approve() {
  error.value = ''
  try {
    await api.patch(`/donations/${reviewing.value.id}/review`, {
      status: 'approved',
      scheduled_date: reviewForm.value.scheduled_date,
      driver_id: reviewForm.value.driver_id,
      collection_id: reviewForm.value.collection_id,
      review_note: reviewForm.value.review_note
    })
    reviewing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function reject(donation, note) {
  error.value = ''
  try {
    await api.patch(`/donations/${donation.id}/review`, { status: 'rejected', review_note: note || '' })
    reviewing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

const openCollections = computed(() => collections.value.filter((c) => c.status !== 'completed'))
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('donations.adminTitle') }}</h1>
      <p class="page-subtitle">{{ t('donations.adminSubtitle') }}</p>
    </div>

    <div class="toolbar">
      <select v-model="statusFilter" style="max-width:240px" @change="load">
        <option value="">{{ t('common.status') }}: —</option>
        <option value="pending">{{ t('donations.pending') }}</option>
        <option value="scheduled">{{ t('donations.scheduled') }}</option>
        <option value="collected">{{ t('donations.collected') }}</option>
        <option value="rejected">{{ t('donations.rejected') }}</option>
      </select>
      <span v-if="pendingCount" class="stat-chip warn">{{ pendingCount }} {{ t('donations.pending') }}</span>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="reviewing" class="card">
      <div class="card-title">✅ {{ t('donations.approve') }} — {{ reviewing.title }}</div>
      <div class="form-row">
        <div><label>{{ t('donations.scheduledDate') }}</label><input v-model="reviewForm.scheduled_date" type="date" /></div>
        <div>
          <label>{{ t('donations.assignDriver') }}</label>
          <select v-model.number="reviewForm.driver_id">
            <option :value="null">—</option>
            <option v-for="driver in drivers" :key="driver.id" :value="driver.id">{{ driver.full_name }}</option>
          </select>
        </div>
        <div>
          <label>{{ t('donations.attachTo') }}</label>
          <select v-model.number="reviewForm.collection_id">
            <option :value="null">{{ t('donations.newCollection') }}</option>
            <option v-for="collection in openCollections" :key="collection.id" :value="collection.id">
              {{ collection.label }} — {{ collection.scheduled_date }}
            </option>
          </select>
        </div>
      </div>
      <div class="field">
        <label>{{ t('donations.reviewNote') }}</label>
        <input v-model="reviewForm.review_note" />
      </div>
      <div class="inline-actions">
        <button class="btn" @click="approve">✅ {{ t('donations.approve') }}</button>
        <button class="btn danger" @click="reject(reviewing, reviewForm.review_note)">{{ t('donations.reject') }}</button>
        <button class="btn ghost" @click="reviewing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('donations.donor') }}</th><th>{{ t('common.name') }}</th>
              <th>{{ t('donations.type') }}</th><th>{{ t('donations.quantity') }}</th>
              <th>{{ t('expiry.label') }}</th><th>{{ t('common.status') }}</th><th>{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="donation in donations" :key="donation.id">
              <td>
                <div>{{ donation.company_name || donation.donor_name }}</div>
                <div class="prod-desc">{{ donation.pickup_address }}</div>
              </td>
              <td>
                <div>{{ donation.title }}</div>
                <div class="prod-desc">{{ donation.description }}</div>
              </td>
              <td><span class="cat-tag">{{ t('donations.' + donation.donation_type) }}</span></td>
              <td><strong>{{ donation.quantity }}</strong></td>
              <td>{{ donation.expiration_date || '—' }}</td>
              <td>
                <span class="badge" :class="statusClass(donation.status)">{{ t(statusLabels[donation.status]) }}</span>
                <div v-if="donation.collection_date" class="prod-desc">🚚 {{ donation.collection_date }}</div>
              </td>
              <td class="inline-actions">
                <button v-if="donation.status === 'pending'" class="btn small" @click="openReview(donation)">
                  {{ t('donations.approve') }}
                </button>
                <button v-if="donation.status === 'pending'" class="btn small danger" @click="reject(donation, '')">
                  {{ t('donations.reject') }}
                </button>
              </td>
            </tr>
            <tr v-if="!donations.length" class="empty-row"><td colspan="7">{{ t('donations.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
