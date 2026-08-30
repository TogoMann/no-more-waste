<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole } from '../services/api.js'

const { t } = useI18n()

const products = ref([])
const search = ref('')
const isStaff = computed(() => hasRole('admin', 'volunteer'))
const isAdmin = computed(() => hasRole('admin'))
const showForm = ref(false)
const error = ref('')
const barcode = ref(null)
const stockTarget = ref(null)
const detail = ref(null)

const containers = ref([])
const containerFilter = ref('')
const form = ref({ name: '', category: '', description: '', quantity: 0, container_id: null, shelf_code: '', expiration_date: '', images: [] })
const stockForm = ref({ movement_type: 'in', quantity: 1, reason: '' })
const detailImages = ref([])

function resizeImage(file) {
  return new Promise((resolve) => {
    const reader = new FileReader()
    reader.onload = (event) => {
      const image = new Image()
      image.onload = () => {
        const maxSize = 720
        let { width, height } = image
        if (width > maxSize || height > maxSize) {
          const ratio = Math.min(maxSize / width, maxSize / height)
          width = Math.round(width * ratio)
          height = Math.round(height * ratio)
        }
        const canvas = document.createElement('canvas')
        canvas.width = width
        canvas.height = height
        canvas.getContext('2d').drawImage(image, 0, 0, width, height)
        resolve(canvas.toDataURL('image/jpeg', 0.75))
      }
      image.src = event.target.result
    }
    reader.readAsDataURL(file)
  })
}

async function handleFiles(fileList, target) {
  for (const file of Array.from(fileList)) {
    if (!file.type.startsWith('image/')) continue
    const dataUri = await resizeImage(file)
    target.push(dataUri)
  }
}

function onCreateFiles(event) {
  handleFiles(event.target.files, form.value.images)
  event.target.value = ''
}

function removeFormImage(index) {
  form.value.images.splice(index, 1)
}

async function load() {
  const params = new URLSearchParams()
  if (search.value) params.set('search', search.value)
  if (containerFilter.value) params.set('container_id', containerFilter.value)
  const query = params.toString() ? `?${params.toString()}` : ''
  products.value = await api.get(`/products${query}`)
}

onMounted(async () => {
  containers.value = await api.get('/containers')
  await load()
})

async function create() {
  error.value = ''
  try {
    await api.post('/products', form.value)
    showForm.value = false
    form.value = { name: '', category: '', description: '', quantity: 0, container_id: null, shelf_code: '', expiration_date: '', images: [] }
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  await api.delete(`/products/${id}`)
  if (detail.value && detail.value.id === id) detail.value = null
  await load()
}

async function showBarcode(id) {
  barcode.value = await api.get(`/products/${id}/barcode-image`)
}

function openStock(product) {
  stockTarget.value = product
  stockForm.value = { movement_type: 'in', quantity: 1, reason: '' }
}

async function submitStock() {
  error.value = ''
  try {
    await api.post(`/products/${stockTarget.value.id}/stock`, stockForm.value)
    stockTarget.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function openDetail(id) {
  detail.value = await api.get(`/products/${id}`)
  detailImages.value = []
  barcode.value = null
}

async function onDetailFiles(event) {
  await handleFiles(event.target.files, detailImages.value)
  event.target.value = ''
}

async function saveDetailImages() {
  if (!detailImages.value.length) return
  error.value = ''
  try {
    await api.put(`/products/${detail.value.id}`, {
      name: detail.value.name,
      category: detail.value.category,
      description: detail.value.description,
      merchant_id: detail.value.merchant_id,
      container_id: detail.value.container_id,
      shelf_code: detail.value.shelf_code,
      expiration_date: detail.value.expiration_date,
      images: detailImages.value
    })
    detailImages.value = []
    await openDetail(detail.value.id)
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function deleteImage(imageId) {
  await api.delete(`/products/${detail.value.id}/images/${imageId}`)
  await openDetail(detail.value.id)
  await load()
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('products.title') }}</h1>
      <p class="page-subtitle">{{ t('products.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <div style="display:flex;gap:12px;flex-wrap:wrap;flex:1">
        <input v-model="search" :placeholder="t('common.search')" style="max-width:320px" @input="load" />
        <select v-model="containerFilter" style="max-width:240px" @change="load">
          <option value="">{{ t('containers.title') }}: —</option>
          <option v-for="container in containers" :key="container.id" :value="container.id">
            {{ container.label }} ({{ container.city_name }})
          </option>
        </select>
      </div>
      <button v-if="isStaff" class="btn" @click="showForm = !showForm">＋ {{ t('products.add') }}</button>
    </div>

    <div v-if="showForm" class="card">
      <div class="card-title">＋ {{ t('products.add') }}</div>
      <div class="form-row">
        <div><label>{{ t('common.name') }}</label><input v-model="form.name" /></div>
        <div><label>{{ t('products.category') }}</label><input v-model="form.category" /></div>
        <div><label>{{ t('common.quantity') }}</label><input v-model.number="form.quantity" type="number" min="0" /></div>
      </div>
      <div class="form-row">
        <div>
          <label>{{ t('containers.location') }}</label>
          <select v-model.number="form.container_id">
            <option :value="null">{{ t('containers.noContainer') }}</option>
            <option v-for="container in containers" :key="container.id" :value="container.id">
              {{ container.label }} — {{ container.city_name }}
            </option>
          </select>
        </div>
        <div><label>{{ t('containers.shelf') }}</label><input v-model="form.shelf_code" placeholder="A-01" /></div>
        <div>
          <label>{{ t('products.expirationDate') }} *</label>
          <input v-model="form.expiration_date" type="date" required />
        </div>
      </div>
      <div class="field">
        <label>{{ t('products.description') }}</label>
        <textarea v-model="form.description" rows="3"></textarea>
      </div>
      <div class="field">
        <label>{{ t('products.images') }}</label>
        <label class="img-uploader">
          <span class="up-icon">🖼️</span>
          <span>{{ t('products.addImages') }}</span>
          <small>{{ t('products.imageHint') }}</small>
          <input type="file" accept="image/*" multiple @change="onCreateFiles" />
        </label>
        <div v-if="form.images.length" class="img-preview-grid">
          <div v-for="(image, index) in form.images" :key="index" class="img-preview">
            <img :src="image" alt="preview" />
            <button class="img-remove" @click="removeFormImage(index)">✕</button>
          </div>
        </div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="inline-actions">
        <button class="btn" @click="create">{{ t('common.create') }}</button>
        <button class="btn ghost" @click="showForm = false">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="barcode" class="card barcode-preview">
      <p>{{ barcode.barcode }}</p>
      <img :src="barcode.image" alt="barcode" />
      <div><button class="btn ghost" @click="barcode = null">{{ t('common.cancel') }}</button></div>
    </div>

    <div v-if="stockTarget" class="card">
      <div class="card-title">🔄 {{ t('products.move') }} — {{ stockTarget.name }}</div>
      <div class="form-row">
        <div><label>{{ t('common.status') }}</label>
          <select v-model="stockForm.movement_type">
            <option value="in">{{ t('products.stockIn') }}</option>
            <option value="out">{{ t('products.stockOut') }}</option>
          </select>
        </div>
        <div><label>{{ t('common.quantity') }}</label><input v-model.number="stockForm.quantity" type="number" min="1" /></div>
        <div><label>{{ t('products.reason') }}</label><input v-model="stockForm.reason" /></div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="inline-actions">
        <button class="btn" @click="submitStock">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="stockTarget = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="detail" class="card">
      <div class="toolbar">
        <div class="card-title" style="margin-bottom:0">📦 {{ detail.name }}</div>
        <button class="btn ghost small" @click="detail = null">{{ t('common.cancel') }}</button>
      </div>
      <div class="detail-grid">
        <div>
          <div class="gallery-grid" v-if="detail.images && detail.images.length">
            <div v-for="image in detail.images" :key="image.id" class="gallery-item">
              <img :src="image.image" :alt="detail.name" />
              <button v-if="isStaff" class="img-remove" @click="deleteImage(image.id)">✕</button>
            </div>
          </div>
          <p v-else class="list-empty">{{ t('products.noImages') }}</p>
          <div v-if="isStaff" style="margin-top:14px">
            <label class="img-uploader">
              <span class="up-icon">🖼️</span>
              <span>{{ t('products.addImages') }}</span>
              <input type="file" accept="image/*" multiple @change="onDetailFiles" />
            </label>
            <div v-if="detailImages.length" class="img-preview-grid">
              <div v-for="(image, index) in detailImages" :key="index" class="img-preview">
                <img :src="image" alt="preview" />
                <button class="img-remove" @click="detailImages.splice(index, 1)">✕</button>
              </div>
            </div>
            <div v-if="detailImages.length" style="margin-top:10px">
              <button class="btn small" @click="saveDetailImages">{{ t('common.save') }}</button>
            </div>
          </div>
        </div>
        <dl class="detail-meta">
          <dt>{{ t('products.category') }}</dt>
          <dd>{{ detail.category || '—' }}</dd>
          <dt>{{ t('products.barcode') }}</dt>
          <dd>{{ detail.barcode }}</dd>
          <dt>{{ t('common.quantity') }}</dt>
          <dd>{{ detail.quantity }}</dd>
          <dt>{{ t('expiry.label') }}</dt>
          <dd>{{ detail.expiration_date || '—' }}</dd>
          <dt>{{ t('containers.location') }}</dt>
          <dd>
            <template v-if="detail.container_name">
              <span class="shelf-chip">{{ detail.shelf_code || '—' }}</span>
              {{ detail.container_name }} · {{ detail.city_name }}
            </template>
            <template v-else>{{ t('containers.noContainer') }}</template>
          </dd>
          <dt>{{ t('products.description') }}</dt>
          <dd>{{ detail.description || '—' }}</dd>
        </dl>
      </div>
    </div>

    <div class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('products.image') }}</th><th>{{ t('common.name') }}</th><th>{{ t('products.category') }}</th>
              <th>{{ t('products.barcode') }}</th><th>{{ t('expiry.label') }}</th><th>{{ t('containers.location') }}</th>
              <th>{{ t('common.quantity') }}</th><th>{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in products" :key="p.id">
              <td>
                <img v-if="p.thumbnail" :src="p.thumbnail" class="prod-thumb" :alt="p.name" />
                <div v-else class="prod-thumb-empty">📦</div>
              </td>
              <td>
                <div>{{ p.name }}</div>
                <div v-if="p.description" class="prod-desc">{{ p.description }}</div>
              </td>
              <td>{{ p.category }}</td>
              <td>{{ p.barcode }}</td>
              <td>
                <span v-if="p.expiration_date" class="badge" :class="p.days_to_expiry < 0 ? 'rejected' : (p.days_to_expiry <= 3 ? 'pending' : 'active')">
                  {{ p.expiration_date }}
                </span>
                <span v-else style="color:var(--faint)">—</span>
              </td>
              <td>
                <div v-if="p.container_name">
                  <span class="shelf-chip">{{ p.shelf_code || '—' }}</span>
                  <div class="prod-desc">{{ p.container_name }} · {{ p.city_name }}</div>
                </div>
                <span v-else style="color:var(--faint)">{{ t('containers.noContainer') }}</span>
              </td>
              <td><strong>{{ p.quantity }}</strong></td>
              <td class="inline-actions">
                <button class="btn small secondary" @click="openDetail(p.id)">{{ t('products.details') }}</button>
                <button class="btn small ghost" @click="showBarcode(p.id)">{{ t('products.showBarcode') }}</button>
                <button v-if="isStaff" class="btn small accent" @click="openStock(p)">{{ t('products.move') }}</button>
                <button v-if="isAdmin" class="btn small danger" @click="remove(p.id)">{{ t('common.delete') }}</button>
              </td>
            </tr>
            <tr v-if="!products.length" class="empty-row"><td colspan="8">{{ t('common.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
