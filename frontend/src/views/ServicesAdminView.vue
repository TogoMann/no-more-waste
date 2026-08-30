<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../services/api.js'

const { t, locale } = useI18n()

const services = ref([])
const editing = ref(null)
const error = ref('')
const detail = ref(null)

const categories = ['cuisine', 'bricolage', 'electricite', 'plomberie', 'reparation', 'vehicule', 'gardiennage']

const emptyForm = () => ({
  title: '', category: 'cuisine', description: '', date: '', time: '10:00',
  location: '', max_capacity: 10, status: 'open'
})
const form = ref(emptyForm())

function isPast(dateTime) {
  return new Date(dateTime.replace(' ', 'T')) < new Date()
}

function formatDateTime(value) {
  const parsed = new Date(value.replace(' ', 'T'))
  return parsed.toLocaleDateString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    day: '2-digit', month: '2-digit', year: 'numeric'
  }) + ' ' + parsed.toLocaleTimeString(locale.value === 'en' ? 'en-GB' : 'fr-FR', {
    hour: '2-digit', minute: '2-digit'
  })
}

async function load() {
  services.value = await api.get('/services')
}

onMounted(load)

function startCreate() {
  editing.value = 'new'
  form.value = emptyForm()
}

function startEdit(service) {
  editing.value = service.id
  const parts = service.date_time.split(' ')
  form.value = {
    title: service.title,
    category: service.category,
    description: service.description,
    date: parts[0],
    time: (parts[1] || '10:00').slice(0, 5),
    location: service.location,
    max_capacity: service.max_capacity,
    status: service.status
  }
}

async function save() {
  error.value = ''
  const payload = {
    title: form.value.title,
    category: form.value.category,
    description: form.value.description,
    date_time: `${form.value.date} ${form.value.time}`,
    location: form.value.location,
    max_capacity: form.value.max_capacity,
    status: form.value.status
  }
  try {
    if (editing.value === 'new') {
      await api.post('/services', payload)
    } else {
      await api.put(`/services/${editing.value}`, payload)
    }
    editing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  error.value = ''
  try {
    await api.delete(`/services/${id}`)
    if (detail.value && detail.value.id === id) detail.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function viewDetail(id) {
  detail.value = await api.get(`/services/${id}`)
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('services.adminTitle') }}</h1>
      <p class="page-subtitle">{{ t('services.adminSubtitle') }}</p>
    </div>

    <div class="toolbar">
      <div></div>
      <button class="btn" @click="startCreate">＋ {{ t('services.add') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="editing !== null" class="card">
      <div class="card-title">🎓 {{ editing === 'new' ? t('services.add') : t('common.edit') }}</div>
      <div class="form-row">
        <div><label>{{ t('common.name') }}</label><input v-model="form.title" /></div>
        <div>
          <label>{{ t('services.category') }}</label>
          <select v-model="form.category">
            <option v-for="category in categories" :key="category" :value="category">
              {{ t('services.' + category) }}
            </option>
          </select>
        </div>
        <div><label>{{ t('common.date') }}</label><input v-model="form.date" type="date" /></div>
        <div><label>{{ t('plannings.start') }}</label><input v-model="form.time" type="time" /></div>
        <div><label>{{ t('services.capacity') }}</label><input v-model.number="form.max_capacity" type="number" min="1" /></div>
        <div>
          <label>{{ t('services.status') }}</label>
          <select v-model="form.status">
            <option value="open">{{ t('services.open') }}</option>
            <option value="closed">{{ t('services.closed') }}</option>
          </select>
        </div>
      </div>
      <div class="form-row">
        <div><label>{{ t('plannings.location') }}</label><input v-model="form.location" /></div>
        <div><label>{{ t('products.description') }}</label><input v-model="form.description" /></div>
      </div>
      <div class="inline-actions">
        <button class="btn" @click="save">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="editing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="detail" class="card">
      <div class="toolbar">
        <div class="card-title" style="margin-bottom:0">👥 {{ detail.title }} — {{ t('services.subscribers') }}</div>
        <button class="btn ghost small" @click="detail = null">{{ t('common.cancel') }}</button>
      </div>
      <div class="skill-tags">
        <span v-for="subscriber in detail.subscribers" :key="subscriber.user_id" class="skill-tag">
          {{ subscriber.full_name }}
        </span>
      </div>
      <p v-if="!detail.subscribers || !detail.subscribers.length" class="list-empty">{{ t('common.none') }}</p>
    </div>

    <div class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th><th>{{ t('services.category') }}</th>
              <th>{{ t('services.dateTime') }}</th><th>{{ t('plannings.location') }}</th>
              <th>{{ t('services.capacity') }}</th><th>{{ t('services.status') }}</th>
              <th>{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="service in services" :key="service.id">
              <td>{{ service.title }}</td>
              <td><span class="cat-tag">{{ t('services.' + service.category) }}</span></td>
              <td>{{ formatDateTime(service.date_time) }}</td>
              <td>{{ service.location }}</td>
              <td><strong>{{ service.subscriber_count }}/{{ service.max_capacity }}</strong></td>
              <td>
                <span class="badge" :class="isPast(service.date_time) ? 'inactive' : (service.status === 'open' ? 'active' : 'pending')">
                  {{ isPast(service.date_time) ? t('services.passed') : (service.status === 'open' ? t('services.open') : t('services.closed')) }}
                </span>
              </td>
              <td class="inline-actions">
                <button class="btn small secondary" @click="viewDetail(service.id)">{{ t('services.subscribers') }}</button>
                <button v-if="!isPast(service.date_time)" class="btn small ghost" @click="startEdit(service)">{{ t('common.edit') }}</button>
                <button class="btn small danger" @click="remove(service.id)">{{ t('common.delete') }}</button>
              </td>
            </tr>
            <tr v-if="!services.length" class="empty-row"><td colspan="7">{{ t('services.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
