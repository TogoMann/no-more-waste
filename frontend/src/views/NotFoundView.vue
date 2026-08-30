<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { isAuthenticated } from '../services/api.js'

const { t } = useI18n()
const homeTarget = computed(() => (isAuthenticated() ? '/dashboard' : '/login'))
</script>

<template>
  <div class="error-page">
    <div class="error-card">
      <div class="error-code">404</div>
      <div class="error-leaf">🌱</div>
      <h1>{{ t('errors.notFoundTitle') }}</h1>
      <p>{{ t('errors.notFoundText') }}</p>
      <router-link class="btn" :to="homeTarget">← {{ t('errors.backHome') }}</router-link>
    </div>
  </div>
</template>
