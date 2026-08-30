<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole } from '../services/api.js'

const { t } = useI18n()

const volunteers = ref([])
const filter = ref('')
const isAdmin = computed(() => hasRole('admin'))

async function load() {
  const query = filter.value ? `?status=${filter.value}` : ''
  volunteers.value = await api.get(`/volunteers${query}`)
}

onMounted(load)

async function setStatus(id, status) {
  await api.patch(`/volunteers/${id}/status`, { status })
  await load()
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  await api.delete(`/volunteers/${id}`)
  await load()
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('volunteers.title') }}</h1>
      <p class="page-subtitle">{{ t('volunteers.subtitle') }}</p>
    </div>

    <div class="toolbar">
      <select v-model="filter" style="max-width:220px" @change="load">
        <option value="">{{ t('common.status') }}: --</option>
        <option value="pending">{{ t('volunteers.pending') }}</option>
        <option value="approved">{{ t('volunteers.approved') }}</option>
        <option value="rejected">{{ t('volunteers.rejected') }}</option>
      </select>
    </div>

    <div class="card">
      <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>{{ t('common.name') }}</th><th>{{ t('common.email') }}</th><th>{{ t('common.phone') }}</th>
            <th>{{ t('volunteers.skills') }}</th><th>{{ t('common.status') }}</th>
            <th v-if="isAdmin">{{ t('common.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="v in volunteers" :key="v.id">
            <td>{{ v.full_name }}</td>
            <td>{{ v.email }}</td>
            <td>{{ v.phone }}</td>
            <td>
              <div class="skill-tags">
                <span v-for="skill in v.skills" :key="skill.id" class="skill-tag">{{ skill.name }}</span>
              </div>
            </td>
            <td><span class="badge" :class="v.status">{{ v.status }}</span></td>
            <td v-if="isAdmin" class="inline-actions">
              <button v-if="v.status !== 'approved'" class="btn small" @click="setStatus(v.id, 'approved')">{{ t('volunteers.approve') }}</button>
              <button v-if="v.status !== 'rejected'" class="btn small ghost" @click="setStatus(v.id, 'rejected')">{{ t('volunteers.reject') }}</button>
              <button class="btn small danger" @click="remove(v.id)">{{ t('common.delete') }}</button>
            </td>
          </tr>
          <tr v-if="!volunteers.length" class="empty-row"><td colspan="6">{{ t('common.none') }}</td></tr>
        </tbody>
      </table>
      </div>
    </div>
  </div>
</template>
