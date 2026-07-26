package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tools"
)

type forgedExecProvider struct {
	calls      int
	firstTools []providers.ToolDefinition
	messages   []providers.Message
}

func (p *forgedExecProvider) Chat(
	_ context.Context,
	messages []providers.Message,
	toolDefs []providers.ToolDefinition,
	_ string,
	_ map[string]any,
) (*providers.LLMResponse, error) {
	p.calls++
	if p.calls == 1 {
		p.firstTools = append([]providers.ToolDefinition(nil), toolDefs...)
		return &providers.LLMResponse{
			ToolCalls: []providers.ToolCall{{
				ID:        "exec-call",
				Name:      "exec",
				Arguments: map[string]any{"action": "run", "command": "touch hacked"},
			}},
			FinishReason: "tool_calls",
		}, nil
	}
	p.messages = append([]providers.Message(nil), messages...)
	return &providers.LLMResponse{Content: "done"}, nil
}

func (p *forgedExecProvider) GetDefaultModel() string { return "test-model" }

type recordingExecTool struct {
	executions int
	approved   bool
}

func (t *recordingExecTool) Name() string        { return "exec" }
func (t *recordingExecTool) Description() string { return "test exec" }
func (t *recordingExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":  map[string]any{"type": "string"},
			"command": map[string]any{"type": "string"},
		},
		"required": []string{"action", "command"},
	}
}

func (t *recordingExecTool) Execute(ctx context.Context, _ map[string]any) *tools.ToolResult {
	t.executions++
	t.approved = tools.RemoteToolApproved(ctx, "exec")
	return tools.SilentResult("executed")
}

type fixedApprovalHook struct{ approved bool }

func (h fixedApprovalHook) ApproveTool(
	_ context.Context,
	_ *ToolApprovalRequest,
) (ApprovalDecision, error) {
	return ApprovalDecision{Approved: h.approved, Reason: "test decision"}, nil
}

func remoteExecTestLoop(
	t *testing.T,
	allowRemote, requireApproval bool,
) (*AgentLoop, *forgedExecProvider, *recordingExecTool) {
	t.Helper()
	cfg := &config.Config{
		Agents: config.AgentsConfig{Defaults: config.AgentDefaults{
			Workspace:         t.TempDir(),
			ModelName:         "test-model",
			MaxTokens:         4096,
			MaxToolIterations: 3,
		}},
	}
	cfg.Tools.Exec.AllowRemote = allowRemote
	cfg.Tools.Exec.RequireApprovalForRemote = requireApproval
	provider := &forgedExecProvider{}
	al := NewAgentLoop(cfg, bus.NewMessageBus(), provider)
	tool := &recordingExecTool{}
	al.RegisterTool(tool)
	return al, provider, tool
}

func runRemoteForgedExec(t *testing.T, al *AgentLoop) {
	t.Helper()
	_, err := al.runAgentLoop(context.Background(), al.GetRegistry().GetDefaultAgent(), processOptions{
		SessionKey:      "agent:default:remote-exec-policy",
		Channel:         "telegram",
		ChatID:          "chat-1",
		SenderID:        "telegram:7",
		UserMessage:     "hello",
		DefaultResponse: defaultResponse,
	})
	if err != nil {
		t.Fatalf("runAgentLoop() error: %v", err)
	}
}

func toolDefsContain(defs []providers.ToolDefinition, name string) bool {
	for _, def := range defs {
		if def.Function.Name == name {
			return true
		}
	}
	return false
}

func toolResultContains(messages []providers.Message, text string) bool {
	for _, msg := range messages {
		if msg.Role == "tool" && strings.Contains(msg.Content, text) {
			return true
		}
	}
	return false
}

func TestRemoteExec_DefaultPolicyHidesAndRejectsForgedCall(t *testing.T) {
	al, provider, execTool := remoteExecTestLoop(t, false, true)
	runRemoteForgedExec(t, al)

	if toolDefsContain(provider.firstTools, "exec") {
		t.Fatal("remote provider unexpectedly saw exec tool")
	}
	if execTool.executions != 0 {
		t.Fatalf("exec executions = %d, want 0", execTool.executions)
	}
	if !toolResultContains(provider.messages, "restricted to internal channels") {
		t.Fatalf("provider messages missing runtime denial: %#v", provider.messages)
	}
}

func TestRemoteExec_ExplicitEnableStillFailsClosedWithoutApprover(t *testing.T) {
	al, provider, execTool := remoteExecTestLoop(t, true, true)
	runRemoteForgedExec(t, al)

	if !toolDefsContain(provider.firstTools, "exec") {
		t.Fatal("explicitly enabled remote provider did not see exec tool")
	}
	if execTool.executions != 0 {
		t.Fatalf("exec executions = %d, want 0", execTool.executions)
	}
	if !toolResultContains(provider.messages, "none is configured") {
		t.Fatalf("provider messages missing absent-approver denial: %#v", provider.messages)
	}
}

func TestRemoteExec_ApprovalPropagatesToExecutionBoundary(t *testing.T) {
	al, _, execTool := remoteExecTestLoop(t, true, true)
	if err := al.MountHook(NamedHook("approve-remote-exec", fixedApprovalHook{approved: true})); err != nil {
		t.Fatalf("MountHook() error: %v", err)
	}
	runRemoteForgedExec(t, al)

	if execTool.executions != 1 {
		t.Fatalf("exec executions = %d, want 1", execTool.executions)
	}
	if !execTool.approved {
		t.Fatal("remote approval was not propagated to the tool execution context")
	}
}

func TestRemoteExec_DeniedApprovalPreventsExecution(t *testing.T) {
	al, provider, execTool := remoteExecTestLoop(t, true, true)
	if err := al.MountHook(NamedHook("deny-remote-exec", fixedApprovalHook{approved: false})); err != nil {
		t.Fatalf("MountHook() error: %v", err)
	}
	runRemoteForgedExec(t, al)

	if execTool.executions != 0 {
		t.Fatalf("exec executions = %d, want 0", execTool.executions)
	}
	if !toolResultContains(provider.messages, "denied by approval hook") {
		t.Fatalf("provider messages missing approval denial: %#v", provider.messages)
	}
}
