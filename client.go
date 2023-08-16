package main

import (
	"context"
	"fmt"
	"log"

	"github.com/microServicesExamples/gRPC/product/productpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var conn productpb.ProductServiceClient

func createProductGRPCClientConnection() {
	fmt.Println("Initiating the gRPC client connection")

	// create a client connection
	cc, err := grpc.Dial("localhost:5051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to created client stub: %v", err)
	}
	// defer cc.Close()

	// create the product service client connection
	conn = productpb.NewProductServiceClient(cc)
}

func GetProductDetails(productId string) (*productpb.GetProductDetailsResponse, error) {
	fmt.Println("Get product details via gRPC function")

	// prepare the request
	req := &productpb.GetProductDetailsRequest{
		Id: productId,
	}

	// execute the rpc function
	resp, err := conn.GetProductDetails(context.Background(), req)
	if err != nil {
		fmt.Printf("error serving the request: %v\n", err)
		return resp, fmt.Errorf("error serving the request: %v", err)
	}

	// display the response
	fmt.Printf("The product details are %+v\n", resp)

	return resp, nil
}

func ListProductDetails(productIds []string) (*productpb.ListProductDetailsResponse, error) {
	fmt.Println("Get product details list via gRPC function")

	// prepare the request
	var productIdsReq []*productpb.GetProductDetailsRequest
	for _, productId := range productIds {
		productIdsReq = append(productIdsReq, &productpb.GetProductDetailsRequest{
			Id: productId,
		})
	}
	req := &productpb.ListProductDetailsRequest{
		Ids: productIdsReq,
	}

	// execute the rpc function
	resp, err := conn.ListProductDetails(context.Background(), req)
	if err != nil {
		fmt.Printf("error serving the request: %v\n", err)
		return &productpb.ListProductDetailsResponse{}, fmt.Errorf("error serving the request: %v", err)
	}

	// display the response
	fmt.Printf("The product details are %+v\n", resp)
	return resp, nil
}

func UpdateProductQuantity(productId string, quantity int64) error {
	fmt.Println("Update product quantity via gRPC function")

	// prepare the request
	req := &productpb.UpdateProductQuantityRequest{
		Id:       productId,
		Quantity: quantity,
	}

	// execute the rpc function
	resp, err := conn.UpdateProductQuantity(context.Background(), req)
	if err != nil {
		fmt.Printf("error serving the request: %v\n", err)
		return fmt.Errorf("error serving the request: %v", err)
	}

	// display the response
	fmt.Println("Updated the product details:", resp)
	return nil
}
