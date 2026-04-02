package openai

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// PayloadType identifies the kind of SSE payload delivered to ProcessReplyStream callbacks.
type PayloadType string

const (
	PayloadTypeComment PayloadType = "comment"
	PayloadTypeData    PayloadType = "data"
	PayloadTypeDone    PayloadType = "done"
)

// ProcessReplyStream 从 body 中读取 SSE（Server-Sent Events）流，
// 将流中的 SSE 字段通过 onPayload 回调返回给调用方。
//
// SSE 协议约定（RFC 8895）：
//   - 每个字段占一行，格式为 "field: value"。
//   - 同一事件内可包含多行 "data:" 字段，用 "\n" 拼接后构成完整 payload。
//   - 以空行（\n\n 或 \r\n\r\n）作为一个事件的结束标志。
//   - 以 ":" 开头的行通常是心跳/注释，本实现会以 comment 类型回调。
//   - 服务端通过 "data: [DONE]" 标志流正常结束，本实现也会回调 done 类型。
//
// 参数：
//   - body    HTTP 响应体，格式要求为 text/event-stream。
//   - onPayload 回调函数：
//   - comment: 注释/心跳内容（去掉开头的 ":" 和一个可选空格）
//   - data: 一个完整事件内拼接后的 data payload
//   - done: 收到 [DONE] 结束标志
//   - 其他字段名（如 event / id / retry）会原样作为 dataType 回调
//     返回非 nil error 时，函数立即中断并将该 error 向上返回。
//
// 返回值：
//   - nil   流正常结束（读完全部内容或收到 [DONE]）。
//   - error 可能来源：onPayload 回调、底层 IO 读取、或 onPayload 为 nil。
func ProcessReplyStream(body io.Reader, onPayload func(dataType PayloadType, data []byte) error) error {
	if onPayload == nil {
		return errors.New("onPayload callback is nil")
	}

	reader := bufio.NewScanner(body)
	// bufio.Scanner 默认单行上限 64KB，这里扩展到 1MB，
	// 避免模型输出较长 JSON 行时触发 "token too long" 错误。
	reader.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// eventData 缓存当前事件内收集到的所有 data 行。
	// 同一事件可能有多行 data，最终用 "\n" 拼接为一个完整 payload。
	var eventData []string

	// flushEvent 在遇到空行（事件边界）时触发：
	//   1. 将 eventData 拼接并清空。
	//   2. 识别 [DONE] 结束标志并回调 done，返回 (true, nil)。
	//   3. 否则以 data 类型回调完整 payload，透传其错误。
	flushEvent := func() (bool, error) {
		if len(eventData) == 0 {
			return false, nil
		}

		payload := strings.Join(eventData, "\n")
		eventData = eventData[:0] // 清空，复用已分配内存

		if payload == "[DONE]" {
			if err := onPayload(PayloadTypeDone, []byte(payload)); err != nil {
				return false, err
			}
			return true, nil
		}
		if err := onPayload(PayloadTypeData, []byte(payload)); err != nil {
			return false, err
		}
		return false, nil
	}

	for reader.Scan() {
		// 兼容 Windows CRLF 行尾，统一去掉 \r，
		// 保证空行判断（line == ""）在两种系统上都正常。
		line := strings.TrimSuffix(reader.Text(), "\r")

		if line == "" {
			// 空行：当前 SSE 事件结束，触发 flush。
			done, err := flushEvent()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			// SSE 注释行（keepalive 心跳等），也透传给调用方。
			comment := strings.TrimPrefix(line, ":")
			comment = strings.TrimPrefix(comment, " ")
			if err := onPayload(PayloadTypeComment, []byte(comment)); err != nil {
				return err
			}
			continue
		}

		fieldName := line
		fieldValue := ""
		if idx := strings.IndexByte(line, ':'); idx >= 0 {
			fieldName = line[:idx]
			fieldValue = line[idx+1:]
			fieldValue = strings.TrimPrefix(fieldValue, " ")
		}

		if fieldName == string(PayloadTypeData) {
			eventData = append(eventData, fieldValue)
			continue
		}

		if err := onPayload(PayloadType(fieldName), []byte(fieldValue)); err != nil {
			return err
		}
	}

	// 检查底层读取是否发生 IO 错误（区别于正常 EOF）。
	if err := reader.Err(); err != nil {
		return err
	}

	// 兜底 flush：兼容流结尾没有空行的服务端实现，
	// 确保最后一个事件不会因缺少空行而丢失。
	_, err := flushEvent()
	return err
}
