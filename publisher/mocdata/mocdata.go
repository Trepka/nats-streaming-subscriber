package mocdata

import (
	"nats-streaming-subscriber/datastruct"
)

var TestOrder = datastruct.Order{
	OrderUID:          "b563feb7b2b84b6test",
	TrackNumber:       "WBILMTESTTRACK",
	Entry:             "WBIL",
	Delivery:          TestDelivery,
	Payment:           TestPayment,
	Items:             []datastruct.Item{TestItem1, TestItem2},
	Locale:            "en",
	InternalSignature: "",
	CustomerID:        "test",
	DeliveryService:   "meest",
	Shardkey:          "9",
	SmID:              99,
	DateCreated:       "2021-11-26T06:22:19Z",
	OofShard:          "1",
}

var TestDelivery = datastruct.Delivery{
	Name:    "Test Testov",
	Phone:   "+9720000000",
	Zip:     "2639809",
	City:    "Kiryat Mozkin",
	Address: "Ploshad Mira 15",
	Region:  "Kraiot",
	Email:   "test@gmail.com",
}

var TestPayment = datastruct.Payment{
	Transaction:  "b563feb7b2b84b6test",
	RequestID:    "",
	Currency:     "USD",
	Provider:     "wbpay",
	Amount:       1917,
	PaymentDT:    1637907727,
	Bank:         "alpha",
	DeliveryCost: 1500,
	GoodsTotal:   317,
	CustomFee:    0,
}

var TestItem1 = datastruct.Item{
	ChartID:     9934930,
	TrackNumber: "WBILMTESTTRACK",
	Price:       453,
	Rid:         "ab4219087a764ae0btest",
	Name:        "Mascaras",
	Sale:        30,
	Size:        "0",
	TotalPrice:  317,
	NmID:        2389212,
	Brand:       "Vivienne Sabo",
	Status:      202,
}

var TestItem2 = datastruct.Item{
	ChartID:     342662,
	TrackNumber: "WBILMTESTTRACK",
	Price:       315,
	Rid:         "asory84kshf87k",
	Name:        "Sweet bubaleh",
	Sale:        23,
	Size:        "0",
	TotalPrice:  456,
	NmID:        5443632,
	Brand:       "Faberlic",
	Status:      202,
}
