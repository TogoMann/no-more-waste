<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole } from '../services/api.js'

const { t } = useI18n()

const collections = ref([])
const merchants = ref([])
const drivers = ref([])
const containers = ref([])
const isAdmin = computed(() => hasRole('admin'))
const editing = ref(null)
const error = ref('')
const completing = ref(null)

const emptyForm = () => ({
  label: '', driver_id: null, scheduled_date: '',
  stops: [{ merchant_id: null }]
})
const form = ref(emptyForm())

const emptyProduct = () => ({
  name: '', category: '', barcode: '', quantity: 1,
  expiration_date: '', container_id: null, shelf_code: '', merchant_id: null
})
const collectedProducts = ref([emptyProduct()])

function isPastDate(value) {
  return new Date(value) < new Date(new Date().toDateString())
}

function statusLabel(status) {
  if (status === 'completed') return t('collections.completed')
  if (status === 'in_progress') return t('collections.inProgress')
  return t('collections.planned')
}

function statusClass(status) {
  if (status === 'completed') return 'approved'
  if (status === 'in_progress') return 'pending'
  return 'planned'
}

async function load() {
  collections.value = await api.get('/collections')
  merchants.value = await api.get('/merchants')
  containers.value = await api.get('/containers')
  const volunteers = await api.get('/volunteers?status=approved')
  drivers.value = volunteers
}

onMounted(load)

function startCreate() {
  editing.value = 'new'
  form.value = emptyForm()
}

async function startEdit(collection) {
  const full = await api.get(`/collections/${collection.id}`)
  editing.value = full.id
  form.value = {
    label: full.label,
    driver_id: full.driver_id,
    scheduled_date: full.scheduled_date,
    stops: full.stops.length ? full.stops.map((stop) => ({ merchant_id: stop.merchant_id })) : [{ merchant_id: null }]
  }
}

function addStop() {
  form.value.stops.push({ merchant_id: null })
}

function removeStop(index) {
  form.value.stops.splice(index, 1)
}

function moveStop(index, direction) {
  const target = index + direction
  if (target < 0 || target >= form.value.stops.length) return
  const stops = form.value.stops
  const temp = stops[index]
  stops[index] = stops[target]
  stops[target] = temp
}

async function save() {
  error.value = ''
  const payload = {
    label: form.value.label,
    driver_id: form.value.driver_id,
    scheduled_date: form.value.scheduled_date,
    stops: form.value.stops
      .filter((stop) => stop.merchant_id)
      .map((stop, index) => ({ merchant_id: stop.merchant_id, order_index: index + 1 }))
  }
  try {
    if (editing.value === 'new') {
      await api.post('/collections', payload)
    } else {
      await api.put(`/collections/${editing.value}`, payload)
    }
    editing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function setStatus(collection, status) {
  error.value = ''
  try {
    await api.patch(`/collections/${collection.id}/status`, { status })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  error.value = ''
  try {
    await api.delete(`/collections/${id}`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function openComplete(collection) {
  completing.value = await api.get(`/collections/${collection.id}`)
  collectedProducts.value = [emptyProduct()]
  editing.value = null
}

function addProductLine() {
  collectedProducts.value.push(emptyProduct())
}

function removeProductLine(index) {
  collectedProducts.value.splice(index, 1)
}

async function submitCompletion() {
  error.value = ''
  const products = collectedProducts.value.filter((product) => product.name && product.expiration_date)
  try {
    const result = await api.patch(`/collections/${completing.value.id}/complete`, { products })
    completing.value = null
    collectedProducts.value = [emptyProduct()]
    await load()
    error.value = ''
    alert(`${result.products_stored} ${t('collections.productsStored')}`)
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('collections.title') }}</h1>
      <p class="page-subtitle">{{ t('collections.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <div></div>
      <button class="btn" @click="startCreate">＋ {{ t('collections.add') }}</button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>

    <div v-if="editing !== null" class="card">
      <div class="card-title">🛻 {{ editing === 'new' ? t('collections.add') : t('common.edit') }}</div>
      <div class="form-row">
        <div><label>{{ t('collections.label') }}</label><input v-model="form.label" /></div>
        <div>
          <label>{{ t('collections.driver') }}</label>
          <select v-model.number="form.driver_id">
            <option :value="null">{{ t('collections.noDriver') }}</option>
            <option v-for="driver in drivers" :key="driver.id" :value="driver.id">{{ driver.full_name }}</option>
          </select>
        </div>
        <div><label>{{ t('collections.scheduledDate') }}</label><input v-model="form.scheduled_date" type="date" /></div>
      </div>

      <label>{{ t('collections.stops') }}</label>
      <div v-for="(stop, index) in form.stops" :key="index" class="stop-row">
        <span class="stop-order">{{ index + 1 }}</span>
        <select v-model.number="stop.merchant_id">
          <option :value="null">--</option>
          <option v-for="merchant in merchants" :key="merchant.id" :value="merchant.id">
            {{ merchant.company_name }} — {{ merchant.address }}
          </option>
        </select>
        <button class="cal-nav-btn" @click="moveStop(index, -1)">↑</button>
        <button class="cal-nav-btn" @click="moveStop(index, 1)">↓</button>
        <button class="btn ghost small" @click="removeStop(index)">✕</button>
      </div>
      <button class="btn secondary small" @click="addStop">{{ t('collections.addStop') }}</button>

      <div class="inline-actions" style="margin-top:14px">
        <button class="btn" @click="save">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="editing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="completing" class="card">
      <div class="toolbar">
        <div class="card-title" style="margin-bottom:0">📥 {{ t('collections.completeTitle') }} — {{ completing.label }}</div>
        <button class="btn ghost small" @click="completing = null">{{ t('common.cancel') }}</button>
      </div>
      <p class="page-subtitle" style="margin-bottom:16px">{{ t('collections.completeHint') }}</p>

      <div v-for="(product, index) in collectedProducts" :key="index" class="form-row collect-row">
        <div><label>{{ t('common.name') }}</label><input v-model="product.name" /></div>
        <div><label>{{ t('products.category') }}</label><input v-model="product.category" /></div>
        <div><label>{{ t('products.barcode') }}</label><input v-model="product.barcode" placeholder="scan" /></div>
        <div><label>{{ t('common.quantity') }}</label><input v-model.number="product.quantity" type="number" min="1" /></div>
        <div>
          <label>{{ t('expiry.label') }} *</label>
          <input v-model="product.expiration_date" type="date" required />
        </div>
        <div>
          <label>{{ t('containers.location') }}</label>
          <select v-model.number="product.container_id">
            <option :value="null">{{ t('containers.noContainer') }}</option>
            <option v-for="container in containers" :key="container.id" :value="container.id">
              {{ container.label }} — {{ container.city_name }}
            </option>
          </select>
        </div>
        <div><label>{{ t('containers.shelf') }}</label><input v-model="product.shelf_code" placeholder="A-01" /></div>
        <div style="display:flex;align-items:flex-end">
          <button class="btn ghost small" @click="removeProductLine(index)">✕</button>
        </div>
      </div>
      <button class="btn secondary small" @click="addProductLine">{{ t('collections.addProduct') }}</button>

      <div class="inline-actions" style="margin-top:16px">
        <button class="btn" @click="submitCompletion">✅ {{ t('collections.validate') }}</button>
      </div>
    </div>

    <div class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('collections.label') }}</th><th>{{ t('collections.driver') }}</th>
              <th>{{ t('collections.scheduledDate') }}</th><th>{{ t('collections.stops') }}</th>
              <th>{{ t('common.status') }}</th><th>{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="collection in collections" :key="collection.id">
              <td>{{ collection.label }}</td>
              <td>{{ collection.driver_name || t('collections.noDriver') }}</td>
              <td>{{ collection.scheduled_date }}</td>
              <td><strong>{{ collection.stop_count }}</strong></td>
              <td><span class="badge" :class="statusClass(collection.status)">{{ statusLabel(collection.status) }}</span></td>
              <td class="inline-actions">
                <button
                  v-if="collection.status === 'planned'"
                  class="btn small secondary"
                  @click="setStatus(collection, 'in_progress')"
                >{{ t('collections.start') }}</button>
                <button
                  v-if="collection.status !== 'completed'"
                  class="btn small accent"
                  @click="openComplete(collection)"
                >{{ t('collections.complete') }}</button>
                <button
                  v-if="collection.status !== 'completed' && !isPastDate(collection.scheduled_date)"
                  class="btn small ghost"
                  @click="startEdit(collection)"
                >{{ t('common.edit') }}</button>
                <button v-if="isAdmin" class="btn small danger" @click="remove(collection.id)">{{ t('common.delete') }}</button>
              </td>
            </tr>
            <tr v-if="!collections.length" class="empty-row"><td colspan="6">{{ t('collections.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
