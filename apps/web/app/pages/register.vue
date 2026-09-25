<script setup lang="ts">
definePageMeta({ layout: 'auth', guest: true })
useHead({ title: 'Create an account · Tasks' })
</script>

<template>
  <AuthCredentialsForm mode="register" />
</template>
