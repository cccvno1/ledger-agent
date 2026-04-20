package chat

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/cccvno1/ledger-agent/internal/base/conf"
)

const systemPromptTemplate = `你是一个小批发商的智能记账助手。

## 当前时间
%s

## 你的工作方式
你只需要理解用户想干什么，提取用户说的原始信息传给工具。
工具会自动处理：客户匹配、商品匹配、日期计算、金额计算。
你不需要自己做任何计算或转换。

## 规则
1. 用户说的客户名、商品名原样传给工具，不要修改。
2. 日期表达原样传（如"昨天"就传"昨天"，工具会计算）。
3. 数字原样传递，不修改。
4. 展示工具返回的数据，不要自己编造数字。
5. 客户名和商品名只做精确匹配和别名匹配，匹配不到会自动新建，无需确认。
6. 未经用户说"确认"或"保存"，不调用 confirm_draft。
7. confirm_draft 保存全部草稿条目，不得跳过。
8. 收款前先调用 calculate_summary 显示欠款，再让用户确认金额。
9. 如果工具报错，如实告知用户，不编造结果。
10. 当用户在一句话中包含多条记录时，逐条调用 add_to_draft。
11. 操作记录中的 entry_id 等标识是真实的，可以直接用于 update_entry、delete_entry 等工具。`

// dynamicModifier returns a MessageModifier that injects current time and
// structured session state (draft, operation log, stats) into the system prompt.
func dynamicModifier(sessions *SessionStore) react.MessageModifier {
	return func(ctx context.Context, input []*schema.Message) []*schema.Message {
		now := time.Now().Format("2006-01-02 (Monday) 15:04 CST")
		prompt := fmt.Sprintf(systemPromptTemplate, now)

		// Inject structured context block from session state.
		sid := sessionIDFromCtx(ctx)
		if sid != "" {
			if sess := sessions.Get(sid); sess != nil {
				if block := sess.RenderContextBlock(); block != "" {
					prompt += "\n" + block
				}
			}
		}

		res := make([]*schema.Message, 0, len(input)+1)
		res = append(res, schema.SystemMessage(prompt))
		res = append(res, input...)
		return res
	}
}

// buildAgent creates a react Agent with the given tools.
func buildAgent(ctx context.Context, cfg conf.MiniMax, sessions *SessionStore, tools []tool.BaseTool) (*react.Agent, error) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("chat: agent: MINIMAX_API_KEY is not set")
	}

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  apiKey,
		Model:   cfg.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("chat: agent: new chat model: %w", err)
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MessageModifier: dynamicModifier(sessions),
	})
	if err != nil {
		return nil, fmt.Errorf("chat: agent: new agent: %w", err)
	}
	return agent, nil
}
