package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"common/server"
	"google.golang.org/grpc"
	"proto/user"
	"user-service/internal"
)

// loggingInterceptor 日志拦截器
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("method: %s, duration: %v, error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}

func main() {
	// 创建用户服务实例
	userService := service.NewService()

	// 配置服务
	config := server.ServiceConfig{
		Name:        "user-service",
		Port:        "", // 使用动态端口
		ServiceDesc: &user.UserService_ServiceDesc,
		Impl:        userService,
		Options: []grpc.ServerOption{
			server.WithUnaryInterceptor(loggingInterceptor),
		},
	}

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	// 启动服务
	go server.StartServer(config)

	// 等待关闭信号
	<-stop
	log.Println("shutting down user service...")

	// 清理资源
	userService.Close()
}
