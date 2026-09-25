<script setup lang="ts">
import { toApiError, type ApiError } from '~/services/client'

// The login and register forms: same fields, different rules and call.
const props = defineProps<{ mode: 'login' | 'register' }>()

const { $api } = useNuxtApp()
const auth = useAuthStore()

const isRegister = computed(() => props.mode === 'register')
const email = ref('')
const password = ref('')
const errors = reactive<{ email?: string; password?: string }>({})
const formError = ref<string | null>(null)
const submitting = ref(false)

const emailId = useId()
const passwordId = useId()

/** Checks the fields; returns the normalized email when both are valid. */
function validate(): string | null {
  const checked = validateEmail(email.value)
  errors.email = checked.problem ?? undefined
  if (isRegister.value) {
    errors.password = validatePassword(password.value) ?? undefined
  } else {
    errors.password = password.value === '' ? 'Password is required' : undefined
  }
  return errors.email || errors.password ? null : checked.value
}

async function submit() {
  formError.value = null
  const normalizedEmail = validate()
  if (!normalizedEmail) return

  submitting.value = true
  try {
    const user = isRegister.value
      ? await $api.auth.register(normalizedEmail, password.value)
      : await $api.auth.login(normalizedEmail, password.value)
    auth.setUser(user)
    await navigateTo('/')
  } catch (error) {
    showError(toApiError(error))
  } finally {
    submitting.value = false
  }
}

// What the user typed stays in the fields, so they can fix it and retry.
function showError(error: ApiError) {
  switch (error.code) {
    case 'VALIDATION_FAILED':
      errors.email = error.fields.email
      errors.password = error.fields.password
      if (!errors.email && !errors.password) formError.value = error.message
      break
    case 'EMAIL_TAKEN':
      errors.email = 'This email is already registered'
      break
    case 'INVALID_CREDENTIALS':
      formError.value = 'Invalid email or password'
      break
    case 'TOO_MANY_ATTEMPTS':
      formError.value = `Too many login attempts. Try again in ${formatWait(error.retryAfter ?? 15 * 60)}.`
      break
    case 'NETWORK_ERROR':
      formError.value = error.message
      break
    default:
      formError.value = 'Something went wrong. Please try again.'
  }
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle>
        <h1 class="text-xl">
          {{ isRegister ? 'Create an account' : 'Log in' }}
        </h1>
      </CardTitle>
      <CardDescription>
        {{
          isRegister ? 'Keep your tasks in one private place.' : 'Welcome back.'
        }}
      </CardDescription>
    </CardHeader>
    <CardContent>
      <form class="grid gap-4" novalidate @submit.prevent="submit">
        <Alert v-if="formError" variant="destructive" data-testid="form-error">
          <AlertDescription>{{ formError }}</AlertDescription>
        </Alert>

        <div class="grid gap-1.5">
          <Label :for="emailId">Email</Label>
          <Input
            :id="emailId"
            v-model="email"
            type="email"
            autocomplete="email"
            :aria-invalid="errors.email ? true : undefined"
            :aria-describedby="errors.email ? `${emailId}-error` : undefined"
          />
          <p
            v-if="errors.email"
            :id="`${emailId}-error`"
            class="text-sm text-destructive"
          >
            {{ errors.email }}
          </p>
        </div>

        <div class="grid gap-1.5">
          <Label :for="passwordId">Password</Label>
          <Input
            :id="passwordId"
            v-model="password"
            type="password"
            :autocomplete="isRegister ? 'new-password' : 'current-password'"
            :aria-invalid="errors.password ? true : undefined"
            :aria-describedby="
              errors.password
                ? `${passwordId}-error`
                : isRegister
                  ? `${passwordId}-hint`
                  : undefined
            "
          />
          <p
            v-if="errors.password"
            :id="`${passwordId}-error`"
            class="text-sm text-destructive"
          >
            {{ errors.password }}
          </p>
          <p
            v-else-if="isRegister"
            :id="`${passwordId}-hint`"
            class="text-sm text-muted-foreground"
          >
            8 to 64 characters.
          </p>
        </div>

        <Button type="submit" :disabled="submitting">
          {{ isRegister ? 'Create account' : 'Log in' }}
        </Button>
      </form>
    </CardContent>
    <CardFooter class="justify-center text-sm text-muted-foreground">
      <template v-if="isRegister">
        Already have an account?
        <NuxtLink to="/login" class="ml-1 font-medium text-foreground underline"
          >Log in</NuxtLink
        >
      </template>
      <template v-else>
        New here?
        <NuxtLink
          to="/register"
          class="ml-1 font-medium text-foreground underline"
        >
          Create an account
        </NuxtLink>
      </template>
    </CardFooter>
  </Card>
</template>
