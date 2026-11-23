package request

type MessageSendReq struct {
	TargetID    uint   `json:"targetId" label:"对方用户ID" validate:"required"`
	MsgType     string `json:"msgType" label:"消息类型" validate:"omitempty,oneof=text"`
	Content     string `json:"content" label:"消息内容" validate:"required"` // 长度上限由 im.max-content-length 配置统一校验
	ClientMsgID string `json:"clientMsgId" label:"客户端消息ID" validate:"required,max=64"`
}

type MessageListReq struct {
	ConversationID uint `json:"conversationId" label:"会话ID" validate:"required"`
	LastID         uint `json:"lastId" label:"上一页最旧消息ID"`
	PageSize       int  `json:"pageSize" label:"每页数量" validate:"omitempty,min=1,max=100"`
}

type MessageReadReq struct {
	ConversationID uint `json:"conversationId" label:"会话ID" validate:"required"`
	LastReadID     uint `json:"lastReadId" label:"最后已读消息ID"` // 0 表示读到最新
}
