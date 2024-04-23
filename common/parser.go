package common

import (
	"bytes"
	regexp "github.com/dlclark/regexp2"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

const (
	XML_TYPE_S = iota // 普通字符串
	XML_TYPE_X        // XML标签
	XML_TYPE_I        // 注释标签

	BYTES_FIELD = "__BYTES_FILED__"
)

type xNode struct {
	index   int
	end     int
	tag     string
	t       int
	content string
	count   int
	attr    map[string]interface{}
	child   []xNode
}

type BytesReadCloser struct {
	*bytes.Reader
}

type XmlParser struct {
	whiteList []string
}

func (BytesReadCloser) Close() error {
	return nil
}

func trimCdata(value string) string {
label:
	valueL := len(value)
	for i := 0; i < valueL; i++ {
		if value[i] == '<' && nextStr(value, i, "![CDATA[") {
			n := searchStr(value, i+9, "]]>")
			if n >= 0 {
				value = value[:i] + value[i+9:n] + value[n+3:]
				goto label
			}
		}
	}
	return value
}

// xml解析的简单实现
func (xml XmlParser) Parse(value string) []xNode {
	messageL := len(value)
	if messageL == 0 {
		return nil
	}
	return xml.xmlLoop(value)
}

func (xml XmlParser) xmlLoop(value string) (slice []xNode) {
	content := value
	contentL := len(content)
	var curr *xNode = nil
	for i := 0; i < contentL; i++ {
		// curr 的标记不完整跳过该标记，重新扫描
		if i == contentL-1 {
			if curr != nil {
				if curr.index < curr.end {
					slice = append(slice, *curr)
					i = curr.end
				} else {
					i = curr.index + len(curr.tag) + 1
				}
				curr = nil
				if i >= contentL {
					return
				}
			}
		}

		if content[i] == '<' {
			// =========================================================
			// ⬇⬇⬇⬇⬇ 结束标记 ⬇⬇⬇⬇⬇
			if curr != nil && next(content, i, '/') {
				n := search(content, i, '>')
				// 找不到 ⬇⬇⬇⬇⬇
				if n == -1 {
					// 丢弃
					curr = nil
					break
				}
				// 找不到 ⬆⬆⬆⬆⬆

				s := strings.Split(curr.tag, " ")
				if s[0] == content[i+2:n] {
					step := 2 + len(curr.tag)
					curr.t = XML_TYPE_X
					curr.end = n + 1
					// 解析xml参数
					if len(s) > 1 {
						curr.tag = s[0]
						curr.attr = parseAttr(s[1:])
					}

					str := content[curr.index+step : curr.end-len(s[0])-3]
					curr.child = xml.xmlLoop(str)
					curr.content = trimCdata(str)
					i = curr.end - 1

					curr.count--
					if curr.count > 0 {
						if i == contentL-1 {
							i--
						}
						continue
					}

					slice = append(slice, *curr)
					curr = nil
				}
				// ⬆⬆⬆⬆⬆ 结束标记 ⬆⬆⬆⬆⬆

				// =========================================================
				//
			} else if nextStr(content, i, "![CDATA[") {
				//
				// ⬇⬇⬇⬇⬇ <![CDATA[xxx]]> CDATA结构体 ⬇⬇⬇⬇⬇
				n := searchStr(content, i+8, "]]>")
				if n < 0 {
					i += 7
					continue
				}
				i = n + 3
				// ⬆⬆⬆⬆⬆ <![CDATA[xxx]]> CDATA结构体 ⬆⬆⬆⬆⬆

				// =========================================================
				//

			} else if nextStr(content, i, "!--") {

				//
				// ⬇⬇⬇⬇⬇ 是否是注释 <!-- xxx --> ⬇⬇⬇⬇⬇

				n := searchStr(content, i+3, "-->")
				if n < 0 {
					i += 3
					continue
				}

				node := xNode{index: i, end: n + 3, content: content[i : n+3], t: XML_TYPE_I}
				slice = append(slice, node)
				// ⬆⬆⬆⬆⬆ 是否是注释 <!-- xxx --> ⬆⬆⬆⬆⬆
				// 循环后置++，所以-1
				i = node.end - 1
				// =========================================================
				//

			} else {

				//
				// ⬇⬇⬇⬇⬇ 新的 XML 标记 ⬇⬇⬇⬇⬇

				idx := i
				n := search(content, idx, '>')
			label:
				if n == -1 {
					break
				}

				idx = igCd(content, i, n)
				if idx == -1 {
					idx = n
					n = search(content, idx+1, '>')
					goto label
				}

				tag := content[i+1 : n]
				// whiteList 为nil放行所有标签，否则只解析whiteList中的
				contains := xml.whiteList == nil || containFor(xml.whiteList, func(item string) bool {
					if strings.HasPrefix(item, "r:") {
						cmp := item[2:]
						c := regexp.MustCompile(cmp, regexp.Compiled)
						matched, err := c.MatchString(tag)
						if err != nil {
							logrus.Warn("compile failed: "+cmp, err)
							return false
						}
						return matched
					}

					s := strings.Split(tag, " ")
					return item == s[0]
				})

				if !contains {
					i = n
					continue
				}

				// 这是一个自闭合的标签 <xxx />
				ch := content[n-1]
				if curr == nil && ch == '/' {
					tag = tag[:len(tag)-1]
					node := xNode{index: i, tag: tag, t: XML_TYPE_X}
					s := strings.Split(node.tag, " ")
					node.t = XML_TYPE_X
					node.end = n + 1
					// 解析xml参数
					if len(s) > 1 {
						node.tag = s[0]
						node.attr = parseAttr(s[1:])
					}
					slice = append(slice, node)
					i = node.end
					continue
				}

				if curr == nil {
					curr = &xNode{index: i, tag: tag, t: XML_TYPE_S, count: 1}
					i = n
					continue
				}

				if curr.tag == tag {
					curr.count++
					i = n
				}
				// ⬆⬆⬆⬆⬆ 新的 XML 标记 ⬆⬆⬆⬆⬆
			}
		}
	}

	return
}

// 查找从 index 开始，符合的字节返回其下标，没有则-1
func search(content string, index int, ch uint8) int {
	contentL := len(content)
	for i := index + 1; i < contentL; i++ {
		if content[i] == ch {
			return i
		}
	}
	return -1
}

// 查找从 index 开始，符合的字符串返回其下标，没有则-1
func searchStr(content string, index int, s string) int {
	l := len(s)
	contentL := len(content)
	for i := index + 1; i < contentL; i++ {
		if i+l > contentL {
			return -1
		}
		if content[i:i+l] == s {
			return i
		}
	}
	return -1
}

// 比较 index 的下一个字节，如果相同返回 true
func next(content string, index int, ch uint8) bool {
	contentL := len(content)
	if index+1 >= contentL {
		return false
	}
	return content[index+1] == ch
}

// 比较 index 的下一个字符串，如果相同返回 true
func nextStr(content string, index int, s string) bool {
	contentL := len(content)
	if index+1+len(s) >= contentL {
		return false
	}
	return content[index+1:index+1+len(s)] == s
}

// 解析xml标签的属性
func parseAttr(slice []string) map[string]interface{} {
	attr := make(map[string]interface{})
	for _, it := range slice {
		n := search(it, 0, '=')
		if n <= 0 {
			if len(it) > 0 && it != "=" {
				attr[it] = true
			}
			continue
		}

		if n == len(it)-1 {
			continue
		}

		if it[n+1] == '"' && it[len(it)-1] == '"' {
			attr[it[:n]] = trimCdata(it[n+2 : len(it)-1])
		}

		s := trimCdata(it[n+1:])
		v1, err := strconv.Atoi(s)
		if err == nil {
			attr[it[:n]] = v1
			continue
		}

		v2, err := strconv.ParseFloat(s, 10)
		if err == nil {
			attr[it[:n]] = v2
			continue
		}

		v3, err := strconv.ParseBool(s)
		if err == nil {
			attr[it[:n]] = v3
			continue
		}
	}
	return attr
}

// 跳过 CDATA标记
func igCd(content string, i, j int) int {
	content = content[i:j]
	n := searchStr(content, 0, "<![CDATA[")
	if n < 0 { // 不是CD
		return j
	}

	n = searchStr(content, n, "]]>")
	if n < 0 { // 没有闭合
		return -1
	}

	if n+3 == j { // 正好是闭合的标记
		return -1
	}

	// 已经闭合
	return j
}
