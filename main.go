package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pborman/uuid"
)

type OrderStatus string

const (
	OrderPlaced     OrderStatus = "placed"
	OrderDispatched OrderStatus = "dispatched"
	OrderCompleted  OrderStatus = "completed"
	OrderReturned   OrderStatus = "returned"
	OrderCancelled  OrderStatus = "cancelled"
)

type Order struct {
	ID           string
	Discount     int64
	Amount       float64
	Status       OrderStatus
	DispatchedAt string
	CreatedAt    string
	UpdatedAt    string
}

// struct describing the items in the order
type OrderItem struct {
	ProductId       string
	ProductQuantity int64
	OrderId         string
}

var (
	orders     = make(map[string]Order)
	orderItems = make(map[string][]OrderItem)
)

func PingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("pong"))
}

func GetOrderItemsDetailsList(orderId string) ([]CreateOrderItemsResponse, error) {
	var orderItemsDetailsList []CreateOrderItemsResponse

	for _, item := range orderItems[orderId] {
		// call gRPC function to get the product details
		productDetails, err := GetProductDetails(item.ProductId)
		if err != nil {
			err := fmt.Errorf("product with id: %v, does not exist", item.ProductId)
			fmt.Println(err)
			return orderItemsDetailsList, err
		}

		// add the product details to the list
		orderItemsDetailsList = append(orderItemsDetailsList, CreateOrderItemsResponse{
			ID:          item.ProductId,
			Name:        productDetails.Name,
			Description: productDetails.Description,
			Category:    productDetails.Category,
			Price:       productDetails.Price,
			Quantity:    item.ProductQuantity,
		})
	}
	return orderItemsDetailsList, nil
}

type CreateOrderItemsRequest struct {
	ProductId string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

type CreateOrderRequest struct {
	Items []CreateOrderItemsRequest `json:"items"`
}

func (coReq *CreateOrderRequest) Validate() (err error) {
	if len(coReq.Items) == 0 {
		fmt.Println("items not provided")
		return errors.New("items not provided")
	}

	// Validate if product ids are repeated
	var uniqueItems []string
	for _, item := range coReq.Items {
		for _, product_id := range uniqueItems {
			if strings.ToLower(item.ProductId) == product_id {
				fmt.Println("product id is repeated")
				return errors.New("product id is repeated")
			}
		}
		uniqueItems = append(uniqueItems, strings.ToLower(item.ProductId))
	}

	for _, item := range coReq.Items {
		// Validate the product id
		if item.ProductId == "" {
			fmt.Println("invalid product id")
			return errors.New("invalid product id")
		}

		// Validate max product quantity is 10
		if !(item.Quantity > 0 && item.Quantity <= 10) {
			fmt.Println("product quantiy must be greater than 0 and less than eqaul to 10")
			return errors.New("product quantiy must be greater than 0 and less than equal to 10")
		}
	}

	return nil
}

type CreateOrderItemsResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Quantity    int64   `json:"quantity"`
}

type CreateOrderResponse struct {
	ID           string                     `json:"id"`
	Items        []CreateOrderItemsResponse `json:"items"`
	Discount     int64                      `json:"discount,omitempty"`
	Amount       float64                    `json:"amount"`
	Status       OrderStatus                `json:"status"`
	DispatchedAt string                     `json:"dispatched_at,omitempty"`
	CreatedAt    string                     `json:"created_at"`
	UpdatedAt    string                     `json:"updated_at"`
}

func PlaceOrderHandler(w http.ResponseWriter, r *http.Request) {
	var oReq CreateOrderRequest

	err := json.NewDecoder(r.Body).Decode(&oReq)
	if err != nil {
		fmt.Println("error unmashiling the request body, err:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Request Body"))
		return
	}

	if err = oReq.Validate(); err != nil {
		fmt.Println("error validating the request body, err:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	for _, item := range oReq.Items {
		// todo: use gRPC apis, get product details
		// todo: Validate if the product exists
		productDetails, err := GetProductDetails(item.ProductId)
		if err != nil {
			fmt.Println("product with id:", item.ProductId, "does not exist")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("product with id: %v does not exist", item.ProductId)))
			return
		}

		// todo: Validate if the inventory contains the required quantity
		if productDetails.Quantity < item.Quantity {
			fmt.Println("product with id:", item.ProductId, "does not have enough inventory")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("product with id: %v does not have enough inventory", item.ProductId)))
			return
		}
	}

	// create an order
	currentTime := time.Now().UTC().String()
	o := Order{
		ID:        uuid.New(),
		Status:    OrderPlaced,
		CreatedAt: currentTime,
		UpdatedAt: currentTime,
	}

	var orderAmount float64
	var numberOfPremiumProducts int64
	var oItems []OrderItem

	for _, item := range oReq.Items {
		// todo use gRPC apis, get product details
		productDetails, err := GetProductDetails(item.ProductId)
		if err != nil {
			fmt.Println("product with id:", item.ProductId, "does not exist while preparing order")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("product with id: %v does not exist while preparing order", item.ProductId)))
			return
		}

		// update the order amount
		orderAmount += productDetails.Price * float64(item.Quantity)

		// updated the counter if item is premium product
		if strings.ToLower(productDetails.Category) == "premium" {
			numberOfPremiumProducts += 1
		}

		// create order items
		oItems = append(oItems, OrderItem{
			ProductId:       item.ProductId,
			ProductQuantity: item.Quantity,
			OrderId:         o.ID,
		})
	}

	// todo: Provide a discount of 10% if order contains 3 premium product
	if numberOfPremiumProducts >= 3 {
		var discountInPercentage int64 = 10
		o.Discount = discountInPercentage

		orderAmount -= orderAmount * float64(discountInPercentage) / 100
		fmt.Println(orderAmount)
	}
	o.Amount = orderAmount

	// update the database
	orders[o.ID] = o
	orderItems[o.ID] = oItems
	fmt.Println("success creating the order:", o, "with items:", oItems)

	// update the product quantity in the inventory
	for _, item := range oReq.Items {
		// todo call gRPC service to get the product details
		productDetails, err := GetProductDetails(item.ProductId)
		if err != nil {
			fmt.Println("product with id:", item.ProductId, "does not exist while updating product quantity in the order inventory")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf("product with id: %v does not exist while updating product quantity in the order inventory", item.ProductId)))
			return
		}
		if err := UpdateProductQuantity(item.ProductId, productDetails.Quantity-item.Quantity); err != nil {
			fmt.Println("inventory for product with id:", item.ProductId, "could not be updated")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("inventory for product with id: %v could not be updated", item.ProductId)))
			return
		}
	}
	fmt.Println("success updating the product inventory")

	// Create the response
	oResp := CreateOrderResponse{
		ID:        o.ID,
		Discount:  o.Discount,
		Amount:    o.Amount,
		Status:    o.Status,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
	// Get the product details
	orderItemsDetailsList, err := GetOrderItemsDetailsList(o.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	oResp.Items = orderItemsDetailsList

	resp, err := json.Marshal(oResp)
	if err != nil {
		fmt.Println("error mashiling the response, err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	var orderList []CreateOrderResponse

	for _, o := range orders {
		orderDetails := CreateOrderResponse{
			ID:           o.ID,
			Discount:     o.Discount,
			Amount:       o.Amount,
			Status:       o.Status,
			DispatchedAt: o.DispatchedAt,
			CreatedAt:    o.CreatedAt,
			UpdatedAt:    o.UpdatedAt,
		}

		// Get the item details
		orderItemsDetailsList, err := GetOrderItemsDetailsList(o.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		orderDetails.Items = orderItemsDetailsList

		orderList = append(orderList, orderDetails)
	}

	resp, err := json.Marshal(orderList)
	if err != nil {
		fmt.Println("error mashiling the response, err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GetOrderDetailsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId := vars["order_id"]

	o, ok := orders[orderId]

	// Verify if the order is present in the database
	if !ok {
		fmt.Println("order with id:", orderId, "does not exist")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("order with id: %v does not exist", orderId)))
		return
	}

	// Prepare the response
	orderDetails := CreateOrderResponse{
		ID:           o.ID,
		Discount:     o.Discount,
		Amount:       o.Amount,
		Status:       o.Status,
		DispatchedAt: o.DispatchedAt,
		CreatedAt:    o.CreatedAt,
		UpdatedAt:    o.UpdatedAt,
	}

	// Get the item details
	orderItemsDetailsList, err := GetOrderItemsDetailsList(o.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	orderDetails.Items = orderItemsDetailsList

	resp, err := json.Marshal(orderDetails)
	if err != nil {
		fmt.Println("error mashiling the response, err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

type UpdateOrderStatusRequest struct {
	Status OrderStatus `json:"status"`
}

func (u *UpdateOrderStatusRequest) Validate() (err error) {
	switch u.Status {
	case OrderPlaced, OrderDispatched, OrderCompleted, OrderReturned, OrderCancelled:
	default:
		fmt.Println("invalid order status")
		return errors.New("invalid order status")
	}
	return nil
}

func UpdateOrderStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderId := vars["order_id"]

	var updateStatusReq UpdateOrderStatusRequest
	err := json.NewDecoder(r.Body).Decode(&updateStatusReq)
	if err != nil {
		fmt.Println("error unmashiling the request body, err:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid Request Body"))
		return
	}

	if err = updateStatusReq.Validate(); err != nil {
		fmt.Println("error validating the request body, err:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	o, ok := orders[orderId]
	// Verify if the order is present in the database
	if !ok {
		fmt.Println("order with id:", orderId, "does not exist")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("order with id: %v does not exist", orderId)))
		return
	}

	// todo validate if the status can be updated to the required status
	orderStatusMap := map[OrderStatus]int64{
		OrderPlaced:     1,
		OrderDispatched: 2,
		OrderCompleted:  3,
		OrderReturned:   4,
		OrderCancelled:  5,
	}
	currentOrderStatusRank := orderStatusMap[o.Status]
	newOrderStatusRank := orderStatusMap[updateStatusReq.Status]
	switch {
	case newOrderStatusRank <= currentOrderStatusRank:
		fmt.Println("order status can be updated to a lower or the same status")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order status can be updated to a lower or the same status"))
		return

	case newOrderStatusRank == 3 && currentOrderStatusRank != 2:
		fmt.Println("order cannot be completed until it is dispatched")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order cannot be completed until it is dispatched"))
		return

	case newOrderStatusRank == 4 && currentOrderStatusRank != 3:
		fmt.Println("order cannot be returned until it is completed")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order cannot be returned until it is completed"))
		return

	case newOrderStatusRank == 5 && currentOrderStatusRank > 2:
		fmt.Println("order cannot be cancelled once it is completed or returned")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order cannot be cancelled once it is completed or returned"))
		return
	}

	// update the order status
	o.Status = updateStatusReq.Status
	if updateStatusReq.Status == OrderDispatched {
		o.DispatchedAt = time.Now().UTC().String()
	}

	// Update the database
	fmt.Println("updating order:", o.ID, "status from:", o.Status, "to: ", updateStatusReq.Status)
	orders[o.ID] = o

	// Prepare the response
	orderDetails := CreateOrderResponse{
		ID:           o.ID,
		Amount:       o.Amount,
		Status:       o.Status,
		DispatchedAt: o.DispatchedAt,
		CreatedAt:    o.CreatedAt,
		UpdatedAt:    o.UpdatedAt,
	}

	// Get the product details
	orderItemsDetailsList, err := GetOrderItemsDetailsList(o.ID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	orderDetails.Items = orderItemsDetailsList

	resp, err := json.Marshal(orderDetails)
	if err != nil {
		fmt.Println("error mashiling the response, err:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func main() {
	createProductGRPCClientConnection()

	fmt.Println("Staring rest api server")

	r := mux.NewRouter()
	r.HandleFunc("/ping", PingHandler).Methods(http.MethodGet)

	s := r.PathPrefix("/orders").Subrouter()
	s.HandleFunc("", PlaceOrderHandler).Methods(http.MethodPost)
	s.HandleFunc("", GetOrdersHandler).Methods(http.MethodGet)
	s.HandleFunc("/{order_id}", GetOrderDetailsHandler).Methods(http.MethodGet)
	s.HandleFunc("/{order_id}/status", UpdateOrderStatusHandler).Methods(http.MethodPut)

	http.ListenAndServe(":8081", r)
}
