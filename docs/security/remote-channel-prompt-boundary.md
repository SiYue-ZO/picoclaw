# Remote channel prompt and tool boundary

PicoClaw treats sender names, sender IDs, chat IDs, and other platform identity values as untrusted data. Current sender metadata is JSON-encoded into the current `user` message. It is never added to a provider system message, Anthropic system block, or OpenAI/Azure/Codex `instructions` field. The original message body remains unchanged in session storage; the model-only envelope is rebuilt for the active turn and is not persisted.

Display names are normalized before model use: CR/LF/TAB become spaces, NUL and control/format characters (including bidi controls) are removed, and the result is limited to 128 Unicode runes. JSON serialization supplies quote and backslash escaping. This normalization is defense in depth; user-role isolation is the security boundary.

## Remote exec policy

Configuration schema v4 changes `tools.exec.allow_remote` from an unsafe default to `false` and adds `tools.exec.require_approval_for_remote`, defaulting to `true`.

- Remote turns do not receive the `exec` tool definition unless `allow_remote` is explicitly enabled.
- Forged calls and tool names rewritten by hooks are checked again at execution time.
- Missing channel context is denied.
- With remote approval required, a missing, failed, timed-out, or denying approval hook is denied.
- Approval is propagated to the exec implementation and checked again for every action, including background-session read/write/kill actions.
- Deny patterns and workspace restrictions are useful guardrails, not a sandbox. Use OS isolation for hostile workloads.

Cron command jobs keep their separate `tools.cron.command_allowed_remotes` gate. Spawned/subagent turns inherit the originating channel, so their exec calls receive the same remote policy. Workspace file tools retain their existing workspace/path restrictions; `allow_remote` controls exec, not every file or messaging tool.

## Upgrade and rollback

When a v3 configuration is loaded, PicoClaw creates a dated backup, upgrades it to v4, sets omitted `allow_remote` values to `false`, and sets omitted remote approval values to `true`. An explicit existing `allow_remote: true` is preserved but produces a high-risk startup warning. The Web UI always saves both booleans explicitly.

For a legitimate remote command workflow, set `allow_remote: true` and configure a `ToolApprover` hook. Disabling `require_approval_for_remote` restores the older behavior but removes an important deterministic boundary and is not recommended.

To downgrade the binary, restore the dated pre-migration backup. A v4 file must not simply be handed to an older binary: restore version 3 and remove `require_approval_for_remote` using the backup instead.
