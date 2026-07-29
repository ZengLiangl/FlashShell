package machine

import "FlashDock/define"

// GroupParallelCommands 将 SubProject 命令切分为串行组；连续 parallel:true 的命令归入同一并发组。
func GroupParallelCommands(commands []define.Command) [][]define.Command {
	if len(commands) == 0 {
		return nil
	}
	var groups [][]define.Command
	i := 0
	for i < len(commands) {
		if !commands[i].Parallel {
			groups = append(groups, []define.Command{commands[i]})
			i++
			continue
		}
		start := i
		for i < len(commands) && commands[i].Parallel {
			i++
		}
		groups = append(groups, commands[start:i])
	}
	return groups
}
