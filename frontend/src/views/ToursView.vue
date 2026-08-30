<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole, downloadFile } from '../services/api.js'

const { t } = useI18n()

const tours = ref([])
const products = ref([])
const isStaff = computed(() => hasRole('admin', 'volunteer'))
const isAdmin = computed(() => hasRole('admin'))
const showForm = ref(false)
const error = ref('')

const emptyForm = () => ({
  label: '', driver_name: '', destination: '', scheduled_date: '',
  items: [{ product_id: null, quantity: 1 }]
})
const form = ref(emptyForm())

async function load() {
  tours.value = await api.get('/tours')
  products.value = await api.get('/products')
}

onMounted(load)

function addItem() {
  form.value.items.push({ product_id: null, quantity: 1 })
}

function removeItem(index) {
  form.value.items.splice(index, 1)
}

async function create() {
  error.value = ''
  try {
    const payload = {
      ...form.value,
      items: form.value.items.filter((item) => item.product_id)
    }
    await api.post('/tours', payload)
    showForm.value = false
    form.value = emptyForm()
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function markDelivered(id) {
  await api.patch(`/tours/${id}/status`, { status: 'delivered' })
  await load()
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  await api.delete(`/tours/${id}`)
  await load()
}

function downloadPdf(id) {
  downloadFile(`/tours/${id}/pdf`, `livraison-tournee-${id}.pdf`).then(load)
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('tours.title') }}</h1>
      <p class="page-subtitle">{{ t('tours.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <div></div>
      <button v-if="isStaff" class="btn" @click="showForm = !showForm">{{ t('tours.add') }}</button>
    </div>

    <div v-if="showForm" class="card">
      <div class="form-row">
        <div><label>{{ t('tours.label') }}</label><input v-model="form.label" /></div>
        <div><label>{{ t('tours.driver') }}</label><input v-model="form.driver_name" /></div>
        <div><label>{{ t('tours.destination') }}</label><input v-model="form.destination" /></div>
        <div><label>{{ t('tours.scheduledDate') }}</label><input v-model="form.scheduled_date" type="date" /></div>
      </div>
      <label>{{ t('tours.items') }}</label>
      <div v-for="(item, index) in form.items" :key="index" class="form-row">
        <div>
          <select v-model.number="item.product_id">
            <option :value="null">--</option>
            <option v-for="p in products" :key="p.id" :value="p.id">{{ p.name }} ({{ p.quantity }})</option>
          </select>
        </div>
        <div><input v-model.number="item.quantity" type="number" min="1" /></div>
        <div><button class="btn ghost small" @click="removeItem(index)">{{ t('common.delete') }}</button></div>
      </div>
      <button class="btn secondary small" @click="addItem">{{ t('tours.addItem') }}</button>
      <p v-if="error" class="error">{{ error }}</p>
      <div style="margin-top:12px">
        <button class="btn" @click="create">{{ t('common.create') }}</button>
      </div>
    </div>

    <div class="card">
      <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ t('tours.label') }}</th><th>{{ t('tours.destination') }}</th><th>{{ t('tours.driver') }}</th>
            <th>{{ t('tours.scheduledDate') }}</th><th>{{ t('common.status') }}</th><th>{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="tour in tours" :key="tour.id">
            <td>{{ tour.label }}</td>
            <td>{{ tour.destination }}</td>
            <td>{{ tour.driver_name }}</td>
            <td>{{ tour.scheduled_date }}</td>
            <td><span class="badge" :class="tour.status">{{ tour.status }}</span></td>
            <td class="inline-actions">
              <button class="btn small secondary" @click="downloadPdf(tour.id)">{{ t('tours.pdf') }}</button>
              <button v-if="isStaff && tour.status !== 'delivered'" class="btn small" @click="markDelivered(tour.id)">{{ t('tours.markDelivered') }}</button>
              <button v-if="isAdmin" class="btn small danger" @click="remove(tour.id)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!tours.length" class="empty-row"><td colspan="6">{{ t('common.none') }}</td></tr>
        </tbody>
      </table>
      </div>
    </div>
  </div>
</template>
