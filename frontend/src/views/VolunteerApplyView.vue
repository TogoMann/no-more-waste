<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '../services/api.js'
import AuthHero from '../components/AuthHero.vue'

const { t } = useI18n()

const form = ref({ full_name: '', email: '', phone: '', skill_ids: [] })
const skills = ref([])
const error = ref('')
const success = ref(false)

onMounted(async () => {
  skills.value = await api.get('/skills')
})

function toggleSkill(id) {
  const index = form.value.skill_ids.indexOf(id)
  if (index === -1) {
    form.value.skill_ids.push(id)
  } else {
    form.value.skill_ids.splice(index, 1)
  }
}

async function submit() {
  error.value = ''
  try {
    await api.post('/volunteers', form.value)
    success.value = true
    form.value = { full_name: '', email: '', phone: '', skill_ids: [] }
  } catch (err) {
    error.value = err.message
  }
}
</script>

<template>
  <div class="auth-page">
    <AuthHero />
    <div class="auth-form-side">
      <div class="auth-card" style="max-width:480px">
        <h2>{{ t('volunteers.apply') }}</h2>
        <p class="sub">{{ t('app.tagline') }}</p>
        <form @submit.prevent="submit">
          <div class="field">
            <label>{{ t('auth.fullName') }}</label>
            <input v-model="form.full_name" required />
          </div>
          <div class="field">
            <label>{{ t('common.email') }}</label>
            <input v-model="form.email" type="email" required />
          </div>
          <div class="field">
            <label>{{ t('common.phone') }}</label>
            <input v-model="form.phone" />
          </div>
          <div class="field">
            <label>{{ t('volunteers.skills') }}</label>
            <div class="checkbox-grid">
              <label v-for="skill in skills" :key="skill.id">
                <input type="checkbox" :checked="form.skill_ids.includes(skill.id)" @change="toggleSkill(skill.id)" />
                {{ skill.name }}
              </label>
            </div>
          </div>
          <p v-if="error" class="error">{{ error }}</p>
          <p v-if="success" class="success">{{ t('volunteers.applySuccess') }}</p>
          <button class="btn" type="submit">{{ t('common.save') }}</button>
        </form>
        <div class="auth-links">
          <router-link to="/login">← {{ t('nav.login') }}</router-link>
        </div>
      </div>
    </div>
  </div>
</template>
