package main

import (
	"context"
	"log"
	"time"

	"common/server"
	"google.golang.org/grpc"
	"order-service/internal"
	pbOrder "proto/order"
)

// loggingInterceptor 日志拦截器
func loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("method: %s, duration: %v, error: %v", info.FullMethod, time.Since(start), err)
	return resp, err
}

func main() {
	// 创建订单服务实例
	orderService := service.NewService()

	// 配置服务
	config := server.ServiceConfig{
		Name:        "order-service",
		Port:        "", // 使用动态端口
		ServiceDesc: &pbOrder.OrderService_ServiceDesc,
		Impl:        orderService,
		Options: []grpc.ServerOption{
			server.WithUnaryInterceptor(loggingInterceptor),
		},
	}

	// 启动服务
	server.StartServer(config)
}
