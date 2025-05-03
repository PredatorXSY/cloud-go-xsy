package client

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	ServiceName string        // 服务名称
	Address     string        // 服务地址
	Timeout     time.Duration // 超时时间
}

// Client 客户端
type Client struct {
	conn   *grpc.ClientConn
	config ClientConfig
}

// NewClient 创建新的客户端
func NewClient(config ClientConfig) (*Client, error) {
	// 设置默认超时时间
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	// 构建服务地址
	if config.Address == "" {
		config.Address = config.ServiceName + ":50051"
	}

	// 创建gRPC连接
	conn, err := grpc.Dial(
		config.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		config: config,
	}, nil
}

// GetConn 获取gRPC连接
func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭连接
func (c *Client) Close() error {
	return c.conn.Close()
}

// GetContext 获取带超时的上下文
func (c *Client) GetContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.config.Timeout)
}
