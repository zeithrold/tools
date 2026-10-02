<script setup lang="ts">
void Promise.resolve(1)
const label = 'Hello'
</script>

<template>
  <button>{{ label }}</button>
  <img src="photo.png">
  <div v-html="label" />
  <p>{{ [1, 2, 3].join(',') }}</p>
</template>
