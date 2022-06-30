package main

import (
	"nats-streaming-subscriber/publisher/mocdata"
	"nats-streaming-subscriber/subscriber/database"
)

func main() {
	db := database.ConnectDB()
	database.AddNewOrder(db.OrdersStorage, mocdata.TestOrder)
}
