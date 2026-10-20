package cluster

type Hooks map[string]map[string][]string

func (h Hooks) ForActionAndStage(action, stage string) []string {
	if h == nil {
		return nil
	}
	stages, ok := h[action]
	if !ok {
		return nil
	}
	cmds, ok := stages[stage]
	if !ok {
		return nil
	}
	return cmds
}
