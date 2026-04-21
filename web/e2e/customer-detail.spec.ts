/**
 * E2E 测试 — 客户详情页 (CustomerDetailView)
 *
 * 覆盖：
 *  1. 页面加载后显示金额汇总卡
 *  2. 记账抽屉 — 打开、填写、保存
 *  3. 收款弹窗 — 打开、填写、保存
 *  4. 账目编辑 — 点击"改"按钮打开编辑模式
 *  5. 账目删除 — 确认对话框
 *  6. 日期筛选
 *  7. 返回按钮回到列表页
 */
import { test, expect, type Page } from '@playwright/test'

// 跳到第一个存在的客户详情页
async function gotoFirstCustomer(page: Page) {
  await page.goto('/')
  await expect(page.locator('.customer-card').first()).toBeVisible({ timeout: 8000 })
  await page.locator('.customer-card').first().click()
  await expect(page).toHaveURL(/\/customers\//, { timeout: 5000 })
  await expect(page.locator('.summary-card')).toBeVisible({ timeout: 8000 })
}

test.describe('客户详情页', () => {
  test('显示金额汇总卡（累计出货 / 累计收款 / 当前欠款）', async ({ page }) => {
    await gotoFirstCustomer(page)
    await expect(page.locator('.summary-label', { hasText: '累计出货' })).toBeVisible()
    await expect(page.locator('.summary-label', { hasText: '累计收款' })).toBeVisible()
    await expect(page.locator('.summary-label', { hasText: '当前欠款' })).toBeVisible()
  })

  test('记账抽屉 — 打开后包含商品表格', async ({ page }) => {
    await gotoFirstCustomer(page)
    await page.locator('button', { hasText: '记一笔账' }).click()
    await expect(page.locator('.el-drawer')).toBeVisible({ timeout: 4000 })
    // 出货单式表格：商品 / 数量 / 单价 / 金额
    await expect(page.locator('.el-drawer .col-name').first()).toBeVisible()
    await expect(page.locator('.el-drawer .col-qty').first()).toBeVisible()
    await expect(page.locator('.el-drawer .col-price').first()).toBeVisible()
    await page.locator('.el-drawer button', { hasText: '取消' }).click()
    await expect(page.locator('.el-drawer')).not.toBeVisible({ timeout: 4000 })
  })

  test('记账抽屉 — 填写商品×数量×单价并保存', async ({ page }) => {
    await gotoFirstCustomer(page)

    // 记录保存前汇总金额
    const totalBefore = await page.locator('.summary-value').first().textContent()

    await page.locator('button', { hasText: '记一笔账' }).click()
    await expect(page.locator('.el-drawer')).toBeVisible({ timeout: 4000 })

    const firstRow = page.locator('.el-drawer .table-row').nth(0)
    await firstRow.locator('.col-name input').fill('自动测试品')
    await firstRow.locator('.col-qty input').fill('2')
    await firstRow.locator('.col-price input').fill('10')

    await page.locator('.el-drawer button', { hasText: '记账' }).click()

    // 抽屉关闭
    await expect(page.locator('.el-drawer')).not.toBeVisible({ timeout: 6000 })
    // 汇总金额有变化
    const totalAfter = await page.locator('.summary-value').first().textContent()
    expect(totalAfter).not.toBe(totalBefore)
  })

  test('收款弹窗 — 打开后显示未结金额', async ({ page }) => {
    await gotoFirstCustomer(page)
    const collectBtn = page.locator('button', { hasText: '收款' })
    await collectBtn.click()
    await expect(page.locator('.el-dialog', { hasText: '收款' })).toBeVisible({ timeout: 4000 })
    await expect(page.locator('.pending-hint')).toBeVisible()
    await page.locator('.el-dialog button', { hasText: '取消' }).click()
    await expect(page.locator('.el-dialog')).not.toBeVisible({ timeout: 4000 })
  })

  test('日期筛选 — 选择日期后可正常显示', async ({ page }) => {
    await gotoFirstCustomer(page)
    // 选一个日期范围
    const picker = page.locator('.el-date-editor input').first()
    await picker.click()
    // 点击今天
    await page.locator('.el-date-table td.today').first().click().catch(() => {
      // 没有 today 就跳过，不失败
    })
    // 页面不崩溃即可
    await expect(page.locator('.summary-card')).toBeVisible()
  })

  test('账目行有"改"和"删"按钮', async ({ page }) => {
    await gotoFirstCustomer(page)
    const items = page.locator('.timeline-item')
    const count = await items.count()
    if (count === 0) {
      test.skip()
      return
    }
    await expect(items.first().locator('button', { hasText: '改' })).toBeVisible()
    await expect(items.first().locator('button', { hasText: '删' })).toBeVisible()
  })

  test('点击"改"打开编辑抽屉', async ({ page }) => {
    await gotoFirstCustomer(page)
    const entryItems = page.locator('.timeline-item .icon-entry')
    const entryCount = await entryItems.count()
    if (entryCount === 0) {
      test.skip()
      return
    }
    const firstItem = page.locator('.timeline-item').filter({ has: page.locator('.icon-entry') }).first()
    await firstItem.locator('button', { hasText: '改' }).click()
    await expect(page.locator('.el-drawer')).toBeVisible({ timeout: 4000 })
    await expect(page.locator('.el-drawer button', { hasText: '保存' })).toBeVisible()
    await page.locator('.el-drawer button', { hasText: '取消' }).click()
  })

  test('点击"删"弹出确认对话框', async ({ page }) => {
    await gotoFirstCustomer(page)
    const items = page.locator('.timeline-item')
    const count = await items.count()
    if (count === 0) {
      test.skip()
      return
    }
    await items.first().locator('button', { hasText: '删' }).click()
    // Element Plus 的 MessageBox 确认框
    await expect(page.locator('.el-message-box')).toBeVisible({ timeout: 4000 })
    // 取消，不真正删除
    await page.locator('.el-message-box button', { hasText: '取消' }).click()
    await expect(page.locator('.el-message-box')).not.toBeVisible({ timeout: 4000 })
  })

  test('返回按钮回到客户列表', async ({ page }) => {
    await gotoFirstCustomer(page)
    await page.locator('button', { hasText: '返回' }).click()
    await expect(page).toHaveURL('/', { timeout: 5000 })
  })
})
