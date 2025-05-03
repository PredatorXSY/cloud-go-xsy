package service

import (
	"context"
	"log"
	"time"

	"common/client"
	pbOrder "proto/order"
	pbUser "proto/user"
)

// OrderService 订单服务
type OrderService struct {
	pbOrder.UnimplementedOrderServiceServer
	userClient *client.Client
}

// NewService 创建新的订单服务
func NewService() *OrderService {
	// 创建用户服务客户端
	userClient, err := client.NewClient(client.ClientConfig{
		ServiceName: "user-service",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		log.Printf("failed to create user service client: %v", err)
	}

	return &OrderService{
		userClient: userClient,
	}
}

// GetUserOrders 获取用户订单
func (s *OrderService) GetUserOrders(ctx context.Context, req *pbOrder.GetUserOrdersRequest) (*pbOrder.GetUserOrdersResponse, error) {
	// 验证用户是否存在
	if s.userClient != nil {
		userCtx, cancel := s.userClient.GetContext()
		defer cancel()

		userClient := pbUser.NewUserServiceClient(s.userClient.GetConn())
		_, err := userClient.GetUser(userCtx, &pbUser.GetUserRequest{
			UserId: req.UserId,
		})
		if err != nil {
			return nil, err
		}
	}

	// 获取用户订单
	orders := []*pbOrder.Order{
		{
			Id:     "1",
			UserId: req.UserId,
		},
		{
			Id:     "2",
			UserId: req.UserId,
		},
	}

	return &pbOrder.GetUserOrdersResponse{
		Orders: orders,
	}, nil
}

// Close 关闭服务
func (s *OrderService) Close() {
	if s.userClient != nil {
		s.userClient.Close()
	}
}
