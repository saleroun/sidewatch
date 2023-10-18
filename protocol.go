package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-ping/ping"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var httpClient = &http.Client{
	Timeout: time.Duration(5 * int(time.Second)),
}

func HttpCheck(url string, timeout int) (*ServerStatus, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %s", err)
	}

	client := httpClient

	resp, err := client.Do(req)
	if err != nil {
		return &ServerStatus{URL: url, StatusCode: http.StatusRequestTimeout}, err
	}
	defer resp.Body.Close()

	return &ServerStatus{URL: url, StatusCode: resp.StatusCode}, nil
}
func Ping(addr string, count int, timeout int) (*ping.Statistics, error) {
	pinger, err := ping.NewPinger(addr)
	// pinger.SetPrivileged(true)
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	pinger.Count = count
	fmt.Printf("PING %s (%s):\n", pinger.Addr(), pinger.IPAddr())
	duration := time.Duration(timeout * int(time.Second))
	pinger.Timeout = duration
	err = pinger.Run()
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	fmt.Printf("Ping result %#v \n", pinger.Statistics())

	return pinger.Statistics(), nil
}
func isRabbitMQHealthy(rabbitMQURL string) float64 {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v\n", err)
		return 0
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v\n", err)
		return 0
	}
	defer ch.Close()

	// Perform a basic operation to check the health
	_, err = ch.QueueDeclare(
		"health_check_queue", // Name of the queue
		false,                // Durable v
		false,                // Delete when unused
		false,                // Exclusive
		false,                // No-wait
		nil,                  // Arguments
	)
	if err != nil {
		log.Printf("Failed to declare a queue: %v\n", err)
		return 0
	}

	return 1
}
func isMongoDBHealthy(mongoURI string, timeout int) float64 {
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Printf("Failed to create MongoDB client: %v\n", err)
		return 0
	}
	defer client.Disconnect(context.Background())

	Timeout := time.Duration(timeout * int(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Printf("Failed to ping MongoDB: %v\n", err)
		return 0
	}

	return 1
}
func isRedisHealthy(redisURI string, timeout int) float64 {
	u, err := url.Parse(redisURI)
	if err != nil {
		log.Errorln(err)
		return 0
	}

	Timeout := time.Duration(timeout * int(time.Second))
	pass, _ := u.User.Password()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     u.Host,
		Username: u.User.Username(),
		Password: pass,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to ping Redis: %v\n", err)
		return 0
	}

	// Close the Redis client after usage.
	if err := redisClient.Close(); err != nil {
		log.Printf("Failed to close Redis client: %v\n", err)
	}

	return 1
}
func isTDengineHealthy(tdURI string, timeout int) float64 {
	taos, err := sql.Open("taosRestful", tdURI)
	if err != nil {
		fmt.Println("failed to connect TDengine, err:", err)
		return 0
	}
	defer taos.Close()

	Timeout := time.Duration(timeout * int(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	err = taos.PingContext(ctx)
	if err != nil {
		log.Printf("Failed to connect to TDengine: %v\n", err)
		return 0
	}

	rows, err := taos.QueryContext(ctx, "SELECT 1")
	if err != nil {
		log.Printf("Failed to execute query: %v\n", err)
		return 0
	}
	defer rows.Close()

	return 1
}
