import { expect, type Page } from '@playwright/test'

export const password = 'correct horse'

/** A unique email per call, so tests never share users. */
export function newEmail(label: string): string {
  return `${label}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}@example.com`
}

export async function register(page: Page, email: string) {
  await page.goto('/register')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password').fill(password)
  await page.getByRole('button', { name: 'Create account' }).click()
  await expect(page).toHaveURL('/')
  await expect(page.getByRole('heading', { name: 'My tasks' })).toBeVisible()
}

export async function addTask(page: Page, title: string) {
  await page.getByLabel('New task').fill(title)
  await page.getByLabel('New task').press('Enter')
  await expect(taskList(page).getByText(title, { exact: true })).toBeVisible()
}

export function taskList(page: Page) {
  return page.getByRole('list', { name: 'Tasks' })
}

export async function logOut(page: Page) {
  await page.getByRole('button', { name: 'Log out' }).click()
  await expect(page).toHaveURL('/login')
}
