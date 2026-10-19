package response

type UserActionListResponse struct {
	ActionID    uint   `json:"actionId"`
	TargetType  string `json:"targetType"`
	TargetID    uint   `json:"targetId"`
	TargetTitle string `json:"targetTitle"`
}
