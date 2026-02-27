module github.com/plucky-groove3/ai-train-infer-platform/services/experiment

go 1.21

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/google/uuid v1.6.0
	github.com/plucky-groove3/ai-train-infer-platform/pkg v0.0.0
	go.uber.org/zap v1.27.1
	gorm.io/driver/postgres v1.5.6
	gorm.io/gorm v1.31.1
)

replace github.com/plucky-groove3/ai-train-infer-platform/pkg => ../../pkg
