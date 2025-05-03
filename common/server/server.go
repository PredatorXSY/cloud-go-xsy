package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name        string // 服务名称
	Port        string // 服务端口
	ServiceDesc *grpc.ServiceDesc
	Impl        interface{}
	Options     []grpc.ServerOption // gRPC服务器选项
}

// StartServer 启动gRPC服务器
func StartServer(config ServiceConfig) {
	// 创建gRPC服务器
	grpcServer := grpc.NewServer(config.Options...)

	// 注册健康检查服务
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus(config.Name, grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册服务
	grpcServer.RegisterService(config.ServiceDesc, config.Impl)

	// 注册反射服务，用于grpcurl等工具
	reflection.Register(grpcServer)

	// 使用动态端口
	port := config.Port
	if port == "" {
		port = "0" // 0表示让系统自动分配端口
	}

	// 启动gRPC服务器
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 获取实际监听的端口
	addr := lis.Addr().(*net.TCPAddr)
	log.Printf("%s service is running on port %d", config.Name, addr.Port)

	// 在Kubernetes中注册服务
	if err := registerServiceInK8s(config.Name, addr.Port); err != nil {
		log.Printf("failed to register service in k8s: %v", err)
	}

	// 优雅关闭
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	// 启动服务
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// 等待关闭信号
	<-stop
	log.Printf("shutting down %s service...", config.Name)

	// 从Kubernetes中注销服务
	if err := unregisterServiceFromK8s(config.Name); err != nil {
		log.Printf("failed to unregister service from k8s: %v", err)
	}

	// 设置健康状态为NOT_SERVING
	healthServer.SetServingStatus(config.Name, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// 优雅关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 停止接受新的连接
	grpcServer.GracefulStop()

	// 等待所有现有连接处理完成
	<-ctx.Done()
	log.Printf("%s service shutdown completed", config.Name)
}

// registerServiceInK8s 在Kubernetes中注册服务
func registerServiceInK8s(serviceName string, port int) error {
	// 获取Pod名称
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		return fmt.Errorf("POD_NAME environment variable not set")
	}

	// 获取Pod IP
	podIP := os.Getenv("POD_IP")
	if podIP == "" {
		return fmt.Errorf("POD_IP environment variable not set")
	}

	// 获取命名空间
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// 构建DNS名称
	dnsName := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)
	log.Printf("registering service %s with DNS name %s", serviceName, dnsName)

	// 创建Kubernetes客户端
	config, err := rest.InClusterConfig()
	if err != nil {
		// 如果在集群外运行，尝试使用kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create k8s config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %v", err)
	}

	// 创建或更新Endpoints
	endpoints := clientset.CoreV1().Endpoints(namespace)
	_, err = endpoints.Get(context.TODO(), serviceName, metav1.GetOptions{})
	if err != nil {
		// 如果Endpoints不存在，创建新的
		_, err = endpoints.Create(context.TODO(), &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: podIP,
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: int32(port),
						},
					},
				},
			},
		}, metav1.CreateOptions{})
	} else {
		// 如果Endpoints已存在，更新它
		_, err = endpoints.Update(context.TODO(), &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{
							IP: podIP,
						},
					},
					Ports: []corev1.EndpointPort{
						{
							Port: int32(port),
						},
					},
				},
			},
		}, metav1.UpdateOptions{})
	}

	if err != nil {
		return fmt.Errorf("failed to create/update endpoints: %v", err)
	}

	return nil
}

// unregisterServiceFromK8s 从Kubernetes中注销服务
func unregisterServiceFromK8s(serviceName string) error {
	// 获取命名空间
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}

	// 创建Kubernetes客户端
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to create k8s config: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %v", err)
	}

	// 删除Endpoints
	endpoints := clientset.CoreV1().Endpoints(namespace)
	err = endpoints.Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete endpoints: %v", err)
	}

	return nil
}

// WithUnaryInterceptor 添加一元拦截器
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.UnaryInterceptor(interceptor)
}

// WithStreamInterceptor 添加流拦截器
func WithStreamInterceptor(interceptor grpc.StreamServerInterceptor) grpc.ServerOption {
	return grpc.StreamInterceptor(interceptor)
}
