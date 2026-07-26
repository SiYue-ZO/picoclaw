> 返回 [README](../../project/README.zh.md)

# Telegram

Telegram Channel 通过 Telegram 机器人 API 使用长轮询实现基于机器人的通信。它支持文本消息、媒体附件（照片、语音、音频、文档）、语音转录（配置见[提供商与模型配置](../../guides/providers.zh.md#语音转录)），以及内置命令处理器。

## 配置

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "123456789:ABCdefGHIjklMNOpqrsTUVwxyz",
      "allow_from": ["123456789"],
      "proxy": "",
      "use_markdown_v2": false
    }
  }
}
```

| 字段             | 类型   | 必填 | 描述                                                      |
| ---------------- | ------ | ---- | --------------------------------------------------------- |
| enabled          | bool   | 是   | 是否启用 Telegram 频道                                    |
| token            | string | 是   | Telegram 机器人 API Token                                 |
| allow_from       | array  | 新建 Web 配置时是 | 用户 ID 白名单；旧配置中的空列表仍表示允许所有用户       |
| proxy            | string | 否   | 连接 Telegram API 的代理 URL (例如 http://127.0.0.1:7890) |
| use_markdown_v2 | bool   | 否   | 启用 Telegram MarkdownV2 格式化                           |

## 设置流程

1. 在 Telegram 中搜索 `@BotFather`
2. 发送 `/newbot` 命令并按照提示创建新机器人
3. 获取 HTTP API Token
4. 将 Token 填入配置文件中
5. 至少配置一个 `allow_from` 用户 ID（可通过 `@userinfobot` 获取）

> **安全提示：** 空列表或 `"*"` 会向所有能够联系机器人的 Telegram 用户开放入口，因此 Web 配置流程要求至少填写一个 ID。为了兼容旧配置，直接加载配置文件时仍保留“空列表允许所有人”的旧语义；启用频道前请补充白名单。频道入口控制不能替代工具授权，远程 `exec` 默认仍为关闭状态。

## 内置命令

Telegram 会在启动时自动注册 PicoClaw 的顶级 Bot 命令，包括 `/start`、`/help`、`/show`、`/list` 和 `/use`。

与技能相关的命令：

- `/list skills`：列出当前 Agent 可见的已安装技能。
- `/use <skill> <message>`：只在本次请求中强制使用指定技能。
- `/use <skill>`：为同一聊天中的下一条消息预先启用该技能。
- `/use clear`：清除待应用的技能覆盖。

示例：

```text
/list skills
/use git explain how to squash the last 3 commits
/use git
explain how to squash the last 3 commits
```

## 高级格式化

您可以设置 `use_markdown_v2: true` 来启用增强的格式化选项。这允许机器人使用 Telegram MarkdownV2 的全部功能，包括嵌套样式、剧透和自定义等宽代码块。

```json
{
  "channel_list": {
    "telegram": {
      "enabled": true,
      "type": "telegram",
      "token": "YOUR_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"],
      "use_markdown_v2": true
    }
  }
}
```
