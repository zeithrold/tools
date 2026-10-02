<script setup lang="ts">
const numbers = [1, 2, 3]
</script>

<template>
  <p>{{ numbers.join(',') }} {{ [1, 2, 3].join(',') }}</p>
</template>
