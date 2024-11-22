package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/internal/store/memory"
	"github.com/sukryu/pAuth/pkg/apis/handlers"
	"github.com/sukryu/pAuth/pkg/controllers"
)

func main() {
	// 설정 로드
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 스토어 초기화
	store := memory.NewMemoryStore()

	// 컨트롤러 초기화
	authController := controllers.NewAuthController(store)

	// 핸들러 초기화
	authHandler := handlers.NewAuthHandler(authController)

	// Gin 라우터 설정
	router := gin.Default()

	// 핸들러 등록
	authHandler.Register(router)

	// 서버 시작
	log.Printf("Server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := router.Run(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
