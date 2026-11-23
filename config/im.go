package config

type IMConfig struct {
	StrangerQuota    int      `yaml:"stranger-quota"`       // 陌生会话发起额度
	MaxContentLength int      `yaml:"max-content-length"`   // 单次消息最大字符数
	SendRateMax      int      `yaml:"send-rate-max"`        // 限流窗口内最大发送条数
	SendRateWindow   int      `yaml:"send-rate-window"`     // 限流窗口（秒）
	WsMaxConnPerUser int      `yaml:"ws-max-conn-per-user"` // 单用户最大 WebSocket 连接数
	WsSendBuffer     int      `yaml:"ws-send-buffer"`       // 每次链接的发送缓冲大小
	WsPingInterval   int      `yaml:"ws-ping-interval"`     // WebSocket 心跳间隔（秒）
	WsTicketTTL      int      `yaml:"ws-ticket-ttl"`        //WebSocket 一次性票据有效期（秒）
	WsOriginPatterns []string `yaml:"ws-origin-patterns"`   // 允许的 Origin（host 或 host:port，支持 *.example.com）
}
