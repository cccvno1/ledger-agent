/**
 * E2E 测试 — 客户列表页 (CustomersView)
 *
 * 覆盖：
 *  1. 页面加载后展示客户卡片
 *  2. 搜索框过滤结果
 *  3. 已结清区块的展开/收起
 *  4. 点击客户卡片跳转到详情页
 *  5. 新增客户弹窗 — 验证 + 成功创建
 */
import { test, expect } from '@playwright/test'

test.describe('客户列表页', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // 等待至少一个客户卡片出现（API 数据加载完成）
    await expect(page.locator('.customer-card').first()).toBeVisible({ timeout: 8000 })
  })

  test('页面加载后展示客户卡片', async ({ page }) => {
    const cards = page.locator('.customer-card')
    const count = await cards.count()
    expect(count).toBeGreaterThan(0)
  })

  test('搜索框可按姓名过滤客户', async ({ page }) => {
    // 先记录初始卡片数量
    const totalBefore = await page.locator('.customer-card').count()

    // 取第一个卡片的姓名，用前两个字符搜索
    const firstName = await page.locator('.customer-name').first().textContent()
    const query = firstName!.slice(0, 2)

    await page.locator('.search-input input').fill(query)
    await page.waitForTimeout(300)  // debounce

    const after = await page.locator('.customer-card').count()
    // 过滤后数量 ≤ 原始数量
    expect(after).toBeLessThanOrEqual(totalBefore)
    // 当前可见的每张卡姓名都应包含查询词
    const visibleNames = await page.locator('.customer-card .customer-name').allTextContents()
    for (const name of visibleNames) {
      expect(name).toContain(query)
    }

    // 清空搜索，恢复全部
    await page.locator('.search-input input').clear()
    await page.waitForTimeout(300)
    const restored = await page.locator('.customer-card').count()
    expect(restored).toBe(totalBefore)
  })

  test('已结清区块可展开和收起', async ({ page }) => {
    // 若无"已结清"区块则跳过
    const settledTitle = page.locator('.settled-title')
    if (!(await settledTitle.isVisible())) {
      test.skip()
      return
    }
    const expandBtn = settledTitle.locator('button')

    // 默认收起
    const isExpanded = (await expandBtn.textContent())?.includes('展开')
    if (isExpanded) {
      // 展开
      await expandBtn.click()
      await expect(expandBtn).toContainText('收起')
      // 收起
      await expandBtn.click()
      await expect(expandBtn).toContainText('展开')
    } else {
      // 收起
      await expandBtn.click()
      await expect(expandBtn).toContainText('展开')
      // 再展开
      await expandBtn.click()
      await expect(expandBtn).toContainText('收起')
    }
  })

  test('点击客户卡片跳转到详情页', async ({ page }) => {
    await page.locator('.customer-card').first().click()
    await expect(page).toHaveURL(/\/customers\/[0-9a-f-]{36}/, { timeout: 5000 })
    // 详情页有"返回"按钮
    await expect(page.locator('button', { hasText: '返回' })).toBeVisible({ timeout: 5000 })
  })

  test('新增客户 — 空姓名应提示校验错误', async ({ page }) => {
    await page.locator('button', { hasText: '新增客户' }).click()
    await expect(page.locator('.el-dialog')).toBeVisible()
    // 不填名字直接保存
    await page.locator('.el-dialog button', { hasText: '保存' }).click()
    await expect(page.locator('.el-form-item__error')).toBeVisible()
    // 关闭弹窗
    await page.locator('.el-dialog button', { hasText: '取消' }).click()
  })

  test('新增客户 — 填写姓名后可成功保存', async ({ page }) => {
    const testName = `测试客户_${Date.now()}`
    await page.locator('button', { hasText: '新增客户' }).click()
    await expect(page.locator('.el-dialog')).toBeVisible()

    await page.locator('.el-dialog input[placeholder="输入姓名"]').fill(testName)
    await page.locator('.el-dialog button', { hasText: '保存' }).click()

    // 弹窗关闭
    await expect(page.locator('.el-dialog')).not.toBeVisible({ timeout: 5000 })

    // 新客户无账目，pending_amount=0 → 归入"已结清"折叠区，需先展开
    const settledTitle = page.locator('.settled-title')
    if (await settledTitle.isVisible()) {
      const expandBtn = settledTitle.locator('button')
      if ((await expandBtn.textContent())?.includes('展开')) {
        await expandBtn.click()
      }
    }

    // 也可以用搜索框定位
    await page.locator('.search-input input').fill(testName)
    await page.waitForTimeout(300)

    await expect(page.locator('.customer-card', { hasText: testName })).toBeVisible({ timeout: 5000 })
  })
})
