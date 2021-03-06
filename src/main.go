package main

import (
	"context"
	"encoding/json"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/application"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/configuration"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/distributionManager"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/foodorderingcontroller"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/foodorderingregister"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/item"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/ratingSystem"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/table"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/tableIdCounter"
	"github.com/vivk-FAF-PR16-2/RestaurantKitchen/src/waiter"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	ConfPath  = "./conf/configuration.json"
	ItemsPath = "./conf/items.json"
)

func main() {
	conf := GetConf()
	container := GetItemContainer()

	rate := ratingSystem.New()

	timeUnitMillisecondMultiplier := time.Duration(conf.TimeUnitMillisecondMultiplier)
	configuration.TimeUnit = time.Millisecond * timeUnitMillisecondMultiplier

	manager := tableIdCounter.New()

	foodorderingcontroller.Get().Setup(manager, &conf)
	foodorderingregister.Register(conf, *container)

	var tables = make([]*table.Table, conf.TableCount)
	for index := range tables {
		tables[index] = table.New(index, manager, container, &conf)
	}

	var waiters = make([]*waiter.Waiter, conf.WaiterCount)
	for index := range waiters {
		waiters[index] = waiter.New(index, &conf)
	}

	distributionManager.SetWaiters(waiters)

	for i, e := range tables {
		waiters[i%conf.WaiterCount].AddTable(e)
	}

	for index := range tables {
		tables[index].SetRatingSystem(rate)
		go tables[index].Run()
	}

	for index := range waiters {
		go waiters[index].Run()
	}

	isDone := make(chan os.Signal)
	signal.Notify(isDone, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	_, cancel := context.WithCancel(context.Background())

	mainApp := application.New(conf)
	go mainApp.Start()

	<-isDone
	cancel()
	mainApp.Shutdown()
}

func GetConf() configuration.Configuration {
	var conf configuration.Configuration

	confFile, _ := os.Open(ConfPath)
	defer func(confFile *os.File) {
		_ = confFile.Close()
	}(confFile)

	jsonData, err := io.ReadAll(confFile)
	if err != nil {
		log.Fatalf("exit: %s\n", err.Error())
		return conf
	}

	err = json.Unmarshal(jsonData, &conf)
	if err != nil {
		log.Fatalf("exit: %s\n", err.Error())
		return conf
	}

	return conf
}

func GetItemContainer() *item.Container {
	var itemList []item.Item

	itemListFile, _ := os.Open(ItemsPath)
	defer func(itemListFile *os.File) {
		_ = itemListFile.Close()
	}(itemListFile)

	jsonData, err := io.ReadAll(itemListFile)
	if err != nil {
		log.Fatalf("exit: %s\n", err.Error())
		return nil
	}

	err = json.Unmarshal(jsonData, &itemList)
	if err != nil {
		log.Fatalf("exit: %s\n", err.Error())
		return nil
	}

	return item.NewContainer(itemList)
}
