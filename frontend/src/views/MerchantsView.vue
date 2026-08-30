<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole } from '../services/api.js'

const { t } = useI18n()

const merchants = ref([])
const reminders = ref([])
const isAdmin = computed(() => hasRole('admin'))
const editing = ref(null)
const error = ref('')

const emptyForm = () => ({
  company_name: '', contact_name: '', email: '', phone: '', address: '',
  membership_start: '', membership_end: '', status: 'active'
})
const form = ref(emptyForm())

async function load() {
  merchants.value = await api.get('/merchants')
  reminders.value = await api.get('/merchants/reminders')
}

onMounted(load)

function startCreate() {
  editing.value = 'new'
  form.value = emptyForm()
}

function startEdit(merchant) {
  editing.value = merchant.id
  form.value = { ...merchant }
}

async function save() {
  error.value = ''
  try {
    if (editing.value === 'new') {
      await api.post('/merchants', form.value)
    } else {
      await api.put(`/merchants/${editing.value}`, form.value)
    }
    editing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  await api.delete(`/merchants/${id}`)
  await load()
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('merchants.title') }}</h1>
      <p class="page-subtitle">{{ t('merchants.subtitle') }}</p>
    </div>

    <div v-if="reminders.length" class="card">
      <div class="card-title">🔔 {{ t('merchants.reminders') }}</div>
      <div class="table-wrap">
      <table>
        <thead>
          <tr><th>{{ t('merchants.company') }}</th><th>{{ t('common.email') }}</th><th>{{ t('merchants.membershipEnd') }}</th><th>{{ t('merchants.daysLeft') }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="r in reminders" :key="r.merchant_id">
            <td>{{ r.company_name }}</td>
            <td>{{ r.email }}</td>
            <td>{{ r.membership_end }}</td>
            <td><span class="badge" :class="r.days_left < 0 ? 'expired' : 'pending'">{{ r.days_left }}</span> {{ r.message }}</td>
          </tr>
        </tbody>
      </table>
      </div>
    </div>

    <div class="toolbar">
      <div></div>
      <button v-if="isAdmin" class="btn" @click="startCreate">{{ t('merchants.add') }}</button>
    </div>

    <div v-if="editing !== null" class="card">
      <div class="form-row">
        <div><label>{{ t('merchants.company') }}</label><input v-model="form.company_name" /></div>
        <div><label>{{ t('merchants.contact') }}</label><input v-model="form.contact_name" /></div>
        <div><label>{{ t('common.email') }}</label><input v-model="form.email" /></div>
      </div>
      <div class="form-row">
        <div><label>{{ t('common.phone') }}</label><input v-model="form.phone" /></div>
        <div><label>{{ t('common.address') }}</label><input v-model="form.address" /></div>
        <div><label>{{ t('common.status') }}</label>
          <select v-model="form.status"><option value="active">active</option><option value="inactive">inactive</option></select>
        </div>
      </div>
      <div class="form-row">
        <div><label>{{ t('merchants.membershipStart') }}</label><input v-model="form.membership_start" type="date" /></div>
        <div><label>{{ t('merchants.membershipEnd') }}</label><input v-model="form.membership_end" type="date" /></div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="inline-actions">
        <button class="btn" @click="save">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="editing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div class="card">
      <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ t('merchants.company') }}</th><th>{{ t('merchants.contact') }}</th><th>{{ t('common.email') }}</th>
            <th>{{ t('merchants.membershipEnd') }}</th><th>{{ t('common.status') }}</th>
            <th v-if="isAdmin">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in merchants" :key="m.id">
            <td>{{ m.company_name }}</td>
            <td>{{ m.contact_name }}</td>
            <td>{{ m.email }}</td>
            <td>{{ m.membership_end }}</td>
            <td><span class="badge" :class="m.status">{{ m.status }}</span></td>
            <td v-if="isAdmin" class="inline-actions">
              <button class="btn small ghost" @click="startEdit(m)">{{ t('common.edit') }}</button>
              <button class="btn small danger" @click="remove(m.id)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!merchants.length" class="empty-row"><td colspan="6">{{ t('common.none') }}</td></tr>
        </tbody>
      </table>
      </div>
    </div>
  </div>
</template>
