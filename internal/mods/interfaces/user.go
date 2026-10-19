package interfaces

// Ownable 定义"归属于某个用户"的资源接口，用于统一校验资源所有权
type Ownable interface {
	GetUserID() uint
}
