package prettyjson

import (
	"bytes"
	"encoding/json"
)

// Marshal封装了一个支持中文且格式美观的 JSON 序列化方法
// v: 要序列化的对象
// 返回: JSON 字节切片 和 错误信息
func Marshal(v interface{}) ([]byte, error) {
	// 使用 Buffer 作为数据缓冲区
	var buf bytes.Buffer

	// 创建 Encoder
	encoder := json.NewEncoder(&buf)

	// 关键设置 1: SetEscapeHTML(false)
	// 禁止转义 HTML 字符，这样中文等非 ASCII 字符就不会被转义成 \uXXXX
	encoder.SetEscapeHTML(false)

	// 关键设置 2: SetIndent("", "    ")
	// 设置美化打印，前缀为空，缩进为 4 个空格
	encoder.SetIndent("", "    ")

	// 执行编码
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// 注意：encoder.Encode 默认会在末尾加一个换行符 \n
	// 为了行为与标准库 json.Marshal 保持一致（通常不自动带换行），这里手动去除
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
