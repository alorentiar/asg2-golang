package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type Order struct {
	ID           int64       `json:"id"`
	OrderedAt    string      `json:"orderedAt"`
	CustomerName string      `json:"customerName"`
	Items        []OrderItem `json:"items"`
}

type OrderItem struct {
	ItemCode    string `json:"itemCode"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
}

const (
	host     = "localhost"
	port     = 5432
	username = "postgres"
	password = ""
	dbName   = "myorders"
)

var (
	db  *sql.DB
	err error
)

func init() {
	var err error
	psqlURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, username, password, dbName)
	db, err = sql.Open("postgres", psqlURL) // Adjust connection string as needed
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to database successfully")
}

func main() {
	router := gin.Default()

	router.POST("/orders", createOrder)
	router.GET("/orders", getOrders)
	router.PUT("/orders/:id", updateOrder)
	router.DELETE("/orders/:id", deleteOrder)

	router.Run(":8080")
}

func createOrder(c *gin.Context) {
	var order Order
	if err := c.BindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		INSERT INTO orders (ordered_at, customer_name)
		VALUES ($1, $2)
		RETURNING id
	`, order.OrderedAt, order.CustomerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	print(result)
	var newOrderID int64
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	order.ID = newOrderID

	for _, item := range order.Items {
		_, err := db.Exec(`
			INSERT INTO order_items (order_id, item_code, description, quantity)
			VALUES ($1, $2, $3, $4)
		`, newOrderID, item.ItemCode, item.Description, item.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, order)
}

func getOrders(c *gin.Context) {
	var orders []Order

	rows, err := db.Query(`
	  SELECT o.id, o.ordered_at, o.customer_name, oi.item_code, oi.description, oi.quantity
	  FROM orders o
	  LEFT JOIN order_items oi ON o.id = oi.order_id
	  ORDER BY o.id ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var order Order
		var item OrderItem
		err := rows.Scan(&order.ID, &order.OrderedAt, &order.CustomerName, &item.ItemCode, &item.Description, &item.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if item.ItemCode != "" {
			order.Items = append(order.Items, item)
		}
	}

	if len(orders) == 0 {
		c.JSON(http.StatusOK, []Order{})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func updateOrder(c *gin.Context) {
	var orderID int64

	var updatedOrder Order
	if err := c.BindJSON(&updatedOrder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
	  UPDATE orders
	  SET ordered_at = $1, customer_name = $2
	  WHERE id = $3
	`, updatedOrder.OrderedAt, updatedOrder.CustomerName, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, updatedOrder)
}

func deleteOrder(c *gin.Context) {
	var orderID int64

	result, err := db.Exec(`
	  DELETE FROM orders
	  WHERE id = $1
	`, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success delete"})
}
