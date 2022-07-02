package server

import (
	"encoding/json"
	"log"
	"nats-streaming-subscriber/datastruct"
	"nats-streaming-subscriber/subscriber/database"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

type Storage struct {
	OrdersStorage database.PostgressOrdersStorage
}

func StartServer(port string) {
	ordersDB := database.ConnectDB()
	storage := Storage{}
	storage.OrdersStorage = ordersDB

	router := chi.NewRouter()
	SetHandlers(storage, router)
	http.ListenAndServe(":"+port, router)
}

func SetHandlers(storage Storage, router *chi.Mux) {
	router.Get("/order/{id}", storage.GetOrder)
}

func (s *Storage) GetOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json charset=utf-8")
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 0, 64)
	if err != nil || id == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	order := datastruct.Order{}
	order, err = s.OrdersStorage.GetOrder(strconv.Itoa(int(id)))
	if err != nil {
		log.Fatal(err)
	}
	err = json.NewEncoder(w).Encode(order)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
