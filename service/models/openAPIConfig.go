package models

// OpenAPIConfig stores the browser-extension credential for one user.
type OpenAPIConfig struct {
	BaseModel
	UserID  uint   `gorm:"uniqueIndex;not null" json:"userId"`
	Token   string `gorm:"type:varchar(64);uniqueIndex" json:"token"`
	Enabled bool   `gorm:"type:tinyint(1);default:0" json:"enabled"`
	User    User   `gorm:"foreignKey:UserID;references:ID" json:"-"`
}
