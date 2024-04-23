package common

import (
	"encoding/json"
	"fmt"
	regexp "github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"
	"sort"
	"strconv"
	"strings"
)

func XmlFlags(messages []interface{}) (newMessages []interface{}) {
	if len(messages) == 0 {
		return nil
	}

	handles := xmlFlagsToContents(messages)

	for _, h := range handles {
		// 正则替换
		if h['t'] == "regex" {
			s := split(h['v'])
			if len(s) < 2 {
				continue
			}

			cmp := strings.TrimSpace(s[0])
			value := strings.TrimSpace(s[1])
			if cmp == "" {
				continue
			}

			// 忽略尾部n条
			pos, _ := strconv.Atoi(h['m'])
			if pos > -1 {
				pos = len(messages) - 1 - pos
				if pos < 0 {
					pos = 0
				}
			} else {
				pos = len(messages)
			}

			c := regexp.MustCompile(cmp, regexp.Compiled)
			for idx, message := range messages {
				m, ok := message.(map[string]interface{})
				if !ok {
					continue
				}
				if idx < pos && m["role"] != "system" {
					str := extractMessage(m)
					if str == nil {
						continue
					}

					replace, e := c.Replace(*str, value, -1, -1)
					if e != nil {
						logrus.Warn("compile failed: "+cmp, e)
						continue
					}

					m["content"] = replace
				}
			}
		}

		// 深度插入
		if h['t'] == "insert" {
			i, _ := strconv.Atoi(h['i'])
			messageL := len(messages)
			if h['m'] == "true" && messageL-1 < abs(i) {
				continue
			}

			pos := 0
			if i > -1 {
				// 正插
				pos = i
				if pos >= messageL {
					pos = messageL - 1
				}
			} else {
				// 反插
				pos = messageL + i
				if pos < 0 {
					pos = 0
				}
			}

			message, ok := messages[pos].(map[string]interface{})
			if !ok {
				continue
			}

			if h['r'] == "" {
				str := extractMessage(message)
				if str == nil {
					continue
				}
				message["content"] = *str + "\n\n" + h['v']
			} else {
				messages = append(messages[:pos+1], append([]interface{}{
					map[string]string{
						"role":    h['r'],
						"content": h['v'],
					},
				}, messages[pos+1:]...)...)
			}
		}

		// 历史记录
		if h['t'] == "histories" {
			content := strings.TrimSpace(h['v'])
			if len(content) < 2 || content[0] != '[' || content[len(content)-1] != ']' {
				continue
			}
			var pMessages []interface{}
			if e := json.Unmarshal([]byte(content), &pMessages); e != nil {
				logrus.Error("histories flags handle failed: ", e)
				continue
			}

			if len(pMessages) == 0 {
				continue
			}

			for idx := 0; idx < len(messages); idx++ {
				message, ok := messages[idx].(map[string]interface{})
				if !ok {
					continue
				}
				if !strings.Contains("|system|function|", message["role"].(string)) {
					messages = append(messages[:idx], append(pMessages, messages[idx:]...)...)
					break
				}
			}
		}
	}

	newMessages = messages
	// TODO
	return
}

func xmlFlagsToContents(messages []interface{}) (handles []map[uint8]string) {
	var (
		parser = XmlParser{[]string{
			"regex",
			`r:@-*\d+`,
			"histories",
		}}
	)

	for _, message := range messages {
		m, ok := message.(map[string]interface{})
		if !ok {
			continue
		}

		role := m["role"]
		if role != "system" && role != "user" {
			continue
		}

		str := extractMessage(m)
		if str == nil {
			continue
		}

		clean := func(ctx string) {
			m["content"] = strings.Replace(*str, ctx, "", -1)
		}

		content := *str
		nodes := parser.Parse(content)
		if len(nodes) == 0 {
			continue
		}

		for _, node := range nodes {
			// 注释内容删除
			if node.t == XML_TYPE_I {
				clean(content[node.index:node.end])
				continue
			}

			// 自由深度插入
			// inserts: 深度插入, i 是深度索引，v 是插入内容， o 是指令
			if node.t == XML_TYPE_X && node.tag[0] == '@' {
				c, _ := regexp.Compile(`@-*\d+`, regexp.Compiled)
				if matched, _ := c.MatchString(node.tag); matched {
					// 消息上下文次数少于插入深度时，是否忽略
					// 如不忽略，将放置在头部或者尾部
					miss := "true"
					if it, ok := node.attr["miss"]; ok {
						if v, o := it.(bool); !o || !v {
							miss = "false"
						}
					}
					// 插入元素
					// 为空则是拼接到该消息末尾
					r := ""
					if it, ok := node.attr["role"]; ok {
						switch s := it.(string); s {
						case "user", "system", "assistant":
							r = s
						}
					}
					handles = append(handles, map[uint8]string{'i': node.tag[1:], 'r': r, 'v': node.content, 'm': miss, 't': "insert"})
					clean(content[node.index:node.end])
				}
				continue
			}

			// 正则替换
			// regex: v 是正则内容
			if node.t == XML_TYPE_X && node.tag == "regex" {
				order := "0" // 优先级
				if o, ok := node.attr["order"]; ok {
					order = fmt.Sprintf("%v", o)
				}

				miss := "-1"
				if m, ok := node.attr["miss"]; ok {
					miss = fmt.Sprintf("%v", m)
				}

				handles = append(handles, map[uint8]string{'m': miss, 'o': order, 'v': node.content, 't': "regex"})
				clean(content[node.index:node.end])
				continue
			}

			// 历史记录
			if node.t == XML_TYPE_X && node.tag == "histories" {
				s := strings.TrimSpace(node.content)
				if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
					handles = append(handles, map[uint8]string{'v': s, 't': "histories"})
					clean(content[node.index:node.end])
				}
				continue
			}
		}
	}

	if len(handles) > 0 {
		sort.Slice(handles, func(i, j int) bool {
			return handles[i]['o'] > handles[j]['o']
		})
	}
	return
}
