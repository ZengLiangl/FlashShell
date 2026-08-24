import test from 'node:test'
import assert from 'node:assert/strict'
import {
  createLocalEchoState,
  applyLocalEchoInput,
  applyRemoteEchoSuppression,
  isTerminalInputCursorAtLineEnd,
} from './shellLocalEcho.js'

function mockTerminal(lineText, cursorX) {
  const chars = [...lineText]
  return {
    buffer: {
      active: {
        cursorY: 0,
        cursorX,
        getLine: () => ({
          length: chars.length,
          getCell: (x) => ({
            getChars: () => (x < chars.length ? chars[x] : ''),
          }),
        }),
      },
    },
  }
}

test('pending 为空时不做本地退格，避免删到提示符', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('ww', state)
  applyRemoteEchoSuppression('ww', state)
  assert.equal(state.pending, '')

  const { display } = applyLocalEchoInput('\x7f', state)
  assert.equal(display, '')
  assert.equal(state.backspaces, 0)
  assert.equal(state.suppressedSuffix, '')
})

test('高延迟下先本地删光再收到整段远端回显时不应复活字符', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('ww123', state)
  applyLocalEchoInput('\x7f\x7f\x7f\x7f\x7f', state)
  assert.equal(state.pending, '')
  assert.equal(state.backspaces, 5)
  assert.equal(state.suppressedSuffix, 'ww123')

  const out = applyRemoteEchoSuppression('ww123', state)
  assert.equal(out, '')
  assert.equal(state.backspaces, 0)
  assert.equal(state.suppressedSuffix, '')
})

test('先删后缀再收到整段远端回显时保留未删前缀', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('ww123', state)
  applyLocalEchoInput('\x7f\x7f\x7f', state)
  assert.equal(state.pending, 'ww')
  assert.equal(state.suppressedSuffix, '123')

  const out = applyRemoteEchoSuppression('ww123', state)
  assert.equal(out, '')
  assert.equal(state.pending, '')
  assert.equal(state.suppressedSuffix, '')
})

test('远端退格回显与本地退格计数应互相抵消', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('ab', state)
  applyLocalEchoInput('\x7f', state)
  assert.equal(state.backspaces, 1)

  const out = applyRemoteEchoSuppression('\b \b', state)
  assert.equal(out, '')
  assert.equal(state.backspaces, 0)
})

test('用户报告场景：删光后迟到的整段回显不应复活字符', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('ww1234242342342', state)
  for (let i = 0; i < 13; i += 1) {
    applyLocalEchoInput('\x7f', state)
  }
  assert.equal(state.pending, 'ww')

  for (let i = 0; i < 2; i += 1) {
    const { display } = applyLocalEchoInput('\x7f', state)
    assert.equal(display, '\b \b')
  }
  assert.equal(state.pending, '')
  assert.equal(state.suppressedSuffix, 'ww1234242342342')

  const ghost = applyRemoteEchoSuppression('ww1234242342342', state)
  assert.equal(ghost, '')
})

test('日志输出不应被误抑制', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('tail -f ', state)
  applyLocalEchoInput('\x7f', state)
  assert.equal(state.backspaces, 1)
  assert.equal(state.suppressedSuffix, ' ')

  const logLine = '2026-08-24 INFO com.example.Service started\r\n'
  const out = applyRemoteEchoSuppression(logLine, state)
  assert.equal(out, logLine)
  assert.equal(state.backspaces, 0)
  assert.equal(state.suppressedSuffix, '')
})

test('光标在行中间时不应视为行尾', () => {
  const line = 'tail -f logs/xyj-merchant-all.log'
  const cursorAfterF = line.indexOf('f') + 1
  assert.equal(isTerminalInputCursorAtLineEnd(mockTerminal(line, cursorAfterF)), false)
  assert.equal(isTerminalInputCursorAtLineEnd(mockTerminal(line, line.length)), true)
})

test('整行删光后收到部分匹配的远端内容应只输出不匹配后缀', () => {
  const state = createLocalEchoState()
  applyLocalEchoInput('abc', state)
  applyLocalEchoInput('\x7f\x7f\x7f', state)
  assert.equal(state.suppressedSuffix, 'abc')

  const out = applyRemoteEchoSuppression('abcX', state)
  assert.equal(out, 'X')
})
