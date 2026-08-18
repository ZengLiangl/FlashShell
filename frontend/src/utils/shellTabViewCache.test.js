import test from 'node:test'
import assert from 'node:assert/strict'
import {
  hasUsefulMonitorSnapshot,
  shouldDiscardMonitorResult,
  shouldKeepCurrentMonitorSnapshot,
  shouldReplaceMonitorCache,
  canRestoreSftpBrowseView,
  shouldSilentSftpRefresh,
} from './shellTabViewCache.js'

const usefulA = {
  machineName: 'gw-12',
  memTotal: '15.2G',
  memPercent: 76,
  cpuPercent: 9,
  uptimeText: '12天',
  topMem: [{ pid: 1 }],
}

const usefulB = {
  machineName: 'jz',
  memTotal: '7.6G',
  memPercent: 40,
  cpuPercent: 3,
  uptimeText: '3天',
  topMem: [{ pid: 8 }],
}

const emptySnap = {
  machineName: 'jz',
  memTotal: '0',
  memPercent: 0,
  cpuPercent: 0,
  uptimeText: '0',
  topMem: [],
}

test('empty or zero snapshot is not useful', () => {
  assert.equal(hasUsefulMonitorSnapshot(null), false)
  assert.equal(hasUsefulMonitorSnapshot(emptySnap), false)
  assert.equal(hasUsefulMonitorSnapshot({ memTotal: '0', cpuPercent: 0 }), false)
})

test('idle CPU still useful when memory totals exist', () => {
  assert.equal(hasUsefulMonitorSnapshot({
    machineName: 'gw-12',
    memTotal: '15.2G',
    memPercent: 76,
    cpuPercent: 0,
    uptimeText: '0',
    topMem: [],
  }), true)
})

test('stale poll after tab switch must be discarded', () => {
  assert.equal(shouldDiscardMonitorResult({
    idle: false,
    activeMachine: 'jz',
    machineAtStart: 'gw-12',
  }), true)
  assert.equal(shouldDiscardMonitorResult({
    idle: true,
    activeMachine: 'gw-12',
    machineAtStart: 'gw-12',
  }), true)
  assert.equal(shouldDiscardMonitorResult({
    idle: false,
    activeMachine: 'jz',
    machineAtStart: 'jz',
  }), false)
})

test('aux miss keeps same-machine cache, not previous tab numbers', () => {
  assert.equal(shouldKeepCurrentMonitorSnapshot({
    current: usefulA,
    activeMachine: 'gw-12',
    incoming: emptySnap,
    auxMissing: true,
  }), true)
  assert.equal(shouldKeepCurrentMonitorSnapshot({
    current: usefulA,
    activeMachine: 'jz',
    incoming: emptySnap,
    auxMissing: true,
  }), false)
  assert.equal(shouldKeepCurrentMonitorSnapshot({
    current: usefulB,
    activeMachine: 'jz',
    incoming: usefulB,
    auxMissing: false,
  }), false)
})

test('empty cache may be written, later empty must not overwrite useful', () => {
  assert.equal(shouldReplaceMonitorCache({ current: emptySnap, hasExisting: false }), true)
  assert.equal(shouldReplaceMonitorCache({ current: emptySnap, hasExisting: true }), false)
  assert.equal(shouldReplaceMonitorCache({ current: usefulA, hasExisting: true }), true)
})

test('sftp restore needs a remembered cwd', () => {
  assert.equal(canRestoreSftpBrowseView(null), false)
  assert.equal(canRestoreSftpBrowseView({ cwd: '' }), false)
  assert.equal(canRestoreSftpBrowseView({ cwd: '/root/app' }), true)
})

test('sftp silent refresh only when restored tree and path unchanged', () => {
  assert.equal(shouldSilentSftpRefresh({
    restored: true,
    cwd: '/root/app',
    target: '/root/app',
    treeHasNodes: true,
  }), true)
  assert.equal(shouldSilentSftpRefresh({
    restored: true,
    cwd: '/root/app',
    target: '/var/log',
    treeHasNodes: true,
  }), false)
  assert.equal(shouldSilentSftpRefresh({
    restored: true,
    cwd: '/root/app',
    target: '/root/app',
    treeHasNodes: false,
  }), false)
  assert.equal(shouldSilentSftpRefresh({
    restored: false,
    cwd: '/root/app',
    target: '/root/app',
    treeHasNodes: true,
  }), false)
})
