package basic

type Article struct {
	BaseModel
	UserID    uint   `gorm:"not null;index" comment:"作者ID"`
	Title     string `gorm:"type:varchar(200);not null" comment:"文章标题"`
	Content   string `gorm:"type:longtext;not null" comment:"正文内容(Markdown或HTML)"`
	Summary   string `gorm:"type:varchar(300)" comment:"文章摘要"`
	Cover     string `gorm:"type:varchar(500)" comment:"封面图URL"`
	ViewCount int    `gorm:"default:0" comment:"阅读量"`
	IsTop     bool   `gorm:"default:false;index" comment:"是否置顶"`
}

func NewArticle(userID uint, title, content, summary, cover string) *Article {
	return &Article{
		UserID:  userID,
		Title:   title,
		Content: content,
		Summary: summary,
		Cover:   cover,
	}
}

func (Article) TableName() string {
	return "articles"
}

func (a Article) GetUserID() uint { return a.UserID }
