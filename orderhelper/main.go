package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"nats-streaming-subscriber/datastruct"
	"net/http"
	"strconv"

	"github.com/davecgh/go-spew/spew"
)

var url = "http://localhost:8080/"

func main() {
	for {
		id, err := getId()
		if err != nil {
			fmt.Println("необходимо указать целое, положительное число")
			continue
		}
		order, err := getOrder(url, id)
		if err != nil {
			fmt.Println(err)
			continue
		}
		spew.Dump(order)
	}
}

func getOrder(url string, id int) (*datastruct.Order, error) {
	requestUrl := url + "order/" + strconv.Itoa(id)
	resp, err := http.Get(requestUrl)
	if err != nil {
		return nil, errors.New("ошибка получения заказа")
	}
	switch resp.StatusCode {
	case http.StatusBadRequest:
		return nil, errors.New("неверный формат id")
	case http.StatusNotFound:
		return nil, errors.New("заказа по данному id не существует")
	case http.StatusInternalServerError:
		return nil, errors.New("ошибка получения заказа")
	}
	defer resp.Body.Close()

	bodyJson, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("ошибка получения заказа")
	}

	var order *datastruct.Order
	err = json.Unmarshal(bodyJson, &order)
	if err != nil {
		return nil, errors.New("ошибка получения заказа")
	}

	return order, nil
}

func getId() (int, error) {
	fmt.Println("введите id заказа")
	var id string
	fmt.Scanln(&id)
	orderId, err := strconv.Atoi(id)
	if err != nil {
		return 0, err
	}
	return orderId, nil
}
