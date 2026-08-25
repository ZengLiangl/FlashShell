import test from 'node:test'
import assert from 'node:assert/strict'
import {
  getTaskStepCounts,
  calcTaskProgressPercentage,
  formatTaskStepProgress,
} from './taskProgress.js'

test('已完成 2/3 步时进度约 67%，文案与计数一致', () => {
  const status = { isRunning: true, completedSteps: 2, totalSteps: 3 }
  assert.deepEqual(getTaskStepCounts(status), { completed: 2, total: 3 })
  assert.equal(calcTaskProgressPercentage(status), 67)
  assert.equal(formatTaskStepProgress(status), '2/3 步')
})

test('首步刚开始时显示 0/N，不把当前步算成已完成', () => {
  const status = { isRunning: true, completedSteps: 0, totalSteps: 3 }
  assert.equal(calcTaskProgressPercentage(status), 0)
  assert.equal(formatTaskStepProgress(status), '0/3 步')
})

test('未运行或总步数为 0 时进度为 0', () => {
  assert.equal(calcTaskProgressPercentage({ isRunning: false, completedSteps: 3, totalSteps: 3 }), 0)
  assert.equal(calcTaskProgressPercentage({ isRunning: true, completedSteps: 1, totalSteps: 0 }), 0)
  assert.equal(formatTaskStepProgress({ isRunning: true, completedSteps: 1, totalSteps: 0 }), '')
})

test('completedSteps 超出 totalSteps 时钳制', () => {
  const status = { isRunning: true, completedSteps: 5, totalSteps: 3 }
  assert.deepEqual(getTaskStepCounts(status), { completed: 3, total: 3 })
  assert.equal(calcTaskProgressPercentage(status), 100)
})
