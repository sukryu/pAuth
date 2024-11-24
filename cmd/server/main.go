package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sukryu/pAuth/internal/config"
	"github.com/sukryu/pAuth/pkg/apis/handlers"
	"github.com/sukryu/pAuth/pkg/apis/router"
	"github.com/sukryu/pAuth/pkg/controllers"
	"github.com/sukryu/pAuth/pkg/utils/jwt"
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
	rbacController := controllers.NewRBACController(store)

	// JWT 매니저 초기화
	jwtManager := jwt.NewJWTManager(
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.TokenExpiration)*time.Hour,
	)

	// 핸들러 초기화
	authHandler := handlers.NewAuthHandler(authController, jwtManager, rbacController)

	// 라우터 초기화
	r := router.NewRouter(authHandler, jwtManager, rbacController)
	engine := r.Setup()

	// 서버 시작
	log.Printf("Server starting on %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := engine.Run(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
