# Shell Mode Redesign Design

**Date:** 2026-07-13  
**Status:** Approved (user: implement directly)  
**Approach:** Dual-channel per machine (PTY + aux SSH for monitor/SFTP)

## Decisions
- History click → connect / focus tab
- Monitor via independent SSH (not PTY)
- SFTP cwd: parse `cd` + low-freq `pwd` calibrate
- Delivery order: history/picker → monitor → SFTP

## Layout
- No tabs: center = connection history
- Tabs present: left = monitor, center = terminal, bottom = Files toolbar (SFTP)
- Tab bar leading folder button → machine picker dialog (connect + edit)

## Persistence
- Connection history: `~/.cmd-config/shell_history.json`
