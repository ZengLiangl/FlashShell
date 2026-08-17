import test from 'node:test'
import assert from 'node:assert/strict'
import {
  resolveCopyToOtherTargets,
  normalizeDestDir,
  canCopyToOtherSide,
} from './sftpCopyToOther.js'

test('normalizeDestDir strips trailing slash', () => {
  assert.equal(normalizeDestDir('/root/app/'), '/root/app')
  assert.equal(normalizeDestDir('/'), '/')
  assert.equal(normalizeDestDir('relative'), '')
})

test('copy to other unavailable without peers', () => {
  assert.equal(canCopyToOtherSide([]), false)
  assert.deepEqual(
    resolveCopyToOtherTargets({
      sourceSessionId: 'web1',
      sessions: [{ machineName: 'web1', connected: true }],
      cwdBySession: { web1: '/root' },
    }),
    [],
  )
})

test('prefers split peers over other sessions', () => {
  const targets = resolveCopyToOtherTargets({
    sourceSessionId: 'web1',
    splitSessionIds: ['web1', 'web2'],
    sessions: [
      { machineName: 'web1', tabLabel: 'web1', connected: true },
      { machineName: 'web2', tabLabel: 'web2', connected: true },
      { machineName: 'web3', tabLabel: 'web3', connected: true },
    ],
    cwdBySession: {
      web1: '/a',
      web2: '/root/app',
      web3: '/other',
    },
  })
  assert.equal(targets.length, 1)
  assert.equal(targets[0].sessionId, 'web2')
  assert.equal(targets[0].destDir, '/root/app')
})

test('falls back to all remote sessions when not split', () => {
  const targets = resolveCopyToOtherTargets({
    sourceSessionId: 'web1',
    splitSessionIds: [],
    sessions: [
      { machineName: 'web1', connected: true },
      { machineName: 'web2', tabLabel: 'B', connected: true },
      { machineName: 'local', kind: 'local', connected: true },
    ],
    cwdBySession: { web1: '/a', web2: '/b', local: '/Users/x' },
  })
  assert.equal(targets.length, 1)
  assert.equal(targets[0].sessionId, 'web2')
})

test('skips disconnected or missing cwd peers', () => {
  const targets = resolveCopyToOtherTargets({
    sourceSessionId: 'web1',
    splitSessionIds: ['web1', 'web2', 'web3'],
    sessions: [
      { machineName: 'web1', connected: true },
      { machineName: 'web2', connected: false },
      { machineName: 'web3', connected: true },
    ],
    cwdBySession: { web1: '/a', web3: '' },
  })
  assert.deepEqual(targets, [])
})
