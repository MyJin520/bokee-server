package request

type ActionCreateReq struct {
	UserID     uint   `json:"-"`
	TargetID   uint   `json:"targetId" label:"目标ID" validate:"required"`
	ActionType string `json:"actionType" label:"操作类型" validate:"required,oneof=like bookmark follow"`
	TargetType string `json:"targetType" label:"目标类型" validate:"required,oneof=article author"`
}

type ActionListReq struct {
	PageReq
	ActionType string `json:"actionType" label:"操作类型" validate:"required,oneof=like bookmark follow"`
	TargetType string `json:"targetType" label:"目标类型" validate:"required,oneof=article author"`
}
