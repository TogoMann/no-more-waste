<script setup>
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="error-page">
    <div class="error-card">
      <div class="error-code">403</div>
      <div class="error-leaf">🔒</div>
      <h1>{{ t('errors.forbiddenTitle') }}</h1>
      <p>{{ t('errors.forbiddenText') }}</p>
      <router-link class="btn" to="/espace">← {{ t('errors.backHome') }}</router-link>
    </div>
  </div>
</template>
