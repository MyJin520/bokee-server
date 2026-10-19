package config

type OssConfig struct {
	LocalOss LocalOss `yaml:"local-oss"` // 本地存储
	MinioOss MinioOss `yaml:"minio-oss"` // Minio 存储
	AliOss   AliOss   `yaml:"ali-oss"`   // 阿里云 OSS
}

type LocalOss struct {
	Path string `yaml:"path"` // 存储路径
}

type MinioOss struct {
	Host            string `yaml:"host"`              // minio服务地址
	AccessKeyID     string `yaml:"access-key-id"`     // minio用户名
	SecretAccessKey string `yaml:"access-key-secret"` // minion用户密码
	BucketName      string `yaml:"bucket-name"`       // 桶名
	UseSSL          bool   `yaml:"use-ssl"`           // 是否使用ssl证书
	FileUrl         string `yaml:"file-url"`          // 对外访问文件的地址
}

type AliOss struct {
	Url string `yaml:"url"` // 服务地址
}
