package request

type ConversationCreateReq struct {
	TargetID uint `json:"targetId" label:"对方用户ID" validate:"required"`
}

type ConversationListReq struct {
	PageReq
}
