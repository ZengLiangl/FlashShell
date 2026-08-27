import test from 'node:test'
import assert from 'node:assert/strict'
import {
  getTaskStepCounts,
  calcTaskProgressPercentage,
  formatTaskStepProgress,
} from './taskProgress.js'

test('首步执行中显示 1/N，与终端「执行步骤 1」一致', () => {
  const status = { isRunning: true, completedSteps: 0, totalSteps: 3 }
  assert.deepEqual(getTaskStepCounts(status), { completed: 1, total: 3 })
  assert.equal(calcTaskProgressPercentage(status), 33)
  assert.equal(formatTaskStepProgress(status), '1/3 步')
})

test('已完成 2 步且正在跑第 3 步时显示 3/3', () => {
  const status = { isRunning: true, completedSteps: 2, totalSteps: 3 }
  assert.deepEqual(getTaskStepCounts(status), { completed: 3, total: 3 })
  assert.equal(calcTaskProgressPercentage(status), 100)
  assert.equal(formatTaskStepProgress(status), '3/3 步')
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

test('已停止时按已完成步数展示，不再 +1', () => {
  const status = { isRunning: false, completedSteps: 2, totalSteps: 10 }
  assert.deepEqual(getTaskStepCounts(status), { completed: 2, total: 10 })
  assert.equal(formatTaskStepProgress(status), '2/10 步')
})
