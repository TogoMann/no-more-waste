<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, authState } from '../services/api.js'

const { t } = useI18n()

const users = ref([])
const editing = ref(null)
const error = ref('')
const form = ref({ full_name: '', role: 'member', status: 'active' })

const currentUserId = computed(() => authState.user?.id)

function roleLabel(role) {
  return t(`roles.${role}`)
}

function initials(name) {
  return (name || '?').split(' ').map((part) => part.charAt(0)).slice(0, 2).join('').toUpperCase()
}

async function load() {
  users.value = await api.get('/users')
}

onMounted(load)

function startEdit(user) {
  editing.value = user.id
  form.value = { full_name: user.full_name, role: user.role, status: user.status }
}

async function save() {
  error.value = ''
  try {
    await api.put(`/users/${editing.value}`, form.value)
    editing.value = null
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function toggleBan(user) {
  error.value = ''
  const status = user.status === 'banned' ? 'active' : 'banned'
  try {
    await api.patch(`/users/${user.id}/status`, { status })
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function remove(user) {
  if (!confirm(t('common.confirmDelete'))) return
  error.value = ''
  try {
    await api.delete(`/users/${user.id}`)
    await load()
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('nav.users') }}</h1>
      <p class="page-subtitle">{{ t('users.subtitle') }}</p>
    </div>

    <div v-if="editing !== null" class="card">
      <div class="card-title">✏️ {{ t('users.editTitle') }}</div>
      <div class="form-row">
        <div><label>{{ t('auth.fullName') }}</label><input v-model="form.full_name" /></div>
        <div>
          <label>{{ t('auth.role') }}</label>
          <select v-model="form.role">
            <option value="admin">{{ t('roles.admin') }}</option>
            <option value="volunteer">{{ t('roles.volunteer') }}</option>
            <option value="merchant">{{ t('roles.merchant') }}</option>
            <option value="member">{{ t('roles.member') }}</option>
          </select>
        </div>
        <div>
          <label>{{ t('common.status') }}</label>
          <select v-model="form.status">
            <option value="active">{{ t('users.active') }}</option>
            <option value="banned">{{ t('users.banned') }}</option>
          </select>
        </div>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
      <div class="inline-actions">
        <button class="btn" @click="save">{{ t('common.save') }}</button>
        <button class="btn ghost" @click="editing = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <p v-if="error && editing === null" class="error">{{ error }}</p>

    <div class="card">
      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{{ t('common.name') }}</th><th>{{ t('common.email') }}</th>
              <th>{{ t('auth.role') }}</th><th>{{ t('common.status') }}</th><th>{{ t('common.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td>
                <div style="display:flex;align-items:center;gap:10px">
                  <div class="list-avatar" style="width:34px;height:34px;font-size:13px">{{ initials(user.full_name) }}</div>
                  <span>{{ user.full_name }}
                    <span v-if="user.id === currentUserId" style="color:var(--muted);font-size:12px">{{ t('users.selfHint') }}</span>
                  </span>
                </div>
              </td>
              <td>{{ user.email }}</td>
              <td><span class="badge active">{{ roleLabel(user.role) }}</span></td>
              <td><span class="badge" :class="user.status === 'banned' ? 'rejected' : 'active'">{{ user.status === 'banned' ? t('users.banned') : t('users.active') }}</span></td>
              <td class="inline-actions">
                <button class="btn small ghost" @click="startEdit(user)">{{ t('common.edit') }}</button>
                <button v-if="user.id !== currentUserId" class="btn small accent" @click="toggleBan(user)">
                  {{ user.status === 'banned' ? t('users.unban') : t('users.ban') }}
                </button>
                <button v-if="user.id !== currentUserId" class="btn small danger" @click="remove(user)">{{ t('common.delete') }}</button>
              </td>
            </tr>
            <tr v-if="!users.length" class="empty-row"><td colspan="5">{{ t('common.none') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
