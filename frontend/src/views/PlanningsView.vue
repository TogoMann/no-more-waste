<script setup>
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { api, hasRole, downloadFile } from '../services/api.js'

const { t, locale } = useI18n()

const plannings = ref([])
const volunteers = ref([])
const selected = ref(null)
const isAdmin = computed(() => hasRole('admin'))
const showForm = ref(false)
const error = ref('')

const cursor = ref(new Date())
const editingId = ref(null)

const emptyForm = () => ({
  title: '', planning_date: '', description: '', location: '',
  start_time: '09:00', end_time: '17:00', max_participants: 10,
  slots: [{ volunteer_id: null, task: '', start_time: '', end_time: '' }]
})
const form = ref(emptyForm())

function isPastDate(value) {
  return new Date(value) < new Date(new Date().toDateString())
}

const localeCode = computed(() => (locale.value === 'en' ? 'en-GB' : 'fr-FR'))

const monthLabel = computed(() =>
  cursor.value.toLocaleDateString(localeCode.value, { month: 'long', year: 'numeric' })
)

const weekdayLabels = computed(() => {
  const monday = new Date(2024, 0, 1)
  const labels = []
  for (let index = 0; index < 7; index += 1) {
    const day = new Date(monday)
    day.setDate(monday.getDate() + index)
    labels.push(day.toLocaleDateString(localeCode.value, { weekday: 'short' }))
  }
  return labels
})

function dateKey(year, month, day) {
  const monthPart = String(month + 1).padStart(2, '0')
  const dayPart = String(day).padStart(2, '0')
  return `${year}-${monthPart}-${dayPart}`
}

const todayKey = dateKey(new Date().getFullYear(), new Date().getMonth(), new Date().getDate())

const planningsByDate = computed(() => {
  const map = {}
  for (const planning of plannings.value) {
    if (!map[planning.planning_date]) {
      map[planning.planning_date] = []
    }
    map[planning.planning_date].push(planning)
  }
  return map
})

const calendarCells = computed(() => {
  const year = cursor.value.getFullYear()
  const month = cursor.value.getMonth()
  const firstDay = new Date(year, month, 1)
  const offset = (firstDay.getDay() + 6) % 7
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const cells = []
  for (let index = 0; index < offset; index += 1) {
    cells.push({ empty: true, id: `e${index}` })
  }
  for (let day = 1; day <= daysInMonth; day += 1) {
    const key = dateKey(year, month, day)
    cells.push({ empty: false, id: key, day, key, events: planningsByDate.value[key] || [] })
  }
  return cells
})

async function load() {
  plannings.value = await api.get('/plannings')
  volunteers.value = await api.get('/volunteers?status=approved')
}

onMounted(load)

function shiftMonth(delta) {
  cursor.value = new Date(cursor.value.getFullYear(), cursor.value.getMonth() + delta, 1)
}

function goToday() {
  cursor.value = new Date()
}

function addSlot() {
  form.value.slots.push({ volunteer_id: null, task: '', start_time: '', end_time: '' })
}

function removeSlot(index) {
  form.value.slots.splice(index, 1)
}

function openCreate(dateValue) {
  if (dateValue && isPastDate(dateValue)) {
    return
  }
  editingId.value = null
  form.value = emptyForm()
  if (dateValue) {
    form.value.planning_date = dateValue
  }
  showForm.value = true
  selected.value = null
}

function openEdit(planning) {
  editingId.value = planning.id
  form.value = {
    title: planning.title,
    planning_date: planning.planning_date,
    description: planning.description || '',
    location: planning.location || '',
    start_time: planning.start_time || '09:00',
    end_time: planning.end_time || '17:00',
    max_participants: planning.max_participants || 10,
    slots: planning.slots.length
      ? planning.slots.map((slot) => ({
          volunteer_id: slot.volunteer_id,
          task: slot.task,
          start_time: slot.start_time,
          end_time: slot.end_time
        }))
      : [{ volunteer_id: null, task: '', start_time: '', end_time: '' }]
  }
  showForm.value = true
  selected.value = null
}

async function submit() {
  error.value = ''
  try {
    const payload = {
      ...form.value,
      slots: form.value.slots.filter((slot) => slot.volunteer_id)
    }
    if (editingId.value) {
      await api.put(`/plannings/${editingId.value}`, payload)
    } else {
      await api.post('/plannings', payload)
    }
    showForm.value = false
    editingId.value = null
    form.value = emptyForm()
    await load()
  } catch (err) {
    error.value = err.message
  }
}

async function view(id) {
  selected.value = await api.get(`/plannings/${id}`)
  showForm.value = false
}

async function remove(id) {
  if (!confirm(t('common.confirmDelete'))) return
  await api.delete(`/plannings/${id}`)
  selected.value = null
  await load()
}

function downloadExcel(id) {
  downloadFile(`/plannings/${id}/excel`, `planning-${id}.xlsx`)
}
</script>

<template>
  <div>
    <div class="page-head">
      <h1 class="page-title">{{ t('plannings.title') }}</h1>
      <p class="page-subtitle">{{ t('plannings.subtitle') }}</p>
    </div>

    <div class="card">
      <div class="cal-toolbar">
        <div class="cal-nav-group">
          <button class="cal-nav-btn" @click="shiftMonth(-1)">‹</button>
          <button class="btn ghost small" @click="goToday">{{ t('plannings.today') }}</button>
          <button class="cal-nav-btn" @click="shiftMonth(1)">›</button>
        </div>
        <div class="cal-month-label">{{ monthLabel }}</div>
        <button v-if="isAdmin" class="btn" @click="openCreate('')">＋ {{ t('plannings.add') }}</button>
        <div v-else></div>
      </div>

      <div class="cal-grid">
        <div v-for="label in weekdayLabels" :key="label" class="cal-weekday">{{ label }}</div>
        <div
          v-for="cell in calendarCells"
          :key="cell.id"
          class="cal-cell"
          :class="{
            empty: cell.empty,
            'in-month': !cell.empty,
            today: cell.key === todayKey,
            past: !cell.empty && isPastDate(cell.key),
            clickable: !cell.empty && isAdmin && !isPastDate(cell.key)
          }"
          @click="!cell.empty && isAdmin ? openCreate(cell.key) : null"
        >
          <template v-if="!cell.empty">
            <span class="cal-daynum">{{ cell.day }}</span>
            <button
              v-for="event in cell.events"
              :key="event.id"
              class="cal-event"
              @click.stop="view(event.id)"
            >
              {{ event.title }}
              <span class="cal-event-count">{{ event.participant_count }}/{{ event.max_participants }}</span>
            </button>
          </template>
        </div>
      </div>
    </div>

    <div v-if="showForm" class="card">
      <div class="card-title">{{ editingId ? '✏️ ' + t('common.edit') : '＋ ' + t('plannings.add') }}</div>
      <div class="form-row">
        <div><label>{{ t('plannings.planningTitle') }}</label><input v-model="form.title" /></div>
        <div><label>{{ t('common.date') }}</label><input v-model="form.planning_date" type="date" /></div>
        <div><label>{{ t('plannings.start') }}</label><input v-model="form.start_time" type="time" /></div>
        <div><label>{{ t('plannings.end') }}</label><input v-model="form.end_time" type="time" /></div>
        <div>
          <label>{{ t('plannings.maxParticipants') }}</label>
          <input v-model.number="form.max_participants" type="number" min="1" />
        </div>
      </div>
      <div class="form-row">
        <div><label>{{ t('plannings.location') }}</label><input v-model="form.location" /></div>
        <div><label>{{ t('products.description') }}</label><input v-model="form.description" /></div>
      </div>
      <label>{{ t('plannings.slots') }}</label>
      <div v-for="(slot, index) in form.slots" :key="index" class="form-row">
        <div>
          <select v-model.number="slot.volunteer_id">
            <option :value="null">--</option>
            <option v-for="v in volunteers" :key="v.id" :value="v.id">{{ v.full_name }}</option>
          </select>
        </div>
        <div><input v-model="slot.task" :placeholder="t('plannings.task')" /></div>
        <div><input v-model="slot.start_time" type="time" /></div>
        <div><input v-model="slot.end_time" type="time" /></div>
        <div><button class="btn ghost small" @click="removeSlot(index)">{{ t('common.delete') }}</button></div>
      </div>
      <button class="btn secondary small" @click="addSlot">{{ t('plannings.addSlot') }}</button>
      <p v-if="error" class="error">{{ error }}</p>
      <div style="margin-top:14px" class="inline-actions">
        <button class="btn" @click="submit">{{ editingId ? t('common.save') : t('common.create') }}</button>
        <button class="btn ghost" @click="showForm = false; editingId = null">{{ t('common.cancel') }}</button>
      </div>
    </div>

    <div v-if="selected" class="card">
      <div class="toolbar">
        <div class="card-title" style="margin-bottom:0">🗓️ {{ selected.title }} — {{ selected.planning_date }}</div>
        <div class="inline-actions">
          <button v-if="isAdmin && !isPastDate(selected.planning_date)" class="btn small" @click="openEdit(selected)">✏️ {{ t('common.edit') }}</button>
          <button class="btn small secondary" @click="downloadExcel(selected.id)">{{ t('plannings.excel') }}</button>
          <button v-if="isAdmin && !isPastDate(selected.planning_date)" class="btn small danger" @click="remove(selected.id)">{{ t('common.delete') }}</button>
          <button class="btn ghost small" @click="selected = null">{{ t('common.cancel') }}</button>
        </div>
      </div>

      <div class="detail-summary">
        <span class="badge" :class="isPastDate(selected.planning_date) ? 'inactive' : 'approved'">
          {{ isPastDate(selected.planning_date) ? t('events.passed') : t('events.upcoming') }}
        </span>
        <span>🕒 {{ selected.start_time }} – {{ selected.end_time }}</span>
        <span v-if="selected.location">📍 {{ selected.location }}</span>
        <span>👥 {{ selected.participant_count }}/{{ selected.max_participants }} {{ t('events.spotsTaken') }}</span>
      </div>
      <p v-if="selected.description" class="event-desc">{{ selected.description }}</p>

      <div v-if="selected.participants && selected.participants.length" style="margin-bottom:18px">
        <label>{{ t('plannings.participants') }}</label>
        <div class="skill-tags">
          <span v-for="participant in selected.participants" :key="participant.user_id" class="skill-tag">
            {{ participant.full_name }}
          </span>
        </div>
      </div>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th>{{ t('plannings.volunteer') }}</th><th>{{ t('plannings.task') }}</th><th>{{ t('plannings.start') }}</th><th>{{ t('plannings.end') }}</th></tr>
          </thead>
          <tbody>
            <tr v-for="slot in selected.slots" :key="slot.id">
              <td>{{ slot.volunteer_name }}</td><td>{{ slot.task }}</td><td>{{ slot.start_time }}</td><td>{{ slot.end_time }}</td>
            </tr>
            <tr v-if="!selected.slots.length" class="empty-row"><td colspan="4">{{ t('plannings.noSlots') }}</td></tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
