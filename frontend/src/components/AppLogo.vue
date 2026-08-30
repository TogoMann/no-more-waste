<script setup>
defineProps({
  size: { type: Number, default: 40 },
  showText: { type: Boolean, default: true },
  tagline: { type: String, default: '' },
  light: { type: Boolean, default: false }
})
</script>

<template>
  <div class="app-logo" :class="{ light }">
    <svg :width="size" :height="size" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
      <defs>
        <linearGradient id="nmwLeaf" x1="6" y1="6" x2="42" y2="42" gradientUnits="userSpaceOnUse">
          <stop stop-color="#34d399" />
          <stop offset="1" stop-color="#0d9488" />
        </linearGradient>
      </defs>
      <path
        d="M24 3.5 8.7 11.2v11.9c0 9.6 6.5 18.6 15.3 21.4 8.8-2.8 15.3-11.8 15.3-21.4V11.2L24 3.5Z"
        fill="url(#nmwLeaf)"
      />
      <path
        d="M24 33.5c0-6.6 4.3-11.4 10.2-12.4-.5 6.9-4.7 11.6-10.2 12.4Z"
        fill="#ffffff"
        fill-opacity="0.95"
      />
      <path
        d="M23.4 33.5c0-5.6-3.6-9.7-8.6-10.6.4 5.9 4 9.9 8.6 10.6Z"
        fill="#ffffff"
        fill-opacity="0.7"
      />
      <path d="M24 34.5v-9.8" stroke="#ffffff" stroke-width="2" stroke-linecap="round" />
    </svg>
    <div v-if="showText" class="logo-text">
      <span class="logo-name">NO MORE WASTE</span>
      <span v-if="tagline" class="logo-tagline">{{ tagline }}</span>
    </div>
  </div>
</template>
