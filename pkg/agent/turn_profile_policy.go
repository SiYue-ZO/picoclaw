package agent

import (
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/constants"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func resolveTurnProfileOptions(cfg *config.Config, opts processOptions) (processOptions, error) {
	if cfg == nil {
		return opts, nil
	}
	profile, ok, err := cfg.Agents.Defaults.ResolveTurnProfile()
	if err != nil {
		return opts, err
	}
	if !ok {
		return opts, nil
	}
	opts.TurnProfile = profile
	if profile.HistoryMode == config.TurnProfileModeOff {
		opts.NoHistory = true
		opts.EnableSummary = false
	}
	return opts, nil
}

func turnProfileSystemPromptOff(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SystemPromptMode == config.TurnProfileModeOff
}

func turnProfileSkillsOff(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SkillsMode == config.TurnProfileModeOff
}

func turnProfileCustomSkills(profile config.EffectiveTurnProfile) bool {
	return profile.Enabled && profile.SkillsMode == config.TurnProfileModeCustom
}

func turnProfileHasCallableTools(
	profile config.EffectiveTurnProfile,
	defs []providers.ToolDefinition,
) bool {
	if !profile.Enabled {
		return true
	}
	return len(filterToolsByTurnProfile(defs, profile)) > 0
}

func turnProfileToolAllowed(profile config.EffectiveTurnProfile, name string) bool {
	if !profile.Enabled {
		return true
	}
	switch profile.ToolsMode {
	case config.TurnProfileModeOff:
		return false
	case config.TurnProfileModeCustom:
		allowed := cleanAllowedSet(profile.AllowedTools)
		if len(allowed) == 0 {
			return false
		}
		_, ok := allowed[strings.ToLower(strings.TrimSpace(name))]
		return ok
	default:
		return true
	}
}

func toolUseSystemPromptRule() string {
	return "**ALWAYS use tools** - When you need to perform an action (schedule reminders, send messages, execute commands, etc.), you MUST call the appropriate tool. Do NOT just say you'll do it or pretend to do it."
}

func filterNamesByTurnProfile(names []string, allowed []string) []string {
	if len(names) == 0 {
		return nil
	}
	allowedSet := cleanAllowedSet(allowed)
	if len(allowedSet) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := allowedSet[strings.ToLower(strings.TrimSpace(name))]; ok {
			out = append(out, name)
		}
	}
	return out
}

func filterToolsByTurnProfile(
	defs []providers.ToolDefinition,
	profile config.EffectiveTurnProfile,
) []providers.ToolDefinition {
	if !profile.Enabled {
		return defs
	}
	switch profile.ToolsMode {
	case config.TurnProfileModeOff:
		return nil
	case config.TurnProfileModeCustom:
		allowed := cleanAllowedSet(profile.AllowedTools)
		if len(allowed) == 0 {
			return nil
		}
		filtered := make([]providers.ToolDefinition, 0, len(defs))
		for _, def := range defs {
			if _, ok := allowed[strings.ToLower(strings.TrimSpace(def.Function.Name))]; ok {
				filtered = append(filtered, def)
			}
		}
		return filtered
	default:
		return defs
	}
}

func toolAllowedForTurn(
	cfg *config.Config,
	profile config.EffectiveTurnProfile,
	channel, name string,
) (bool, string) {
	name = strings.ToLower(strings.TrimSpace(name))
	if !turnProfileToolAllowed(profile, name) {
		return false, "tool is not allowed by the active turn profile"
	}
	if name != "exec" {
		return true, ""
	}

	channel = strings.TrimSpace(channel)
	if channel == "" {
		return false, "exec requires channel context"
	}
	if constants.IsInternalChannel(channel) {
		return true, ""
	}
	if cfg == nil || !cfg.Tools.Exec.AllowRemote {
		return false, "exec is restricted to internal channels"
	}
	return true, ""
}

func filterToolsForTurn(
	cfg *config.Config,
	defs []providers.ToolDefinition,
	profile config.EffectiveTurnProfile,
	channel string,
) []providers.ToolDefinition {
	filtered := filterToolsByTurnProfile(defs, profile)
	if len(filtered) == 0 {
		return filtered
	}
	out := make([]providers.ToolDefinition, 0, len(filtered))
	for _, def := range filtered {
		if allowed, _ := toolAllowedForTurn(cfg, profile, channel, def.Function.Name); allowed {
			out = append(out, def)
		}
	}
	return out
}

func remoteExecRequiresApproval(cfg *config.Config, channel, name string) bool {
	if cfg == nil || !cfg.Tools.Exec.RequireApprovalForRemote ||
		!strings.EqualFold(strings.TrimSpace(name), "exec") {
		return false
	}
	channel = strings.TrimSpace(channel)
	return channel != "" && !constants.IsInternalChannel(channel)
}

func cleanAllowedSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}
