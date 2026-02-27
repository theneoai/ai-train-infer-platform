package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null;size:255"`
	Name      string         `json:"name" gorm:"not null;size:255"`
	Password  string         `json:"-" gorm:"not null;size:255"`
	Role      string         `json:"role" gorm:"default:'user';size:50"`
	OrgID     *uuid.UUID     `json:"org_id" gorm:"type:uuid;index"`
	Avatar    string         `json:"avatar" gorm:"size:500"`
	Status    string         `json:"status" gorm:"default:'active';size:50"`
	LastLogin *time.Time     `json:"last_login"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Organization *Organization `json:"organization,omitempty" gorm:"foreignKey:OrgID"`
	Projects     []Project     `json:"projects,omitempty" gorm:"foreignKey:OwnerID"`
	Experiments  []Experiment  `json:"experiments,omitempty" gorm:"foreignKey:UserID"`
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// Organization 组织模型
type Organization struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"not null;size:255"`
	Description string         `json:"description" gorm:"size:1000"`
	Plan        string         `json:"plan" gorm:"default:'free';size:50"`
	Quota       JSON           `json:"quota" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"default:'active';size:50"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Users []User `json:"users,omitempty" gorm:"foreignKey:OrgID"`
}

// BeforeCreate 创建前钩子
func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Organization) TableName() string {
	return "organizations"
}

// Project 项目模型
type Project struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"not null;size:255"`
	Description string         `json:"description" gorm:"size:1000"`
	OwnerID     uuid.UUID      `json:"owner_id" gorm:"type:uuid;not null;index"`
	OrgID       *uuid.UUID     `json:"org_id" gorm:"type:uuid;index"`
	Config      JSON           `json:"config" gorm:"type:jsonb;default:'{}'"`
	Status      string         `json:"status" gorm:"default:'active';size:50"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Owner       *User         `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Organization *Organization `json:"organization,omitempty" gorm:"foreignKey:OrgID"`
	Experiments []Experiment  `json:"experiments,omitempty" gorm:"foreignKey:ProjectID"`
	Models      []Model       `json:"models,omitempty" gorm:"foreignKey:ProjectID"`
}

// BeforeCreate 创建前钩子
func (p *Project) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// TableName 表名
func (Project) TableName() string {
	return "projects"
}

// JSON JSON 类型
type JSON map[string]interface{}
