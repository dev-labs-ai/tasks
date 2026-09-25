import { expect, test } from '@playwright/test'
import { addTask, logOut, newEmail, register, taskList } from './helpers'

test('main flow: register, create, complete, filter, delete, log out', async ({ page }) => {
  await register(page, newEmail('main'))
  await expect(page.getByText('No tasks yet. Add your first one above.')).toBeVisible()

  await addTask(page, 'Buy milk')
  await addTask(page, 'Walk the dog')
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Walk the dog', 'Buy milk'])

  await page.getByRole('checkbox', { name: 'Buy milk' }).click()
  await expect(page.getByRole('checkbox', { name: 'Buy milk' })).toBeChecked()

  await page.getByRole('tab', { name: 'Completed' }).click()
  await expect(page).toHaveURL('/?status=completed')
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Buy milk'])

  await page.getByRole('button', { name: 'Delete task: Buy milk' }).click()
  await expect(page.getByText('No completed tasks yet.')).toBeVisible()

  await page.getByRole('tab', { name: 'All' }).click()
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Walk the dog'])

  await logOut(page)
})

test('a reload keeps the user signed in and keeps the filter', async ({ page }) => {
  await register(page, newEmail('reload'))
  await addTask(page, 'Pending task')
  await addTask(page, 'Done task')
  await page.getByRole('checkbox', { name: 'Done task' }).click()
  await expect(page.getByRole('checkbox', { name: 'Done task' })).toBeChecked()
  await page.getByRole('tab', { name: 'Pending' }).click()
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Pending task'])

  await page.reload()

  await expect(page).toHaveURL('/?status=pending')
  await expect(page.getByRole('tab', { name: 'Pending' })).toHaveAttribute('aria-selected', 'true')
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Pending task'])
})

test('editing a title', async ({ page }) => {
  await register(page, newEmail('edit'))
  await addTask(page, 'Write the report')

  await page.getByRole('button', { name: 'Edit task: Write the report' }).click()
  const title = page.getByRole('textbox', { name: 'Title', exact: true })
  await expect(title).toHaveValue('Write the report')
  await title.fill('Write the final report')
  await title.press('Enter')

  await expect(taskList(page).getByRole('listitem')).toHaveText(['Write the final report'])
  await page.reload()
  await expect(taskList(page).getByRole('listitem')).toHaveText(['Write the final report'])
})

test("a second user cannot see the first user's tasks", async ({ browser }) => {
  const ana = await browser.newPage()
  await register(ana, newEmail('ana'))
  await addTask(ana, "Ana's secret task")

  const bob = await browser.newPage()
  await register(bob, newEmail('bob'))

  await expect(bob.getByText('No tasks yet. Add your first one above.')).toBeVisible()
  await expect(bob.getByText("Ana's secret task")).toHaveCount(0)
})

test('after logout, / redirects to /login', async ({ page }) => {
  await register(page, newEmail('logout'))
  await logOut(page)

  await page.goto('/')

  await expect(page).toHaveURL('/login')
  await expect(page.getByRole('heading', { name: 'Log in' })).toBeVisible()
})
