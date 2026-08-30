<script setup>
import { useI18n } from 'vue-i18n'
import AppLogo from './AppLogo.vue'

const { t } = useI18n()
</script>

<template>
  <div class="auth-hero">
    <div class="hero-logo">
      <AppLogo :size="52" :tagline="t('app.tagline')" light />
    </div>
    <h1>{{ t('app.heroTitle') }}</h1>
    <p class="lead">{{ t('app.heroLead') }}</p>
    <ul class="hero-points">
      <li><span class="dot">🥖</span>{{ t('app.point1') }}</li>
      <li><span class="dot">🚚</span>{{ t('app.point2') }}</li>
      <li><span class="dot">🤝</span>{{ t('app.point3') }}</li>
    </ul>
  </div>
</template>
