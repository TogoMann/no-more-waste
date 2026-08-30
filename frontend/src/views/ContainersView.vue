<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole } from '../services/api.js'

const { t } = useI18n()

const containers = ref([])
const cities = ref([])
const cityFilter = ref('')
const isAdmin = computed(() => hasRole('admin'))
const editing = ref(null)
const error = ref('')
const content = ref(null)
const contentProducts = ref([])

const emptyForm = () => ({ city_id: null, label: '', address: '', capacity: 100, status: 'active' })
const form = ref(emptyForm())

async function load() {
  const query = cityFilter.value ? `?city_id=${cityFilter.value}` : ''
  containers.value = await api.get(`/containers${query}`)
}

onMounted(async () => {
  cities.value = await api.get('/cities')
  await load()
})

function startCreate() {
  editing.value = 'new'
  form.value = emptyForm()
  if (cities.value.length) {
    form.value.city_id = cities.value[0].id
  }
}

function startEdit(container) {
  editing.value = container.id
  form.value = {
    city_id: container.city_id,
    label: container.label,
    address: container.address,
    capacity: container.capacity,
    status: container.status
  }
}

async function save() {
  error.value = ''
  try {
    if (editing.value === 'new') {
      await api.post('/containers', form.value)
    } else {
      await api.put(`/containers/${editing.value}`, form.value)
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
    await api.delete(`/containers/${id}`)
    if (content.value && content.value.id === id) content.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function viewContent(container) {
  content.value = container
  contentProducts.value = await api.get(`/containers/${container.id}/products`)
}

function statusLabel(status) {
  if (status === 'full') return t('containers.statusFull')
  if (status === 'maintenance') return t('containers.statusMaintenance')
  return t('containers.statusActive')
}

function occupancyTone(value) {
  if (value >= 90) return 'danger'
  if (value >= 65) return 'warn'
  return ''
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('containers.title') }}</h1>
      <p class="page-subtitle">{{ t('containers.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <select v-model="cityFilter" style="max-width:240px" @change="load">
        <option value="">{{ t('containers.allCities') }}</option>
        <option v-for="city in cities" :key="city.id" :value="city.id">{{ city.name }}</option>
      </select>
      <button v-if="isAdmin" class="btn" @click="startCreate">＋ {{ t('containers.add') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="editing !== null" class="card">
      <div class="card-title">🏬 {{ editing === 'new' ? t('containers.add') : t('common.edit') }}</div>
      <div class="form-row">
        <div>
          <label>{{ t('containers.city') }}</label>
          <select v-model.number="form.city_id">
            <option v-for="city in cities" :key="city.id" :value="city.id">{{ city.name }}</option>
          </select>
        </div>
        <div><label>{{ t('containers.label') }}</label><input v-model="form.label" /></div>
        <div><label>{{ t('containers.capacity') }}</label><input v-model.number="form.capacity" type="number" min="1" /></div>
        <div>
          <label>{{ t('common.status') }}</label>
          <select v-model="form.status">
            <option value="active">{{ t('containers.statusActive') }}</option>
            <option value="full">{{ t('containers.statusFull') }}</option>
            <option value="maintenance">{{ t('containers.statusMaintenance') }}</option>
          </select>
        </div>
      </div>
      <div class="field"><label>{{ t('containers.address') }}</label><input v-model="form.address" /></div>
      <div class="inline-actions">
        <button class="btn" @click="save">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="editing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="content" class="card">
      <div class="toolbar">
        <div class="card-title" style="margin-bottom:0">
          📦 {{ t('containers.contentOf') }} {{ content.label }} — {{ content.city_name }}
        </div>
        <button class="btn ghost small" @click="content = null">{{ t('common.cancel') }}</button>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('containers.shelf') }}</th><th>{{ t('common.name') }}</th>
              <th>{{ t('products.category') }}</th><th>{{ t('products.barcode') }}</th><th>{{ t('common.quantity') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="product in contentProducts" :key="product.id">
              <td><span class="shelf-chip">{{ product.shelf_code || '—' }}</span></td>
              <td>
                <div class="prod-cell">
                  <img v-if="product.thumbnail" :src="product.thumbnail" class="prod-thumb" :alt="product.name" />
                  <div v-else class="prod-thumb-empty">📦</div>
                  <span>{{ product.name }}</span>
                </div>
              </td>
              <td>{{ product.category }}</td>
              <td>{{ product.barcode }}</td>
              <td><strong>{{ product.quantity }}</strong></td>
            </tr>
            <tr v-if="!contentProducts.length" class="empty-row"><td colspan="5">{{ t('common.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="container-grid">
      <article v-for="container in containers" :key="container.id" class="container-card">
        <div class="container-head">
          <div>
            <div class="container-label">{{ container.label }}</div>
            <div class="container-city">📍 {{ container.city_name }}</div>
          </div>
          <span class="badge" :class="container.status === 'active' ? 'active' : (container.status === 'full' ? 'rejected' : 'pending')">
            {{ statusLabel(container.status) }}
          </span>
        </div>
        <p class="container-address">{{ container.address }}</p>
        <div class="capacity-bar">
          <div class="capacity-fill" :class="occupancyTone(container.occupancy)"
            :style="{ width: Math.min(100, container.occupancy) + '%' }"></div>
        </div>
        <div class="container-stats">
          <span>{{ container.stored }} / {{ container.capacity }} · {{ container.occupancy }}%</span>
          <span>{{ container.products }} {{ t('containers.productsCount') }}</span>
        </div>
        <div class="inline-actions" style="margin-top:14px">
          <button class="btn small secondary" @click="viewContent(container)">{{ t('containers.viewContent') }}</button>
          <button v-if="isAdmin" class="btn small ghost" @click="startEdit(container)">{{ t('common.edit') }}</button>
          <button v-if="isAdmin" class="btn small danger" @click="remove(container.id)">{{ t('common.delete') }}</button>
        </div>
      </article>
      <p v-if="!containers.length" class="list-empty">{{ t('common.none') }}</p>
    </div>
  </div>
</template>
