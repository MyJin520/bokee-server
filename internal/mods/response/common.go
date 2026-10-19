package response

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	SUCCESS = 0
	ERROR   = 500 // 通用业务错误
)

// Response 基础 JSON 响应体，Data 支持泛型
type Response[T any] struct {
	Code int    `json:"code"`
	Data T      `json:"data"`
	Msg  string `json:"msg"`
}

// PageData 分页数据结构
type PageData[T any] struct {
	List     T     `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// Result 核心返回方法，状态码 200，业务码由 code 控制
func Result[T any](code int, data T, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, Response[T]{
		Code: code,
		Data: data,
		Msg:  msg,
	})
}

// Ok 成功，无数据
func Ok(c *gin.Context) {
	Result(SUCCESS, struct{}{}, "操作成功", c)
}

// OkWithMessage 成功，自定义消息
func OkWithMessage(msg string, c *gin.Context) {
	Result(SUCCESS, struct{}{}, msg, c)
}

// OkWithData 成功，返回数据
// Deprecated: 已废弃，请使用 OkWithDetailed 替代
func OkWithData[T any](data T, msg string, c *gin.Context) {
	OkWithDetailed(data, msg, c)
}

// OkWithDetailed 成功，返回数据和自定义消息
func OkWithDetailed[T any](data T, msg string, c *gin.Context) {
	Result(SUCCESS, data, msg, c)
}

// OkWithPage 分页成功
func OkWithPage[T any](list T, total int64, page, pageSize int, msg string, c *gin.Context) {
	Result(SUCCESS, PageData[T]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, msg, c)
}

// Fail 失败，自定义错误码与消息
func Fail(code int, msg string, c *gin.Context) {
	Result(code, struct{}{}, msg, c)
}

// FailWithRequest 请求体参数错误
func FailWithRequest(msg string, c *gin.Context) {
	Result(http.StatusBadRequest, struct{}{}, msg, c)
}

// FailWithMessage 通用失败，使用默认 ERROR 码
func FailWithMessage(msg string, c *gin.Context) {
	Result(ERROR, struct{}{}, msg, c)
}

// SetupStreamHeaders 设置 SSE / 流式响应头，调用后需手动 Flush
func SetupStreamHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	c.Writer.Flush()
}

// StreamSSE 发送 SSE 事件，会自动 Flush
func StreamSSE(event string, data any, c *gin.Context) {
	c.SSEvent(event, data)
	c.Writer.Flush()
}

// StreamData 写入原始字节流，并立即 Flush
func StreamData(data []byte, c *gin.Context) {
	_, _ = c.Writer.Write(data)
	c.Writer.Flush()
}

// DownloadFile 以附件形式下载文件
func DownloadFile(filepath, filename string, c *gin.Context) {
	c.FileAttachment(filepath, filename)
}

// InlineFile 内联预览文件（不强制下载）
func InlineFile(filepath string, c *gin.Context) {
	c.File(filepath)
}

// DownloadData 从内存数据创建下载响应
func DownloadData(data []byte, filename, contentType string, c *gin.Context) {
	c.Header("Content-Disposition", "attachment; filename="+url.QueryEscape(filename))
	c.Data(http.StatusOK, contentType, data)
}
