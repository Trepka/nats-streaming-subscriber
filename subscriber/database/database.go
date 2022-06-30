package database

import (
	"fmt"
	"log"
	"nats-streaming-subscriber/datastruct"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgressOrdersStorage struct {
	OrdersStorage *sqlx.DB
}

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "orders"
)

var schema = `
CREATE TABLE IF NOT EXISTS deliveries(
	delivery_id INT GENERATED ALWAYS AS IDENTITY,
	name TEXT NOT NULL,
	phone TEXT NOT NULL,
	zip TEXT NOT NULL,
	city TEXT NOT NULL,
	address TEXT NOT NULL,
	region TEXT NOT NULL,
	email TEXT NOT NULL,
	PRIMARY KEY (delivery_id)
);

CREATE TABLE IF NOT EXISTS payments(
	payment_id INT GENERATED ALWAYS AS IDENTITY,
	transaction TEXT UNIQUE NOT NULL,
	request_id TEXT NOT NULL,
	currency TEXT NOT NULL,
	provider TEXT NOT NULL,
	amount INT NOT NULL,
	payment_dt INT NOT NULL,
	bank TEXT NOT NULL,
	delivery_cost INT NOT NULL,
	good_total INT NOT NULL,
	custom_fee INT NOT NULL,
	PRIMARY KEY (payment_id)
);

CREATE TABLE IF NOT EXISTS items(
	item_id INT GENERATED ALWAYS AS IDENTITY,
	chrt_id INT NOT NULL,
	track_number VARCHAR(30) NOT NULL,
	price INT NOT NULL,
	rid VARCHAR(30) NOT NULL,
	name VARCHAR(30) NOT NULL,
	sale SMALLINT NOT NULL,
	size TEXT NOT NULL,
	total_price INT NOT NULL,
	nm_id INT NOT NULL,
	brand TEXT,
	status INT NOT NULL,
	PRIMARY KEY (item_id)
);

CREATE TABLE IF NOT EXISTS orders(
	order_id INT GENERATED ALWAYS AS IDENTITY,
	order_uid TEXT UNIQUE,
	track_number TEXT UNIQUE,
	entry TEXT NOT NULL,
	delivery_id INT REFERENCES delivery ON DELETE CASCADE,
	payment_id INT REFERENCES payment ON DELETE CASCADE,
	items INT[] NOT NULL DEFAULT '{}'::INT[],
	FOREIGN KEY (EACH ELEMENT OF items) REFERENCES items,
	locale TEXT NOT NULL,
	internal_signature TEXT NOT NULL,
	customer_id TEXT NOT NULL,
	delivery_service TEXT NOT NULL,
	shardkey TEXT NOT NULL,
	sm_id INT NOT NULL,
	date_created TEXT NOT NULL,
	oof_shard TEXT NOT NULL,
	PRIMARY KEY (order_id)
);
`

func ConnectDB() PostgressOrdersStorage {
	var storage PostgressOrdersStorage
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatal("Database connection failed")
	}

	db.MustExec(schema)

	storage.OrdersStorage = db
	return storage
}

func AddNewOrder(db *sqlx.DB, order datastruct.Order) {
	tx, err := db.Beginx()
	if err != nil {
		log.Fatal("can't begin transaction")
		return
	}

	var deliveryID int
	err = tx.QueryRowx("INSERT INTO deliveries (name, phone, zip, city, address, region, email) Values ($1, $2, $3, $4, $5, $6, $7) RETURNING delivery_id", order.Delivery.Name,
		order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email).Scan(&deliveryID)
	if err != nil {
		tx.Rollback()
		log.Fatal("delivery data error")
		return
	}

	var paymentID int
	err = tx.QueryRowx("INSERT INTO payments (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, good_total, custom_fee) Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING payment_id",
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee).Scan(&paymentID)
	if err != nil {
		tx.Rollback()
		log.Fatal("payment data error")
		return
	}

	var itemsID []int
	for i := 0; i < len(order.Items); i++ {
		var itemID int
		err = tx.QueryRowx("INSERT INTO items (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING item_id",
			order.Items[i].ChartID, order.Items[i].TrackNumber, order.Items[i].Price, order.Items[i].Rid, order.Items[i].Name, order.Items[i].Sale, order.Items[i].Size,
			order.Items[i].TotalPrice, order.Items[i].NmID, order.Items[i].Brand, order.Items[i].Status).Scan(&itemID)
		if err != nil {
			tx.Rollback()
			log.Fatal("insert item error")
			return
		}
		itemsID = append(itemsID, itemID)
	}

	_, err = tx.Exec("INSERT INTO orders (order_uid, track_number, entry, delivery_id, payment_id, items, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)",
		order.OrderUID, order.TrackNumber, order.Entry, deliveryID, paymentID, pq.Array(&itemsID), order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		tx.Rollback()
		fmt.Println(err)
		log.Fatal("order data error")
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal("can't commit data to database")
		return
	}
}
