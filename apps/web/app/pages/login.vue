<script setup lang="ts">
definePageMeta({ layout: 'auth', guest: true })
useHead({ title: 'Log in · Tasks' })
</script>

<template>
  <AuthCredentialsForm mode="login" />
</template>
