package service

import (
	"context"
	"log"
	"time"

	"common/client"
	pbOrder "proto/order"
	pbUser "proto/user"
)

// UserService 用户服务
type UserService struct {
	pbUser.UnimplementedUserServiceServer
	orderClient *client.Client
}

// NewService 创建新的用户服务
func NewService() *UserService {
	// 创建订单服务客户端
	orderClient, err := client.NewClient(client.ClientConfig{
		ServiceName: "order-service",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		log.Printf("failed to create order service client: %v", err)
	}

	return &UserService{
		orderClient: orderClient,
	}
}

// GetUser 获取用户信息
func (s *UserService) GetUser(ctx context.Context, req *pbUser.GetUserRequest) (*pbUser.GetUserResponse, error) {
	// 获取用户信息
	user := &pbUser.User{
		Id:   req.UserId,
		Name: "User " + req.UserId,
	}

	// 调用订单服务获取用户的订单
	if s.orderClient != nil {
		orderCtx, cancel := s.orderClient.GetContext()
		defer cancel()

		orderClient := pbOrder.NewOrderServiceClient(s.orderClient.GetConn())
		orders, err := orderClient.GetUserOrders(orderCtx, &pbOrder.GetUserOrdersRequest{
			UserId: req.UserId,
		})
		if err != nil {
			log.Printf("failed to get user orders: %v", err)
		} else {
			user.Orders = orders.Orders
		}
	}

	return &pbUser.GetUserResponse{
		User: user,
	}, nil
}

// Close 关闭服务
func (s *UserService) Close() {
	if s.orderClient != nil {
		s.orderClient.Close()
	}
}
