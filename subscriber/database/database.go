package database

import (
	"errors"
	"fmt"
	"log"
	"nats-streaming-subscriber/datastruct"
	"nats-streaming-subscriber/subscriber/memcache"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgressOrdersStorage struct {
	OrdersStorage *sqlx.DB
	cache         memcache.Cache
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

CREATE TABLE IF NOT EXISTS orders(
	order_id INT GENERATED ALWAYS AS IDENTITY,
	order_uid TEXT UNIQUE,
	track_number TEXT,
	entry TEXT NOT NULL,
	delivery_id INT REFERENCES deliveries ON DELETE CASCADE,
	payment_id INT REFERENCES payments ON DELETE CASCADE,
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

CREATE TABLE IF NOT EXISTS items(
	item_id INT GENERATED ALWAYS AS IDENTITY,
	order_id INT REFERENCES orders ON DELETE CASCADE,
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
	storage.cache = *memcache.NewCache(5*time.Minute, 10*time.Minute)
	storage.updateCache()
	return storage
}

func (p *PostgressOrdersStorage) AddNewOrder(order datastruct.Order) {
	tx, err := p.OrdersStorage.Beginx()
	if err != nil {
		log.Fatal("can't begin transaction")
		return
	}
	deliveryID := insertDelivery(tx, order)
	paymentID := insertPayment(tx, order)
	orderID := insertOrder(tx, order, deliveryID, paymentID)

	for i := 0; i < len(order.Items); i++ {
		insertItem(tx, order.Items[i], orderID)

	}

	err = tx.Commit()
	if err != nil {
		log.Println("can't commit data to database")
		return
	}
}

func insertDelivery(tx *sqlx.Tx, order datastruct.Order) int {
	var deliveryID int
	queryStr := `INSERT INTO deliveries (name, phone, zip, city, address, region, email)
	 			Values ($1, $2, $3, $4, $5, $6, $7) RETURNING delivery_id;`
	err := tx.QueryRowx(queryStr, order.Delivery.Name, order.Delivery.Phone,
		order.Delivery.Zip, order.Delivery.City, order.Delivery.Address,
		order.Delivery.Region, order.Delivery.Email).Scan(&deliveryID)
	if err != nil {
		tx.Rollback()
		log.Println("can't insert delivery data")
	}
	return deliveryID
}

func insertPayment(tx *sqlx.Tx, order datastruct.Order) int {
	var paymentID int
	queryStr := `INSERT INTO payments 
				(transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, good_total, custom_fee) 
				Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING payment_id;`
	err := tx.QueryRowx(queryStr, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee).Scan(&paymentID)
	if err != nil {
		tx.Rollback()
		log.Println("can't insert payment data")
	}
	return paymentID
}

func insertOrder(tx *sqlx.Tx, order datastruct.Order, deliveryID int, paymentID int) int {
	var orderID int
	queryStr := `INSERT INTO orders (order_uid, track_number, entry, delivery_id, payment_id, locale, 
				internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard) 
				Values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING order_id;`
	err := tx.QueryRowx(queryStr, order.OrderUID, order.TrackNumber, order.Entry, deliveryID, paymentID,
		order.Locale, order.InternalSignature, order.CustomerID, order.DeliveryService,
		order.Shardkey, order.SmID, order.DateCreated, order.OofShard).Scan(&orderID)
	if err != nil {
		tx.Rollback()
		log.Println("can't insert order: ", err)
	}
	return orderID
}

func insertItem(tx *sqlx.Tx, item datastruct.Item, orderID int) {
	queryStr := `INSERT INTO items (order_id, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);`
	_, err := tx.Exec(queryStr, orderID, item.ChartID, item.TrackNumber, item.Price,
		item.Rid, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
	if err != nil {
		tx.Rollback()
		log.Println("can't insert item: ", err)
	}
}

func (p *PostgressOrdersStorage) GetOrder(id string) (*datastruct.Order, error) {
	order, ok := p.cache.Get(id)
	if ok {
		return order, nil
	}
	order = &datastruct.Order{}
	var deliveryID int
	var paymentID int
	row := p.OrdersStorage.QueryRowx("SELECT order_uid, track_number, entry, delivery_id, payment_id, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard FROM orders WHERE order_id = $1", id)
	err := row.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &deliveryID, &paymentID, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if err != nil {
		return nil, errors.New("can't scan row")
	}

	var delivery datastruct.Delivery
	row = p.OrdersStorage.QueryRowx("SELECT name, phone, zip, city, address, region, email FROM deliveries WHERE delivery_id = $1", deliveryID)
	err = row.Scan(&delivery.Name, &delivery.Phone, &delivery.Zip, &delivery.City, &delivery.Address, &delivery.Region, &delivery.Email)
	if err != nil {
		log.Fatalf("can't scan row %v", err)
	}
	order.Delivery = delivery

	var payment datastruct.Payment
	row = p.OrdersStorage.QueryRowx("SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, good_total, custom_fee FROM payments WHERE payment_id = $1", paymentID)
	err = row.Scan(&payment.Transaction, &payment.RequestID, &payment.Currency, &payment.Provider, &payment.Amount, &payment.PaymentDT, &payment.Bank, &payment.DeliveryCost, &payment.GoodsTotal, &payment.CustomFee)
	if err != nil {
		log.Fatalf("can't scan row %v", err)
	}
	order.Payment = payment

	items := []datastruct.Item{}
	sql := `SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status FROM items WHERE order_id = $1`
	err = p.OrdersStorage.Select(&items, sql, id)
	if err != nil {
		fmt.Println(err)
	}

	order.Items = items
	p.cache.Set(id, *order, 5*time.Minute)
	return order, nil
}

func (p *PostgressOrdersStorage) updateCache() {
	rows, err := p.OrdersStorage.Query(`SELECT order_id FROM orders;`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var order_id int
	for rows.Next() {
		err := rows.Scan(&order_id)
		if err != nil {
			log.Fatal(err)
		}
		order, _ := p.GetOrder(strconv.Itoa(order_id))
		p.cache.Set(strconv.Itoa(order_id), *order, 5*time.Minute)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
